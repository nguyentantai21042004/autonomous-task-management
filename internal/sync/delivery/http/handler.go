package http

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"

	"autonomous-task-management/internal/sync"
	pkgLog "autonomous-task-management/pkg/log"
	pkgResponse "autonomous-task-management/pkg/response"
)

type handler struct {
	uc sync.UseCase
	l  pkgLog.Logger
}

func NewHandler(uc sync.UseCase, l pkgLog.Logger) sync.Handler {
	return &handler{
		uc: uc,
		l:  l,
	}
}

func (h *handler) HandleMemosWebhook(c *gin.Context) {
	ctx := c.Request.Context()

	var payload sync.MemosWebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		h.l.Errorf(ctx, "webhook: failed to parse payload: %v", err)
		pkgResponse.Error(c, err, nil)
		return
	}

	h.l.Infof(ctx, "webhook: received %s for memo %s", payload.ActivityType, payload.Memo.UID)

	// Process in background to avoid blocking Memos
	go func(p sync.MemosWebhookPayload) {
		bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		memoID := p.Memo.UID

		switch p.ActivityType {
		case "memos.memo.created", "memos.memo.updated":
			if err := h.uc.SyncTask(bgCtx, memoID); err != nil {
				h.l.Errorf(bgCtx, "webhook: FAILED to sync task %s: %v", memoID, err)
			}

		case "memos.memo.deleted":
			if err := h.uc.DeleteTask(bgCtx, memoID); err != nil {
				h.l.Errorf(bgCtx, "webhook: failed to delete task %s: %v", memoID, err)
			}
		}
	}(payload)

	// Acknowledge immediately
	pkgResponse.OK(c, map[string]string{"status": "accepted"})
}
