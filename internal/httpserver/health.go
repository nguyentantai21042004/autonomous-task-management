package httpserver

import (
	"autonomous-task-management/pkg/response"

	"github.com/gin-gonic/gin"
)

// Health response constants (single source for version and service identity).
const (
	HealthMessage = "From Smap API V1 With Love"
	HealthVersion = "1.0.0"
	ServiceName   = "autonomous-task-management"
)

// healthCheck handles health check requests
// @Summary Health Check
// @Description Check if the API is healthy
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "API is healthy"
// @Router /health [get]
func (srv HTTPServer) healthCheck(c *gin.Context) {
	response.OK(c, gin.H{
		"status":  "healthy",
		"message": HealthMessage,
		"version": HealthVersion,
		"service": ServiceName,
	})
}

// readyCheck handles readiness check requests (Postgres + Redis).
// @Summary Readiness Check
// @Description Check if the API is ready to serve traffic
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "API is ready"
// @Router /ready [get]
func (srv HTTPServer) readyCheck(c *gin.Context) {
	ctx := c.Request.Context()
	if err := srv.postgresDB.PingContext(ctx); err != nil {
		c.JSON(503, gin.H{
			"status":  "not ready",
			"message": "Database connection failed",
			"error":   err.Error(),
		})
		return
	}
	if err := srv.redisClient.Ping(ctx); err != nil {
		c.JSON(503, gin.H{
			"status":  "not ready",
			"message": "Redis connection failed",
			"error":   err.Error(),
		})
		return
	}
	response.OK(c, gin.H{
		"status":   "ready",
		"message":  HealthMessage,
		"version":  HealthVersion,
		"service":  ServiceName,
		"database": "connected",
		"redis":    "connected",
	})
}

// liveCheck handles liveness check requests
// @Summary Liveness Check
// @Description Check if the API is alive
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "API is alive"
// @Router /live [get]
func (srv HTTPServer) liveCheck(c *gin.Context) {
	response.OK(c, gin.H{
		"status":  "alive",
		"message": HealthMessage,
		"version": HealthVersion,
		"service": ServiceName,
	})
}
