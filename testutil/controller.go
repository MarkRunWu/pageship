package testutil

import (
	"context"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/oursky/pageship/internal/config"
	"github.com/oursky/pageship/internal/db"
	"github.com/oursky/pageship/internal/handler/controller"
	"github.com/oursky/pageship/internal/models"
	"github.com/oursky/pageship/internal/storage"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type TestController struct {
	Context    context.Context
	DB         db.DB
	Storage    storage.Storage
	controller *controller.Controller
}

func (c *TestController) SigninUser(name string) (user *models.User, token string) {
	now := time.Now()
	user = models.NewUser(now, name)
	err := db.WithTx(c.Context, c.DB, func(tx db.Tx) error {
		context := c.Context
		return tx.CreateUser(context, user)
	})
	if err != nil {
		panic(err)
	}

	claims := models.NewTokenClaims(models.TokenSubjectUser(user.ID), user.Name)

	config := c.controller.Config
	claims.Issuer = config.TokenAuthority
	claims.Audience = jwt.ClaimStrings{config.TokenAuthority}
	claims.IssuedAt = jwt.NewNumericDate(now)
	claims.ExpiresAt = jwt.NewNumericDate(now.Add(100 * time.Minute))
	token, err = jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(config.TokenSigningKey)
	if err != nil {
		panic(err)
	}
	return
}

func (c *TestController) NewApp(name string, user *models.User, config *config.AppConfig) (id string) {
	db.WithTx(c.Context, c.DB, func(tx db.Tx) error {
		now := time.Now()
		context := c.Context
		app := models.NewApp(now, name, user.ID)
		app.Config = config
		return tx.CreateApp(context, app)
	})
	return name
}

func (c *TestController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.controller.Handler().ServeHTTP(w, r)
}

func WithTestController(f func(*TestController)) {
	LoadTestEnvs()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	storage, err := storage.New(ctx, viper.GetString("storage-url"))
	if err != nil {
		panic(err)
	}
	db_url := viper.GetString("database-url")
	database, err := db.New(db_url)
	if err != nil {
		panic(err)
	}
	err = doMigrate(db_url, false)
	defer doMigrate(db_url, true)
	if err != nil {
		panic(err)
	}
	logger, _ := zap.NewDevelopmentConfig().Build()
	hostPattern := viper.GetString("host-pattern")
	storageKeyPrefix := viper.GetString("key-prefix")
	maxDeploymentSize := viper.GetInt("deployment-size")
	defaultConfig := controller.Config{
		MaxDeploymentSize: int64(maxDeploymentSize),
		StorageKeyPrefix:  storageKeyPrefix,
		HostPattern:       config.NewHostPattern(hostPattern),
		HostIDScheme:      config.HostIDSchemeDefault,
		TokenSigningKey:   []byte("test"),
		TokenAuthority:    "test",
		ServerVersion:     "test",
	}
	ctrl := &controller.Controller{
		Context: ctx,
		Config:  defaultConfig,
		Storage: storage,
		DB:      database,
		Logger:  logger,
	}
	f(&TestController{
		DB:         database,
		Storage:    *storage,
		Context:    ctx,
		controller: ctrl,
	})
}
