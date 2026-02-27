package orchestrator

import (
	"fmt"
	"time"
)

// Date format
const (
	DateFormatISO = "2006-01-02"
)

// buildTimeContext creates a temporal context string for LLM
func buildTimeContext(timezone string) string {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}

	now := time.Now().In(loc)

	// Calculate week boundaries (Monday-Sunday)
	weekday := int(now.Weekday())
	if weekday == 0 { // Sunday
		weekday = 7
	}
	weekStart := now.AddDate(0, 0, -(weekday - 1)) // Monday
	weekEnd := weekStart.AddDate(0, 0, 6)          // Sunday
	tomorrow := now.AddDate(0, 0, 1)

	// Build context string using template from constant.go
	context := fmt.Sprintf(
		TimeContextTemplate,
		now.Format(DateFormatISO),
		now.Weekday().String(),
		weekStart.Format(DateFormatISO),
		weekEnd.Format(DateFormatISO),
		tomorrow.Format(DateFormatISO),
		weekStart.Format(DateFormatISO),
		weekEnd.Format(DateFormatISO),
		tomorrow.Format(DateFormatISO),
	)

	return context
}
