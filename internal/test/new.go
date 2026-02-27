package test

import (
	"autonomous-task-management/internal/agent/orchestrator"
	"autonomous-task-management/internal/router"
	pkgLog "autonomous-task-management/pkg/log"

	"github.com/gin-gonic/gin"
)

// Handler is the interface for the test handler
type Handler interface {
	HandleTestMessage(c *gin.Context)
	HandleResetSession(c *gin.Context)
	HandleHealthCheck(c *gin.Context)
}

// New creates a new test handler
func New(
	l pkgLog.Logger,
	router router.Router,
	orchestrator *orchestrator.Orchestrator,
) Handler {
	return &handler{
		l:            l,
		router:       router,
		orchestrator: orchestrator,
	}
}
