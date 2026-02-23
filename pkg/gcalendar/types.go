package gcalendar

import "time"

// CreateEventRequest is the input for creating a Google Calendar event.
type CreateEventRequest struct {
	CalendarID  string
	Summary     string
	Description string
	StartTime   time.Time
	EndTime     time.Time
	Timezone    string // e.g. "Asia/Ho_Chi_Minh"
}

// Event is a simplified representation of a Google Calendar event.
type Event struct {
	ID          string
	Summary     string
	Description string
	HtmlLink    string
	StartTime   time.Time
	EndTime     time.Time
}
