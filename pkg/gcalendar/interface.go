package gcalendar

import "context"

// IGCalendar defines the interface for Google Calendar operations.
// Implementations are safe for concurrent use.
type IGCalendar interface {
	CreateEvent(ctx context.Context, req CreateEventRequest) (*Event, error)
	ListEvents(ctx context.Context, req ListEventsRequest) ([]Event, error)
}

// New creates a new IGCalendar instance from raw Service Account JSON bytes.
func New(ctx context.Context, credentialsJSON []byte) (IGCalendar, error) {
	return NewClientFromCredentialsJSON(ctx, credentialsJSON)
}
