package controller_test

import (
	"bytes"
	"net/http/httptest"
	"testing"

	_ "github.com/golang-migrate/migrate/v4/database/pgx"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/oursky/pageship/internal/api"
	"github.com/oursky/pageship/internal/config"
	_ "github.com/oursky/pageship/internal/db/postgres"
	_ "github.com/oursky/pageship/internal/db/sqlite"
	"github.com/oursky/pageship/internal/handler/controller"
	"github.com/oursky/pageship/internal/models"
	"github.com/oursky/pageship/testutil"
	"github.com/stretchr/testify/assert"
)

var defaultConfig = controller.Config{
	TokenSigningKey: bytes.NewBufferString("test").Bytes(),
	TokenAuthority:  "test",
}

func TestDomainListDomains(t *testing.T) {
	t.Run("Should list all of domains", func(t *testing.T) {
		testutil.WithTestController(func(c *testutil.TestController) {
			user, token := c.SigninUser("mock user")
			c.NewApp("test", user, nil)
			req := httptest.NewRequest("GET", "http://localtest.me/api/v1/apps/test/domains", nil)
			req.Header.Add("Authorization", "bearer "+token)
			w := httptest.NewRecorder()
			c.ServeHTTP(w, req)
			domains, err := testutil.DecodeJSONResponse[[]*api.APIDomain](w.Result())
			if assert.NoError(t, err) {
				assert.Equal(t, 0, len(domains))
			}

		})
	})
}

func TestDomainCreation(t *testing.T) {
	t.Run("Should raise domain is undefined", func(t *testing.T) {
		testutil.WithTestController(func(c *testutil.TestController) {
			user, token := c.SigninUser("mock user")
			c.NewApp("test", user, nil)
			req := httptest.NewRequest("POST", "http://localtest.me/api/v1/apps/test/domains/test-domain", nil)
			req.Header.Add("Authorization", "bearer "+token)
			w := httptest.NewRecorder()
			c.ServeHTTP(w, req)
			_, err := testutil.DecodeJSONResponse[*api.APIDomain](w.Result())
			if assert.Error(t, err) {
				err := err.(testutil.ServerError)
				assert.Equal(t, 400, err.Code)
				assert.Equal(t, models.ErrUndefinedDomain.Error(), err.Message)
			}
		})
	})
	t.Run("Should create new domain", func(t *testing.T) {
		testutil.WithTestController(func(c *testutil.TestController) {
			user, token := c.SigninUser("mock user")
			c.NewApp("test", user, &config.AppConfig{
				Domains: []config.AppDomainConfig{
					{
						Site:   "main",
						Domain: "test.com",
					},
				},
				Team: []*config.AccessRule{
					{

						ACLSubjectRule: config.ACLSubjectRule{
							PageshipUser: user.ID,
						},
						Access: config.AccessLevelAdmin,
					},
				},
			})
			req := httptest.NewRequest("POST", "http://localtest.me/api/v1/apps/test/domains/test.com", nil)
			req.Header.Add("Authorization", "bearer "+token)
			w := httptest.NewRecorder()
			c.ServeHTTP(w, req)
			domain, err := testutil.DecodeJSONResponse[*api.APIDomain](w.Result())
			if assert.NoError(t, err) {
				assert.Equal(t, "test.com", domain.Domain.Domain)
			}
		})
	})
	t.Run("Should replace existed domain", func(t *testing.T) {
		testutil.WithTestController(func(c *testutil.TestController) {
            // TODO: create new domain with app A and replace it with app B
			user, token := c.SigninUser("mock user")
			c.NewApp("test", user, &config.AppConfig{
				Domains: []config.AppDomainConfig{
					{
						Site:   "main",
						Domain: "test.com",
					},
				},
				Team: []*config.AccessRule{
					{

						ACLSubjectRule: config.ACLSubjectRule{
							PageshipUser: user.ID,
						},
						Access: config.AccessLevelAdmin,
					},
				},
			})
		})
	})
	t.Run("Should get domain when it exists", func(t *testing.T) {
		testutil.WithTestController(func(c *testutil.TestController) {
			user, token := c.SigninUser("mock user")
			c.NewApp("test", user, &config.AppConfig{
				Domains: []config.AppDomainConfig{
					{
						Site:   "main",
						Domain: "test.com",
					},
				},
				Team: []*config.AccessRule{
					{

						ACLSubjectRule: config.ACLSubjectRule{
							PageshipUser: user.ID,
						},
						Access: config.AccessLevelAdmin,
					},
				},
			})
			req := httptest.NewRequest("POST", "http://localtest.me/api/v1/apps/test/domains/test.com", nil)
			req.Header.Add("Authorization", "bearer "+token)
			w := httptest.NewRecorder()
			c.ServeHTTP(w, req)
			domain, err := testutil.DecodeJSONResponse[*api.APIDomain](w.Result())
			if assert.NoError(t, err) {
				assert.Equal(t, "test.com", domain.Domain.Domain)
			}
			w = httptest.NewRecorder()
			c.ServeHTTP(w, req)
			domain, err = testutil.DecodeJSONResponse[*api.APIDomain](w.Result())
			if assert.NoError(t, err) {
				assert.Equal(t, "test.com", domain.Domain.Domain)
			}
		})
	})
	t.Run("Should raise domain verificaiton not supported", func(t *testing.T) {
		testutil.WithTestController(func(c *testutil.TestController) {
			user, token := c.SigninUser("mock user")
			c.NewApp("test", user, &config.AppConfig{
				Domains: []config.AppDomainConfig{
					{
						Site:               "main",
						Domain:             "test.com",
						DomainVerification: true,
					},
				},
				Team: []*config.AccessRule{
					{

						ACLSubjectRule: config.ACLSubjectRule{
							PageshipUser: user.ID,
						},
						Access: config.AccessLevelAdmin,
					},
				},
			})
			req := httptest.NewRequest("POST", "http://localtest.me/api/v1/apps/test/domains/test.com", nil)
			req.Header.Add("Authorization", "bearer "+token)
			w := httptest.NewRecorder()
			c.ServeHTTP(w, req)
			_, err := testutil.DecodeJSONResponse[*api.APIDomain](w.Result())
			if assert.Error(t, err) {
				assert.Equal(t, 400, err.(testutil.ServerError).Code)
				assert.Equal(t, models.ErrDomainVerificationNotSupported.Error(), err.(testutil.ServerError).Message)
			}
		})
	})
}

func TestDomainDeletion(t *testing.T) {
	t.Run("Should raise domain not defined", func(t *testing.T) {
		testutil.WithTestController(func(c *testutil.TestController) {
			user, token := c.SigninUser("mock user")
			c.NewApp("test", user, &config.AppConfig{
				Domains: []config.AppDomainConfig{
					{
						Site:   "main",
						Domain: "test.com",
					},
				},
				Team: []*config.AccessRule{
					{

						ACLSubjectRule: config.ACLSubjectRule{
							PageshipUser: user.ID,
						},
						Access: config.AccessLevelAdmin,
					},
				},
			})
			req := httptest.NewRequest("DELETE", "http://localtest.me/api/v1/apps/test/domains/test.com", nil)
			req.Header.Add("Authorization", "bearer "+token)
			w := httptest.NewRecorder()
			c.ServeHTTP(w, req)
			_, err := testutil.DecodeJSONResponse[*api.APIDomain](w.Result())
			if assert.Error(t, err) {
				assert.Equal(t, 404, err.(testutil.ServerError).Code)
				assert.Equal(t, models.ErrDomainNotFound.Error(), err.(testutil.ServerError).Message)
			}
		})
	})
	t.Run("Should delete domain", func(t *testing.T) {
		testutil.WithTestController(func(c *testutil.TestController) {
			user, token := c.SigninUser("mock user")
			c.NewApp("test", user, &config.AppConfig{
				Domains: []config.AppDomainConfig{
					{
						Site:   "main",
						Domain: "test.com",
					},
				},
				Team: []*config.AccessRule{
					{

						ACLSubjectRule: config.ACLSubjectRule{
							PageshipUser: user.ID,
						},
						Access: config.AccessLevelAdmin,
					},
				},
			})
			req := httptest.NewRequest("POST", "http://localtest.me/api/v1/apps/test/domains/test.com", nil)
			req.Header.Add("Authorization", "bearer "+token)
			w := httptest.NewRecorder()
			c.ServeHTTP(w, req)
			domain, err := testutil.DecodeJSONResponse[*api.APIDomain](w.Result())
			if assert.NoError(t, err) {
				assert.Equal(t, "test.com", domain.Domain.Domain)
			}

			req = httptest.NewRequest("DELETE", "http://localtest.me/api/v1/apps/test/domains/test.com", nil)
			req.Header.Add("Authorization", "bearer "+token)
			w = httptest.NewRecorder()
			c.ServeHTTP(w, req)
			deleteResult := w.Result()
			assert.Equal(t, 200, deleteResult.StatusCode)
			_, err = c.DB.GetDomainByName(c.Context, "test.com")
			if assert.Error(t, err) {
				assert.ErrorIs(t, models.ErrDomainNotFound, err)
			}
		})
	})
}

func TestDomainActivation(t *testing.T) {
	t.Run("Should raise domain is undefined", func(t *testing.T) {
		testutil.WithTestController(func(c *testutil.TestController) {
			user, token := c.SigninUser("mock user")
			c.NewApp("test", user, &config.AppConfig{
				Domains: []config.AppDomainConfig{
					{
						Site:   "main",
						Domain: "test.com",
					},
				},
				Team: []*config.AccessRule{
					{

						ACLSubjectRule: config.ACLSubjectRule{
							PageshipUser: user.ID,
						},
						Access: config.AccessLevelAdmin,
					},
				},
			})

            req := httptest.NewRequest("POST", "http://localtest.me/api/v1/apps/test/domains/test.com/activate", nil)
			req.Header.Add("Authorization", "bearer "+token)
			w = httptest.NewRecorder()
			c.ServeHTTP(w, req)
        })
    })
	t.Run("Should activate domain", func(t *testing.T) {
    })
	t.Run("Should get activation", func(t *testing.T) {
    })
}
