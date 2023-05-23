package controller

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/dustin/go-humanize"
	"github.com/gin-gonic/gin"
	"github.com/oursky/pageship/internal/config"
	"github.com/oursky/pageship/internal/db"
	"github.com/oursky/pageship/internal/deploy"
	"github.com/oursky/pageship/internal/models"
)

type apiDeployment struct {
	*models.Deployment
	Sites []string `json:"sites"`
}

func (c *Controller) makeAPIDeployment(d *models.Deployment) apiDeployment {
	deployment := *d
	deployment.Metadata.Files = nil // Avoid large file list

	return apiDeployment{Deployment: d}
}

func (c *Controller) handleDeploymentGet(ctx *gin.Context) {
	appID := ctx.Param("app-id")
	deploymentName := ctx.Param("deployment-name")

	err := db.WithTx(ctx, c.DB, func(conn db.Conn) error {
		deployment, err := conn.GetDeployment(ctx, appID, deploymentName)
		if err != nil {
			return err
		}

		ctx.JSON(http.StatusOK, response{Result: c.makeAPIDeployment(deployment)})
		return nil
	})

	if err != nil {
		if errors.Is(err, models.ErrDeploymentNotFound) {
			ctx.JSON(http.StatusNotFound, response{Error: err})
		} else {
			ctx.AbortWithError(http.StatusInternalServerError, err)
		}
	}
}

func (c *Controller) handleDeploymentCreate(ctx *gin.Context) {
	appID := ctx.Param("app-id")

	var request struct {
		Name       string             `json:"name" binding:"required,dnsLabel"`
		Files      []models.FileEntry `json:"files" binding:"required"`
		SiteConfig *config.SiteConfig `json:"site_config" binding:"required"`
	}
	if err := checkBind(ctx, ctx.ShouldBindJSON(&request)); err != nil {
		return
	}
	name := request.Name
	files := request.Files
	siteConfig := request.SiteConfig

	if len(files) > models.MaxFiles {
		ctx.JSON(http.StatusBadRequest, response{Error: deploy.ErrTooManyFiles})
		return
	}

	if err := config.ValidateSiteConfig(siteConfig); err != nil {
		ctx.JSON(http.StatusBadRequest, response{Error: err})
		return
	}

	var totalSize int64 = 0
	for _, entry := range files {
		totalSize += entry.Size
	}
	if totalSize > c.Config.MaxDeploymentSize {
		ctx.JSON(http.StatusBadRequest, response{
			Error: fmt.Errorf(
				"deployment too large: %s > %s",
				humanize.Bytes(uint64(totalSize)),
				humanize.Bytes(uint64(c.Config.MaxDeploymentSize)),
			),
		})
		return
	}

	err := db.WithTx(ctx, c.DB, func(conn db.Conn) error {
		_, err := conn.GetApp(ctx, appID)
		if errors.Is(err, models.ErrAppNotFound) {
			ctx.JSON(http.StatusNotFound, response{Error: err})
			return db.ErrRollback
		} else if err != nil {
			return err
		}

		metadata := &models.DeploymentMetadata{
			Files:  files,
			Config: *siteConfig,
		}
		deployment := models.NewDeployment(c.Clock.Now().UTC(), name, appID, c.Config.StorageKeyPrefix, metadata)

		err = conn.CreateDeployment(ctx, deployment)
		if errors.Is(err, models.ErrDeploymentUsedName) {
			ctx.JSON(http.StatusBadRequest, response{Error: err})
			return db.ErrRollback
		} else if err != nil {
			return err
		}

		ctx.JSON(http.StatusOK, response{Result: c.makeAPIDeployment(deployment)})
		return nil
	})

	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
	}
}

func (c *Controller) handleDeploymentUpload(ctx *gin.Context) {
	appID := ctx.Param("app-id")
	deploymentName := ctx.Param("deployment-name")

	var deployment *models.Deployment
	err := db.WithTx(ctx, c.DB, func(conn db.Conn) (err error) {
		deployment, err = conn.GetDeploymentByName(ctx, appID, deploymentName)
		if err != nil {
			return
		}

		if deployment.UploadedAt != nil {
			err = models.ErrDeploymentAlreadyUploaded
			return
		}

		return
	})

	if err != nil {
		if errors.Is(err, models.ErrDeploymentNotFound) {
			ctx.JSON(http.StatusNotFound, response{Error: err})
		} else if errors.Is(err, models.ErrDeploymentAlreadyUploaded) {
			ctx.JSON(http.StatusConflict, response{Error: err})
		} else {
			ctx.AbortWithError(http.StatusInternalServerError, err)
		}
		return
	}

	// Extract tarball to object stoarge

	handleFile := func(e models.FileEntry, r io.Reader) error {
		key := deployment.StorageKeyPrefix + e.Path
		return c.Storage.Upload(ctx, key, r)
	}

	err = deploy.ExtractFiles(ctx.Request.Body, deployment.Metadata.Files, handleFile)
	if errors.As(err, new(deploy.Error)) {
		ctx.JSON(http.StatusBadRequest, response{Error: err})
		return
	} else if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	now := c.Clock.Now().UTC()

	// Mark deployment as completed, but inactive
	err = db.WithTx(ctx, c.DB, func(conn db.Conn) error {
		deployment, err = conn.GetDeployment(ctx, appID, deployment.ID)
		if err != nil {
			return err
		}

		if deployment.UploadedAt != nil {
			return models.ErrDeploymentAlreadyUploaded
		}

		err = conn.MarkDeploymentUploaded(ctx, now, deployment)
		if err != nil {
			return err
		}

		ctx.JSON(http.StatusOK, response{Result: c.makeAPIDeployment(deployment)})
		return nil
	})

	if err != nil {
		if errors.Is(err, models.ErrDeploymentNotFound) {
			ctx.JSON(http.StatusNotFound, response{Error: err})
		} else if errors.Is(err, models.ErrDeploymentAlreadyUploaded) {
			ctx.JSON(http.StatusConflict, response{Error: err})
		} else {
			ctx.AbortWithError(http.StatusInternalServerError, err)
		}
	}
}

func (c *Controller) handleDeploymentList(ctx *gin.Context) {
	appID := ctx.Param("app-id")

	err := db.WithTx(ctx, c.DB, func(conn db.Conn) error {
		deployments, err := conn.ListDeployments(ctx, appID)
		if err != nil {
			return err
		}

		result := mapModels(deployments, c.makeAPIDeployment)

		ctx.JSON(http.StatusOK, response{Result: result})
		return nil
	})

	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
	}
}
