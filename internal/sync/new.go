package sync

import (
	"autonomous-task-management/internal/task/repository"
	pkgLog "autonomous-task-management/pkg/log"
)

type WebhookHandler struct {
	memosRepo  repository.MemosRepository
	vectorRepo repository.VectorRepository
	l          pkgLog.Logger
}

func NewWebhookHandler(memosRepo repository.MemosRepository, vectorRepo repository.VectorRepository, l pkgLog.Logger) *WebhookHandler {
	return &WebhookHandler{
		memosRepo:  memosRepo,
		vectorRepo: vectorRepo,
		l:          l,
	}
}
