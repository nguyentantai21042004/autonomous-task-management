package middleware

import (
	"autonomous-task-management/pkg/response"

	"github.com/gin-gonic/gin"
)

// ServiceAuth validates X-Service-Key header for internal service-to-service authentication
func (m Middleware) ServiceAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader("X-Service-Key")
		if raw == "" {
			response.Unauthorized(c)
			c.Abort()
			return
		}
		decrypted, err := m.encrypter.Decrypt(raw)
		if err != nil {
			m.l.Errorf(c.Request.Context(), "ServiceAuth: decrypt failed: %v", err)
			response.Unauthorized(c)
			c.Abort()
			return
		}
		if m.internalKey == "" || decrypted != m.internalKey {
			response.Unauthorized(c)
			c.Abort()
			return
		}
		c.Set("auth_type", "internal")
		c.Next()
	}
}
