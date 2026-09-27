package middleware

import (
	"net/http"
	"strings"

	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// DemoDeleteDisabledCode tells the client why the demo admin's delete failed.
const DemoDeleteDisabledCode = "DEMO_DELETE_DISABLED"

// DemoAdminNoDelete refuses every DELETE from the public demo admin. Its
// password is printed on the login page, and one visitor emptying a list
// would leave it empty for everyone else until the next rewind. It runs in
// the global chain, so it reads the token itself.
//
// A user's own sessions are exempt: ending one signs a device out and
// deletes no data.
func DemoAdminNoDelete() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodDelete || strings.HasPrefix(c.Request.URL.Path, "/api/auth/sessions") {
			c.Next()
			return
		}
		claims := requestClaims(c)
		if claims == nil || !claims.Demo || claims.Role != "admin" {
			c.Next()
			return
		}
		if logger.Log != nil {
			logger.Info("demo admin delete refused", zap.String("path", c.Request.URL.Path))
		}
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Demo yönetici hesabıyla silme işlemi yapılamaz.",
			"code":  DemoDeleteDisabledCode,
		})
		c.Abort()
	}
}
