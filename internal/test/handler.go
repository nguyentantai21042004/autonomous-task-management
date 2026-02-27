package test

import (
	"fmt"

	"autonomous-task-management/internal/agent/orchestrator"
	"autonomous-task-management/internal/model"
	"autonomous-task-management/internal/router"
	pkgLog "autonomous-task-management/pkg/log"

	"github.com/gin-gonic/gin"
)

type handler struct {
	l            pkgLog.Logger
	router       router.Router
	orchestrator *orchestrator.Orchestrator
}

// HandleTestMessage is a test endpoint to simulate message processing
// @Summary Test message processing
// @Description Send a test message to simulate bot interaction and test semantic router
// @Tags test
// @Accept json
// @Produce json
// @Param request body TestMessageRequest true "Test message"
// @Success 200 {object} TestMessageResponse
// @Router /test/message [post]
func (h *handler) HandleTestMessage(c *gin.Context) {
	ctx := c.Request.Context()

	var req TestMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	// Default values
	if req.UserID == 0 {
		req.UserID = 999999999 // Test user ID
	}

	// Get scope
	sc := model.Scope{UserID: fmt.Sprintf("telegram_%d", req.UserID)}

	// Get conversation history
	session := h.orchestrator.GetSession(sc.UserID)
	history := []string{}
	if session != nil && len(session.Messages) > 0 {
		start := len(session.Messages) - 6
		if start < 0 {
			start = 0
		}
		for i := start; i < len(session.Messages); i++ {
			if len(session.Messages[i].Parts) > 0 {
				history = append(history, session.Messages[i].Parts[0].Text)
			}
		}
	}

	// Classify intent using router
	routerOutput, err := h.router.Classify(ctx, req.Text, history)
	if err != nil {
		h.l.Errorf(ctx, "internal.test.HandleTestMessage: Router classification failed: %v", err)
		c.JSON(500, TestMessageResponse{
			Success: false,
			Error:   "Router classification failed",
			Details: err.Error(),
		})
		return
	}

	// Return response WITHOUT calling actual Telegram API
	response := TestMessageResponse{
		Success:    true,
		Intent:     string(routerOutput.Intent),
		Confidence: routerOutput.Confidence,
		Reasoning:  routerOutput.Reasoning,
		Text:       req.Text,
		UserID:     req.UserID,
		History:    history,
	}

	h.l.Infof(ctx, "internal.test.HandleTestMessage: text=%q intent=%s confidence=%d%%",
		req.Text, routerOutput.Intent, routerOutput.Confidence)

	c.JSON(200, response)
}

// HandleResetSession resets the conversation session for a test user
// @Summary Reset test user session
// @Description Clear conversation history for a test user
// @Tags test
// @Accept json
// @Produce json
// @Param request body ResetSessionRequest true "Reset session"
// @Success 200 {object} ResetSessionResponse
// @Router /test/reset [post]
func (h *handler) HandleResetSession(c *gin.Context) {
	ctx := c.Request.Context()

	var req ResetSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	if req.UserID == 0 {
		req.UserID = 999999999
	}

	sc := model.Scope{UserID: fmt.Sprintf("telegram_%d", req.UserID)}
	h.orchestrator.ClearSession(sc.UserID)

	h.l.Infof(ctx, "internal.test.HandleResetSession: Cleared session for user_id=%d", req.UserID)

	c.JSON(200, ResetSessionResponse{
		Success: true,
		Message: fmt.Sprintf("Session cleared for user %d", req.UserID),
		UserID:  req.UserID,
	})
}

// HandleHealthCheck returns the health status of test endpoints
// @Summary Test health check
// @Description Check if test endpoints are available
// @Tags test
// @Produce json
// @Success 200 {object} HealthCheckResponse
// @Router /test/health [get]
func (h *handler) HandleHealthCheck(c *gin.Context) {
	c.JSON(200, HealthCheckResponse{
		Status:  "ok",
		Message: "Test endpoints are available",
	})
}
