package http

import (
	"autonomous-task-management/internal/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes maps HTTP verbs and paths to Handler methods.
// All routes are protected by the Auth middleware by convention.
// Adjust middleware as needed per route.
func RegisterRoutes(rg *gin.RouterGroup, h *handler, mw middleware.Middleware) {
	items := rg.Group("/items")
	{
		items.POST("", mw.Auth(), h.Create)
		items.GET("", mw.Auth(), h.List)
		items.GET("/:id", mw.Auth(), h.Detail)
		items.PUT("/:id", mw.Auth(), h.Update)
		items.DELETE("/:id", mw.Auth(), h.Delete)
	}
}
