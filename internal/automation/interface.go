package automation

import (
	"autonomous-task-management/internal/model"
	"context"
)

type UseCase interface {
	// ProcessWebhook processes a webhook event and updates tasks
	ProcessWebhook(ctx context.Context, sc model.Scope, input ProcessWebhookInput) (ProcessWebhookOutput, error)

	// CompleteTask manually marks a task as complete
	CompleteTask(ctx context.Context, sc model.Scope, taskID string) error

	// ArchiveCompletedTasks archives all fully completed tasks
	ArchiveCompletedTasks(ctx context.Context, sc model.Scope) (int, error)
}
