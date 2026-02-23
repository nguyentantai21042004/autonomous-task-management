package usecase

import "time"

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
