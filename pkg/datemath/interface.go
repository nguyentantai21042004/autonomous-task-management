package datemath

import "time"

// IParser defines the interface for the date math parser.
type IParser interface {
	// Parse converts a relative date string to an absolute time.Time.
	Parse(relative string, baseTime time.Time) (time.Time, error)
	// EndOfDay returns 23:59:59 at the end of the given start-of-day time.
	EndOfDay(startOfDay time.Time) time.Time
}
