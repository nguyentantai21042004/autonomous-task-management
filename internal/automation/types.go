package automation

import (
	"autonomous-task-management/internal/model"
)

// ProcessWebhookInput is input for webhook processing
type ProcessWebhookInput struct {
	Event model.WebhookEvent
}

// ProcessWebhookOutput is result of webhook processing
type ProcessWebhookOutput struct {
	TasksUpdated int      // Number of tasks updated
	TaskIDs      []string // IDs of updated tasks
	Message      string   // Summary message
}

// MatchCriteria defines how to match webhook event to tasks
type MatchCriteria struct {
	Repository string   // Repository name (e.g., "user/repo")
	Tags       []string // Tags to match (e.g., ["#repo/myproject", "#pr/123"])
	Keywords   []string // Keywords in content
}

// TaskMatch represents a matched task
type TaskMatch struct {
	TaskID      string  // Memos task ID
	Content     string  // Task content
	MatchScore  float64 // Match confidence (0-1)
	MatchReason string  // Why it matched
}
