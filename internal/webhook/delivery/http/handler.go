package http

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"autonomous-task-management/internal/automation"
	"autonomous-task-management/internal/model"
	"autonomous-task-management/internal/webhook"
	pkgLog "autonomous-task-management/pkg/log"
	pkgResponse "autonomous-task-management/pkg/response"
)

type handler struct {
	uc           webhook.UseCase
	automationUC automation.UseCase
	l            pkgLog.Logger
}

func NewHandler(uc webhook.UseCase, automationUC automation.UseCase, l pkgLog.Logger) webhook.Handler {
	return &handler{
		uc:           uc,
		automationUC: automationUC,
		l:            l,
	}
}

func (h *handler) HandleGitHubWebhook(c *gin.Context) {
	ctx := c.Request.Context()

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.l.Errorf(ctx, "Failed to read webhook body: %v", err)
		pkgResponse.Error(c, err, nil)
		return
	}

	signature := c.GetHeader("X-Hub-Signature-256")
	eventType := c.GetHeader("X-GitHub-Event")

	event, err := h.uc.ParseGitHubEvent(ctx, body, eventType, signature)
	if err != nil {
		h.l.Errorf(ctx, "GitHub webhook error: %v", err)
		if err == webhook.ErrInvalidSignature {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
			return
		}
		if err == webhook.ErrRateLimitExceeded {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		pkgResponse.Error(c, err, nil)
		return
	}

	// Process in background
	go h.processWebhookAsync(*event)

	pkgResponse.OK(c, gin.H{"status": "accepted"})
}

func (h *handler) HandleGitLabWebhook(c *gin.Context) {
	ctx := c.Request.Context()

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.l.Errorf(ctx, "Failed to read webhook body: %v", err)
		pkgResponse.Error(c, err, nil)
		return
	}

	token := c.GetHeader("X-Gitlab-Token")
	eventType := c.GetHeader("X-Gitlab-Event")

	event, err := h.uc.ParseGitLabEvent(ctx, body, eventType, token)
	if err != nil {
		h.l.Errorf(ctx, "GitLab webhook error: %v", err)
		if err == webhook.ErrInvalidSignature {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		if err == webhook.ErrRateLimitExceeded {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		pkgResponse.Error(c, err, nil)
		return
	}

	// Process in background
	go h.processWebhookAsync(*event)

	pkgResponse.OK(c, gin.H{"status": "accepted"})
}

func (h *handler) processWebhookAsync(event model.WebhookEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	h.l.Infof(ctx, "Processing webhook async: %s/%s from %s", event.EventType, event.Action, event.Repository)

	sc := model.Scope{UserID: "system_webhook"}
	output, err := h.automationUC.ProcessWebhook(ctx, sc, automation.ProcessWebhookInput{
		Event: event,
	})

	if err != nil {
		h.l.Errorf(ctx, "Webhook processing failed: %v", err)
		return
	}

	h.l.Infof(ctx, "Webhook processed: %s", output.Message)
}
