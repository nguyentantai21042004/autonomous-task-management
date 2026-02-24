package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"autonomous-task-management/internal/agent"
	"autonomous-task-management/pkg/gcalendar"
	pkgLog "autonomous-task-management/pkg/log"
)

// CalendarClient abstract Google Calendar API for mocking
type CalendarClient interface {
	ListEvents(ctx context.Context, req gcalendar.ListEventsRequest) ([]gcalendar.Event, error)
}

type CheckCalendarTool struct {
	calendar CalendarClient
	l        pkgLog.Logger
}

func NewCheckCalendarTool(calendar CalendarClient, l pkgLog.Logger) *CheckCalendarTool {
	return &CheckCalendarTool{
		calendar: calendar,
		l:        l,
	}
}

func (t *CheckCalendarTool) Name() string {
	return "check_calendar"
}

func (t *CheckCalendarTool) Description() string {
	return "Check Google Calendar for events in a specific time range. Useful for detecting scheduling conflicts."
}

func (t *CheckCalendarTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"start_date": map[string]interface{}{
				"type":        "string",
				"description": "Start date in YYYY-MM-DD format",
			},
			"end_date": map[string]interface{}{
				"type":        "string",
				"description": "End date in YYYY-MM-DD format",
			},
			"time_zone": map[string]interface{}{
				"type":        "string",
				"description": "Time zone (e.g., 'Asia/Ho_Chi_Minh')",
				"default":     "Asia/Ho_Chi_Minh",
			},
		},
		"required": []string{"start_date", "end_date"},
	}
}

type CheckCalendarInput struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	TimeZone  string `json:"time_zone"`
}

type CheckCalendarOutput struct {
	Events      []CalendarEvent `json:"events"`
	EventCount  int             `json:"event_count"`
	HasConflict bool            `json:"has_conflict"`
	Summary     string          `json:"summary"`
}

type CalendarEvent struct {
	Title     string    `json:"title"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Location  string    `json:"location,omitempty"`
}

func (t *CheckCalendarTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	// Parse input
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}

	var params CheckCalendarInput
	if err := json.Unmarshal(inputBytes, &params); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	// Set default timezone
	if params.TimeZone == "" {
		params.TimeZone = "Asia/Ho_Chi_Minh"
	}

	t.l.Infof(ctx, "check_calendar: checking %s to %s (%s)", params.StartDate, params.EndDate, params.TimeZone)

	// Parse dates
	startTime, err := time.Parse("2006-01-02", params.StartDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start_date format: %w", err)
	}

	endTime, err := time.Parse("2006-01-02", params.EndDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end_date format: %w", err)
	}

	// Add timezone and set time bounds
	loc, err := time.LoadLocation(params.TimeZone)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone: %w", err)
	}

	startTime = time.Date(startTime.Year(), startTime.Month(), startTime.Day(), 0, 0, 0, 0, loc)
	endTime = time.Date(endTime.Year(), endTime.Month(), endTime.Day(), 23, 59, 59, 0, loc)

	// Query Google Calendar
	events, err := t.calendar.ListEvents(ctx, gcalendar.ListEventsRequest{
		TimeMin: startTime,
		TimeMax: endTime,
	})
	if err != nil {
		t.l.Errorf(ctx, "check_calendar: failed to get events: %v", err)
		return CheckCalendarOutput{
			Events:      []CalendarEvent{},
			EventCount:  0,
			HasConflict: false,
			Summary:     fmt.Sprintf("❌ Không thể truy cập lịch: %v", err),
		}, nil
	}

	// Convert to output format
	var calendarEvents []CalendarEvent
	for _, event := range events {
		calendarEvents = append(calendarEvents, CalendarEvent{
			Title:     event.Summary,
			StartTime: event.StartTime,
			EndTime:   event.EndTime,
			Location:  event.Location,
		})
	}

	// Generate summary
	var summary string
	if len(calendarEvents) == 0 {
		summary = fmt.Sprintf("📅 Không có sự kiện nào từ %s đến %s", params.StartDate, params.EndDate)
	} else {
		summary = fmt.Sprintf("📅 Tìm thấy %d sự kiện từ %s đến %s:\n", len(calendarEvents), params.StartDate, params.EndDate)
		for i, event := range calendarEvents {
			summary += fmt.Sprintf("%d. %s (%s - %s)\n",
				i+1,
				event.Title,
				event.StartTime.Format("02/01 15:04"),
				event.EndTime.Format("15:04"))
		}
	}

	return CheckCalendarOutput{
		Events:      calendarEvents,
		EventCount:  len(calendarEvents),
		HasConflict: len(calendarEvents) > 0,
		Summary:     summary,
	}, nil
}

// Verify interface compliance
var _ agent.Tool = (*CheckCalendarTool)(nil)
