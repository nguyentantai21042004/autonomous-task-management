package sync

import "github.com/gin-gonic/gin"

// Handler defines the interface for the webhook sync handler.
type Handler interface {
	// HandleMemosWebhook processes incoming webhook payloads from Memos.
	HandleMemosWebhook(c *gin.Context)
}
