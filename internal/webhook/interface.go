package webhook

import (
	"autonomous-task-management/internal/model"
	"context"

	"github.com/gin-gonic/gin"
)

// UseCase defines the business logic interface for the webhook domain.
type UseCase interface {
	ParseGitHubEvent(ctx context.Context, payload []byte, eventType string, signature string) (*model.WebhookEvent, error)
	ParseGitLabEvent(ctx context.Context, payload []byte, eventType string, token string) (*model.WebhookEvent, error)
}

// Handler defines the interface for the webhook delivery handler.
type Handler interface {
	HandleGitHubWebhook(c *gin.Context)
	HandleGitLabWebhook(c *gin.Context)
}
