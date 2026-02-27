package telegram

import (
	"github.com/gin-gonic/gin"

	"autonomous-task-management/internal/agent/orchestrator"
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
	bot *pkgTelegram.Bot,
	orch *orchestrator.Orchestrator,
	automationUC automation.UseCase,
	checklistSvc checklist.Service,
	memosRepo repository.MemosRepository,
	router router.Router, // 🆕 Use interface for better testability
) Handler {
	return &handler{
		l:            l,
		uc:           uc,
		bot:          bot,
		orchestrator: orch,
		automationUC: automationUC,
		checklistSvc: checklistSvc,
		memosRepo:    memosRepo,
		router:       router, // 🆕 Inject router
	}
}
