package gcalendar

import "time"

// Event represents a calendar event.
type Event struct {
	ID          string    `json:"id"`
	Summary     string    `json:"summary"`
	Description string    `json:"description"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Location    string    `json:"location"`
	HtmlLink    string    `json:"html_link"`
}

// CreateEventRequest contains fields for creating an event.
type CreateEventRequest struct {
	CalendarID  string
	Summary     string
	Description string
	Location    string
	StartTime   time.Time
	EndTime     time.Time
	Timezone    string
}

// ListEventsRequest contains fields for listing events.
type ListEventsRequest struct {
	CalendarID string
	TimeMin    time.Time
	TimeMax    time.Time
	MaxResults int64
}
