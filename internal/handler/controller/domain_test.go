package controller_test

import (
	"bytes"
	"net/http/httptest"
	"testing"

	_ "github.com/golang-migrate/migrate/v4/database/pgx"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite"
	// "github.com/oursky/pageship/internal/api"
	// "github.com/oursky/pageship/internal/config"
	_ "github.com/oursky/pageship/internal/db/postgres"
	_ "github.com/oursky/pageship/internal/db/sqlite"
	"github.com/oursky/pageship/internal/handler/controller"
	// "github.com/oursky/pageship/internal/models"
	"github.com/oursky/pageship/testutil"
	"github.com/stretchr/testify/assert"
)

var defaultConfig = controller.Config{
	TokenSigningKey: bytes.NewBufferString("test").Bytes(),
	TokenAuthority:  "test",
}

func TestDomainListDomains(t *testing.T) {
	testutil.WithTestController(func(c *testutil.TestController) {
		user, token := c.SigninUser("mock user")
		c.NewApp("test", user, nil)
		req := httptest.NewRequest("GET", "http://localtest.me/api/v1/apps/test/domains", nil)
		req.Header.Add("Authorization", "bearer "+token)
		w := httptest.NewRecorder()
		c.ServeHTTP(w, req)
		result := w.Result()
		assert.Equal(t, 200, result.StatusCode)
	})
}

// func TestDomainCreation(t *testing.T) {
// 	testutil.WithTestController(func(c *testutil.TestController) {
// 		user, token := c.SigninUser("mock user")
// 		c.NewApp("test", user, nil)
// 		req := httptest.NewRequest("POST", "http://localtest.me/api/v1/apps/test/domains/test-domain", nil)
// 		req.Header.Add("Authorization", "bearer "+token)
// 		w := httptest.NewRecorder()
// 		c.ServeHTTP(w, req)
// 		_, err := testutil.DecodeJSONResponse[*api.APIDomain](w.Result())
// 		if assert.Error(t, err) {
// 			err := err.(testutil.ServerError)
// 			assert.Equal(t, 400, err.Code)
// 			assert.Equal(t, models.ErrUndefinedDomain.Error(), err.Message)
// 		}
// 	})
// 	testutil.WithTestController(func(c *testutil.TestController) {
// 		user, token := c.SigninUser("mock user")
// 		c.NewApp("test", user, &config.AppConfig{
// 			Domains: []config.AppDomainConfig{
// 				{
// 					Site:   "main",
// 					Domain: "test.com",
// 				},
// 			},
// 			Team: []*config.AccessRule{
// 				{
//
// 					ACLSubjectRule: config.ACLSubjectRule{
// 						PageshipUser: user.ID,
// 					},
// 					Access: config.AccessLevelAdmin,
// 				},
// 			},
// 		})
// 		req := httptest.NewRequest("POST", "http://localtest.me/api/v1/apps/test/domains/test-domain", nil)
// 		req.Header.Add("Authorization", "bearer "+token)
// 		w := httptest.NewRecorder()
// 		c.ServeHTTP(w, req)
// 		domain, err := testutil.DecodeJSONResponse[*api.APIDomain](w.Result())
// 		if assert.NoError(t, err) {
// 			assert.Equal(t, "test.com", domain.Domain.Domain)
// 		}
// 	})
// }
