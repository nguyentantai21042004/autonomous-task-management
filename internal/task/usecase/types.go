package usecase

import "time"

// ParsedTask is a task extracted from user input by the LLM.
type ParsedTask struct {
	Title                    string   `json:"title"`
	Description              string   `json:"description"`
	DueDateAbsolute          string   `json:"due_date_absolute"`
	Priority                 string   `json:"priority"`
	Tags                     []string `json:"tags"`
	EstimatedDurationMinutes int      `json:"estimated_duration_minutes"`
}

// taskWithDate is a private type used internally to carry a parsed task
// alongside its computed absolute due date.
type taskWithDate struct {
	Title                    string
	Description              string
	DueDateAbsolute          time.Time
	Priority                 string
	Tags                     []string
	EstimatedDurationMinutes int
}
