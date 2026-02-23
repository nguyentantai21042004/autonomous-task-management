package datemath

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Parser converts relative date strings to absolute time.Time values.
type Parser struct {
	location *time.Location
}

// NewParser creates a new date parser for the given IANA timezone string.
// e.g. "Asia/Ho_Chi_Minh"
func NewParser(timezone string) (*Parser, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone %q: %w", timezone, err)
	}
	return &Parser{location: loc}, nil
}

// Parse converts a relative date string to an absolute time.Time.
// The baseTime is used as the reference point (usually time.Now()).
func (p *Parser) Parse(relative string, baseTime time.Time) (time.Time, error) {
	relative = strings.ToLower(strings.TrimSpace(relative))

	switch relative {
	case "today":
		return p.startOfDay(baseTime), nil
	case "tomorrow":
		return p.startOfDay(baseTime.AddDate(0, 0, 1)), nil
	case "yesterday":
		return p.startOfDay(baseTime.AddDate(0, 0, -1)), nil
	}

	// Handle "in X days/weeks/months"
	if strings.HasPrefix(relative, "in ") {
		return p.parseInDuration(relative, baseTime)
	}

	// Handle "next <weekday>"
	if strings.HasPrefix(relative, "next ") {
		return p.parseNextWeekday(relative, baseTime)
	}

	// Fallback: treat unknown as today
	return p.startOfDay(baseTime), nil
}

// parseInDuration handles patterns like "in 3 days", "in 2 weeks", "in 1 month".
func (p *Parser) parseInDuration(relative string, baseTime time.Time) (time.Time, error) {
	re := regexp.MustCompile(`in (\d+) (day|days|week|weeks|month|months)`)
	matches := re.FindStringSubmatch(relative)
	if len(matches) != 3 {
		return baseTime, fmt.Errorf("invalid duration format: %q", relative)
	}

	amount, _ := strconv.Atoi(matches[1])
	unit := matches[2]

	switch {
	case strings.HasPrefix(unit, "day"):
		return p.startOfDay(baseTime.AddDate(0, 0, amount)), nil
	case strings.HasPrefix(unit, "week"):
		return p.startOfDay(baseTime.AddDate(0, 0, amount*7)), nil
	case strings.HasPrefix(unit, "month"):
		return p.startOfDay(baseTime.AddDate(0, amount, 0)), nil
	}

	return baseTime, fmt.Errorf("unknown time unit: %q", unit)
}

// parseNextWeekday handles patterns like "next monday", "next friday".
func (p *Parser) parseNextWeekday(relative string, baseTime time.Time) (time.Time, error) {
	weekdays := map[string]time.Weekday{
		"monday":    time.Monday,
		"tuesday":   time.Tuesday,
		"wednesday": time.Wednesday,
		"thursday":  time.Thursday,
		"friday":    time.Friday,
		"saturday":  time.Saturday,
		"sunday":    time.Sunday,
	}

	dayName := strings.TrimPrefix(relative, "next ")
	targetWeekday, ok := weekdays[dayName]
	if !ok {
		return baseTime, fmt.Errorf("unknown weekday: %q", dayName)
	}

	currentWeekday := baseTime.Weekday()
	daysUntil := int(targetWeekday - currentWeekday)
	if daysUntil <= 0 {
		daysUntil += 7
	}

	return p.startOfDay(baseTime.AddDate(0, 0, daysUntil)), nil
}

// startOfDay returns midnight at the start of the given day in the parser's timezone.
func (p *Parser) startOfDay(t time.Time) time.Time {
	t = t.In(p.location)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, p.location)
}

// EndOfDay returns 23:59:59 at the end of the given start-of-day time.
func (p *Parser) EndOfDay(startOfDay time.Time) time.Time {
	return startOfDay.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
}
