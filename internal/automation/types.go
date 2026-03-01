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
