package datemath

import "time"

// ParseResult holds the result of parsing a relative date string.
type ParseResult struct {
	AbsoluteTime time.Time
	IsAllDay     bool
}
