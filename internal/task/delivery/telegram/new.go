package telegram

import (
	"github.com/gin-gonic/gin"

	"autonomous-task-management/internal/task"
	pkgLog "autonomous-task-management/pkg/log"
	pkgTelegram "autonomous-task-management/pkg/telegram"
)

// Handler is the interface for the Telegram delivery handler.
type Handler interface {
	HandleWebhook(c *gin.Context)
}

// New creates a new Telegram delivery handler.
func New(l pkgLog.Logger, uc task.UseCase, bot *pkgTelegram.Bot) Handler {
	return &handler{
		l:   l,
		uc:  uc,
		bot: bot,
	}
}
