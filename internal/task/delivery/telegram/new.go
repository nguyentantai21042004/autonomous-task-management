package telegram

import (
	"github.com/gin-gonic/gin"

	"autonomous-task-management/internal/agent"
	"autonomous-task-management/internal/automation"
	"autonomous-task-management/internal/checklist"
	"autonomous-task-management/internal/router"
	"autonomous-task-management/internal/task"
	"autonomous-task-management/internal/task/repository"
	pkgLog "autonomous-task-management/pkg/log"
	pkgTelegram "autonomous-task-management/pkg/telegram"
)

// Handler is the interface for the Telegram delivery handler.
type Handler interface {
	HandleWebhook(c *gin.Context)
}

// New creates a new Telegram delivery handler.
func New(
	l pkgLog.Logger,
	uc task.UseCase,
	bot pkgTelegram.IBot,
	agentUC agent.UseCase,
	automationUC automation.UseCase,
	checklistUC checklist.UseCase,
	memosRepo repository.MemosRepository,
	routerUC router.UseCase,
) Handler {
	return &handler{
		l:            l,
		uc:           uc,
		bot:          bot,
		agent:        agentUC,
		automationUC: automationUC,
		checklistSvc: checklistUC,
		memosRepo:    memosRepo,
		router:       routerUC,
	}
}
