package webhook

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"autonomous-task-management/internal/automation"
	"autonomous-task-management/internal/model"
	pkgResponse "autonomous-task-management/pkg/response"
)

// HandleGitHubWebhook processes GitHub webhook events
func (h *Handler) HandleGitHubWebhook(c *gin.Context) {
	ctx := c.Request.Context()

	// Read body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.l.Errorf(ctx, "Failed to read webhook body: %v", err)
		pkgResponse.Error(c, err, nil)
		return
	}

	// Verify signature
	signature := c.GetHeader("X-Hub-Signature-256")
	if err := h.security.ValidateGitHubSignature(body, signature); err != nil {
		h.l.Errorf(ctx, "GitHub signature verification failed: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	// Check rate limit
	if err := h.security.CheckRateLimit("github"); err != nil {
		h.l.Warnf(ctx, "Rate limit exceeded: %v", err)
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
		return
	}

	// Get event type
	eventType := c.GetHeader("X-GitHub-Event")

	// Parse event
	var event *model.WebhookEvent
	switch eventType {
	case "push":
		event, err = h.githubParser.ParsePushEvent(body)
	case "pull_request":
		event, err = h.githubParser.ParsePullRequestEvent(body)
	case "issues":
		event, err = h.githubParser.ParseIssueEvent(body)
	default:
		h.l.Infof(ctx, "Unsupported GitHub event type: %s", eventType)
		pkgResponse.OK(c, gin.H{"status": "ignored", "reason": "unsupported event type"})
		return
	}

	if err != nil {
		h.l.Errorf(ctx, "Failed to parse GitHub event: %v", err)
		pkgResponse.Error(c, err, nil)
		return
	}

	// Process in background
	go h.processWebhookAsync(*event)

	// Acknowledge immediately
	pkgResponse.OK(c, gin.H{"status": "accepted"})
}

// HandleGitLabWebhook processes GitLab webhook events
func (h *Handler) HandleGitLabWebhook(c *gin.Context) {
	ctx := c.Request.Context()

	// Read body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.l.Errorf(ctx, "Failed to read webhook body: %v", err)
		pkgResponse.Error(c, err, nil)
		return
	}

	// Verify token
	token := c.GetHeader("X-Gitlab-Token")
	if err := h.security.ValidateGitLabToken(token); err != nil {
		h.l.Errorf(ctx, "GitLab token verification failed: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	// Check rate limit
	if err := h.security.CheckRateLimit("gitlab"); err != nil {
		h.l.Warnf(ctx, "Rate limit exceeded: %v", err)
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
		return
	}

	// Get event type
	eventType := c.GetHeader("X-Gitlab-Event")

	// Parse event
	var event *model.WebhookEvent
	switch eventType {
	case "Push Hook":
		event, err = h.gitlabParser.ParsePushEvent(body)
	case "Merge Request Hook":
		event, err = h.gitlabParser.ParseMergeRequestEvent(body)
	case "Issue Hook":
		event, err = h.gitlabParser.ParseIssueEvent(body)
	default:
		h.l.Infof(ctx, "Unsupported GitLab event type: %s", eventType)
		pkgResponse.OK(c, gin.H{"status": "ignored", "reason": "unsupported event type"})
		return
	}

	if err != nil {
		h.l.Errorf(ctx, "Failed to parse GitLab event: %v", err)
		pkgResponse.Error(c, err, nil)
		return
	}

	// Process in background
	go h.processWebhookAsync(*event)

	// Acknowledge immediately
	pkgResponse.OK(c, gin.H{"status": "accepted"})
}

// processWebhookAsync processes webhook in background
func (h *Handler) processWebhookAsync(event model.WebhookEvent) {
	// Create background context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	h.l.Infof(ctx, "Processing webhook async: %s/%s from %s", event.EventType, event.Action, event.Repository)

	// Process webhook
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
