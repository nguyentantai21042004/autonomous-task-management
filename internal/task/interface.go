package task

import (
	"context"

	"autonomous-task-management/internal/model"
)

// UseCase defines the business logic interface for the task domain.
type UseCase interface {
	// CreateBulk parses raw text from the user, creates tasks in Memos, and optionally schedules events in Google Calendar.
	CreateBulk(ctx context.Context, sc model.Scope, input CreateBulkInput) (CreateBulkOutput, error)

	// Search performs semantic search on tasks.
	Search(ctx context.Context, sc model.Scope, input SearchInput) (SearchOutput, error)
}
