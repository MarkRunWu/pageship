package controller

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/oursky/pageship/internal/config"
	"github.com/oursky/pageship/internal/db"
	"github.com/oursky/pageship/internal/models"
)

type apiSite struct {
	*models.Site
	URL            string  `json:"url"`
	DeploymentName *string `json:"deploymentName"`
}

func (c *Controller) makeAPISite(app *models.App, site db.SiteInfo) *apiSite {
	sub := site.Name
	if site.Name == app.Config.DefaultSite {
		sub = ""
	}

	return &apiSite{
		Site: site.Site,
		URL: c.Config.HostPattern.MakeURL(
			c.Config.HostIDScheme.Make(site.AppID, sub),
		),
		DeploymentName: site.DeploymentName,
	}
}

func (c *Controller) handleSiteList(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "app-id")

	sites, err := tx(r.Context(), c.DB, func(conn db.Conn) ([]*apiSite, error) {
		app, err := conn.GetApp(r.Context(), appID)
		if err != nil {
			return nil, err
		}

		sites, err := conn.ListSitesInfo(r.Context(), appID)
		if err != nil {
			return nil, err
		}

		return mapModels(sites, func(site db.SiteInfo) *apiSite {
			return c.makeAPISite(app, site)
		}), nil
	})

	writeResponse(w, sites, err)
}

func (c *Controller) handleSiteCreate(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "app-id")

	var request struct {
		Name string `json:"name" binding:"required,dnsLabel"`
	}
	if !bindJSON(w, r, &request) {
		return
	}

	site, err := tx(r.Context(), c.DB, func(conn db.Conn) (*apiSite, error) {
		app, err := conn.GetApp(r.Context(), appID)
		if err != nil {
			return nil, err
		}

		if _, ok := app.Config.ResolveSite(request.Name); !ok {
			return nil, models.ErrUndefinedSite
		}

		site := models.NewSite(c.Clock.Now().UTC(), appID, request.Name)
		info, err := conn.CreateSiteIfNotExist(r.Context(), site)
		if err != nil {
			return nil, err
		}

		return c.makeAPISite(app, *info), nil
	})

	writeResponse(w, site, err)
}

func (c *Controller) updateDeploymentExpiry(
	ctx context.Context,
	conn db.Conn,
	now time.Time,
	conf *config.AppConfig,
	deployment *models.Deployment,
) error {
	sites, err := conn.GetDeploymentSiteNames(ctx, deployment)
	if err != nil {
		return err
	}

	if len(sites) == 0 && deployment.ExpireAt == nil {
		deploymentTTL, err := time.ParseDuration(conf.Deployments.TTL)
		if err != nil {
			return err
		}

		expireAt := now.Add(deploymentTTL)
		deployment.ExpireAt = &expireAt
		err = conn.SetDeploymentExpiry(ctx, deployment)
		if err != nil {
			return err
		}
	} else if len(sites) > 0 && deployment.ExpireAt != nil {
		deployment.ExpireAt = nil
		err = conn.SetDeploymentExpiry(ctx, deployment)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Controller) siteUpdateDeploymentName(
	ctx context.Context,
	conn db.Conn,
	now time.Time,
	conf *config.AppConfig,
	site *models.Site,
	deploymentName string,
) error {
	var currentDeployment *models.Deployment
	if site.DeploymentID != nil {
		d, err := conn.GetDeployment(ctx, site.AppID, *site.DeploymentID)
		if err != nil {
			return err
		}

		if d.Name == deploymentName {
			// Same deployment
			return nil
		}
		currentDeployment = d
	} else if deploymentName == "" {
		// Same deployment
		return nil
	}

	var newDeployment *models.Deployment
	if deploymentName != "" {
		d, err := conn.GetDeploymentByName(ctx, site.AppID, deploymentName)
		if err != nil {
			return err
		}

		if err := d.CheckAlive(now); err != nil {
			return err
		}

		err = conn.AssignDeploymentSite(ctx, d, site.ID)
		if err != nil {
			return err
		}
		site.DeploymentID = &d.ID
		newDeployment = d
	} else {
		err := conn.UnassignDeploymentSite(ctx, currentDeployment, site.ID)
		if err != nil {
			return err
		}
		site.DeploymentID = nil
		newDeployment = nil
	}

	if currentDeployment != nil {
		if err := c.updateDeploymentExpiry(ctx, conn, now, conf, currentDeployment); err != nil {
			return err
		}
	}
	if newDeployment != nil {
		if err := c.updateDeploymentExpiry(ctx, conn, now, conf, newDeployment); err != nil {
			return err
		}
	}

	return nil
}

func (c *Controller) handleSiteUpdate(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "app-id")
	siteName := chi.URLParam(r, "site-name")

	var request struct {
		DeploymentName *string `json:"deploymentName,omitempty" binding:"omitempty"`
	}
	if !bindJSON(w, r, &request) {
		return
	}

	now := c.Clock.Now().UTC()

	site, err := tx(r.Context(), c.DB, func(conn db.Conn) (*apiSite, error) {
		app, err := conn.GetApp(r.Context(), appID)
		if err != nil {
			return nil, err
		}

		site, err := conn.GetSiteByName(r.Context(), appID, siteName)
		if err != nil {
			return nil, err
		}

		if request.DeploymentName != nil {
			if err := c.siteUpdateDeploymentName(r.Context(), conn, now, app.Config, site, *request.DeploymentName); err != nil {
				return nil, err
			}
		}

		info, err := conn.GetSiteInfo(r.Context(), appID, site.ID)
		if err != nil {
			return nil, err
		}

		return c.makeAPISite(app, *info), nil
	})

	writeResponse(w, site, err)
}
