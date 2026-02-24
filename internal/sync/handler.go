package sync

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"

	pkgResponse "autonomous-task-management/pkg/response"
)

// HandleMemosWebhook processes Memos webhook events.
func (h *WebhookHandler) HandleMemosWebhook(c *gin.Context) {
	ctx := c.Request.Context()

	var payload MemosWebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		h.l.Errorf(ctx, "webhook: failed to parse payload: %v", err)
		pkgResponse.Error(c, err, nil)
		return
	}

	h.l.Infof(ctx, "webhook: received %s for memo %s", payload.ActivityType, payload.Memo.UID)

	// Process in background to avoid blocking Memos
	go func(p MemosWebhookPayload) {
		// CRITICAL: Add timeout to prevent goroutine leak
		bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		memoID := p.Memo.UID

		switch p.ActivityType {
		case "memos.memo.created", "memos.memo.updated":
			// Re-embed task (upsert)
			h.syncWithRetry(bgCtx, memoID)

		case "memos.memo.deleted":
			// Delete from Qdrant
			if err := h.vectorRepo.DeleteTask(bgCtx, memoID); err != nil {
				h.l.Errorf(bgCtx, "webhook: failed to delete task %s: %v", memoID, err)
			} else {
				h.l.Infof(bgCtx, "webhook: deleted task %s from Qdrant", memoID)
			}
		}
	}(payload)

	// Acknowledge immediately
	pkgResponse.OK(c, map[string]string{"status": "accepted"})
}

// syncWithRetry embeds task to Qdrant with exponential backoff.
func (h *WebhookHandler) syncWithRetry(ctx context.Context, memoID string) {
	maxRetries := 3
	backoff := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		// Fetch task from Memos
		task, err := h.memosRepo.GetTask(ctx, memoID)
		if err != nil {
			h.l.Warnf(ctx, "webhook: fetch memo failed (retry %d/%d): %v", i+1, maxRetries, err)
			time.Sleep(backoff)
			backoff *= 2
			continue
		}

		// Embed to Qdrant
		if err := h.vectorRepo.EmbedTask(ctx, task); err != nil {
			h.l.Warnf(ctx, "webhook: embed failed (retry %d/%d): %v", i+1, maxRetries, err)
			time.Sleep(backoff)
			backoff *= 2
			continue
		}

		// Success
		h.l.Infof(ctx, "webhook: successfully synced task %s to Qdrant", memoID)
		return
	}

	// All retries failed
	h.l.Errorf(ctx, "webhook: FAILED to sync task %s after %d retries. Data drift occurred!", memoID, maxRetries)
}
