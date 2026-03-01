package test

import (
	"autonomous-task-management/internal/agent"
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
	routerUC router.UseCase,
	agentUC agent.UseCase,
) Handler {
	return &handler{
		l:      l,
		router: routerUC,
		agent:  agentUC,
	}
}
