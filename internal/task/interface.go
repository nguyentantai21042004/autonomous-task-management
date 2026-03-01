package task

import (
	"context"

	"autonomous-task-management/internal/agent"
	"autonomous-task-management/internal/model"
	"autonomous-task-management/pkg/gcalendar"
)

// CalendarClient abstracts the Google Calendar API.
type CalendarClient interface {
	CreateEvent(ctx context.Context, req gcalendar.CreateEventRequest) (*gcalendar.Event, error)
	ListEvents(ctx context.Context, req gcalendar.ListEventsRequest) ([]gcalendar.Event, error)
}

// UseCase defines the business logic interface for the task domain.
type UseCase interface {
	// CreateBulk parses raw text from the user, creates tasks in Memos, and optionally schedules events in Google Calendar.
	CreateBulk(ctx context.Context, sc model.Scope, input CreateBulkInput) (CreateBulkOutput, error)

	// Search performs semantic search on tasks.
	Search(ctx context.Context, sc model.Scope, input SearchInput) (SearchOutput, error)

	// AnswerQuery handles questions and synthesizes intelligence via RAG contextualization.
	AnswerQuery(ctx context.Context, sc model.Scope, input QueryInput) (QueryOutput, error)

	// RegisterAgentTools registers this domain's agent tools into the registry.
	RegisterAgentTools(registry *agent.ToolRegistry)
}
