package sync

import (
	"context"

	"github.com/gin-gonic/gin"
)

// UseCase defines the business logic interface for syncing Memos with Qdrant.
type UseCase interface {
	SyncTask(ctx context.Context, memoID string) error
	DeleteTask(ctx context.Context, memoID string) error
}

// Handler defines the interface for the webhook sync handler.
type Handler interface {
	// HandleMemosWebhook processes incoming webhook payloads from Memos.
	HandleMemosWebhook(c *gin.Context)
}
