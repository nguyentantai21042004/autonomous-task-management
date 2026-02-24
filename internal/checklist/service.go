package checklist

import (
	"context"
	"regexp"
	"strings"
)

const (
	CheckboxUnchecked = `- [ ]`
	CheckboxChecked   = `- [x]`
	// Regex pattern: captures indent, checkbox state, and text
	// Example: "  - [x] Task name" → groups: ["  ", "x", "Task name"]
	CheckboxPattern = `(?m)^(\s*)- \[([ xX])\] (.+)$`
)

type Service interface {
	// ParseCheckboxes extracts all checkboxes from markdown content
	ParseCheckboxes(content string) []Checkbox

	// GetStats calculates checklist statistics
	GetStats(content string) ChecklistStats

	// UpdateCheckbox updates checkbox state by text match
	UpdateCheckbox(ctx context.Context, input UpdateCheckboxInput) (UpdateCheckboxOutput, error)

	// UpdateAllCheckboxes sets all checkboxes to specified state
	UpdateAllCheckboxes(content string, checked bool) string

	// IsFullyCompleted checks if all checkboxes are checked
	IsFullyCompleted(content string) bool
}

type service struct {
	pattern *regexp.Regexp
}

func New() Service {
	return &service{
		pattern: regexp.MustCompile(CheckboxPattern),
	}
}

// sanitizeContent removes code blocks before checkbox parsing
// Prevents matching fake checkboxes in code examples
func sanitizeContent(content string) string {
	// Remove fenced code blocks (```...```)
	fencedCodeBlockPattern := regexp.MustCompile("(?s)```.*?```")
	sanitized := fencedCodeBlockPattern.ReplaceAllString(content, "")

	// Remove inline code blocks (`...`)
	inlineCodePattern := regexp.MustCompile("`[^`]+`")
	sanitized = inlineCodePattern.ReplaceAllString(sanitized, "")

	return sanitized
}

// ParseCheckboxes extracts all checkboxes from markdown
func (s *service) ParseCheckboxes(content string) []Checkbox {
	// Sanitize first to remove code blocks
	sanitized := sanitizeContent(content)

	matches := s.pattern.FindAllStringSubmatch(sanitized, -1)
	checkboxes := make([]Checkbox, 0, len(matches))

	lineNum := 0
	for _, match := range matches {
		if len(match) != 4 {
			continue
		}

		checkbox := Checkbox{
			Line:    lineNum,
			Indent:  match[1],
			Checked: strings.ToLower(match[2]) == "x",
			Text:    strings.TrimSpace(match[3]),
			RawLine: match[0],
		}
		checkboxes = append(checkboxes, checkbox)
		lineNum++
	}

	return checkboxes
}

// GetStats calculates checklist statistics
func (s *service) GetStats(content string) ChecklistStats {
	checkboxes := s.ParseCheckboxes(content)
	total := len(checkboxes)

	if total == 0 {
		return ChecklistStats{
			Total:     0,
			Completed: 0,
			Pending:   0,
			Progress:  0,
		}
	}

	completed := 0
	for _, cb := range checkboxes {
		if cb.Checked {
			completed++
		}
	}

	pending := total - completed
	progress := float64(completed) / float64(total) * 100

	return ChecklistStats{
		Total:     total,
		Completed: completed,
		Pending:   pending,
		Progress:  progress,
	}
}

// UpdateCheckbox updates checkbox state by text match (partial match)
func (s *service) UpdateCheckbox(ctx context.Context, input UpdateCheckboxInput) (UpdateCheckboxOutput, error) {
	if input.Content == "" {
		return UpdateCheckboxOutput{Content: input.Content, Updated: false}, nil
	}

	lines := strings.Split(input.Content, "\n")
	updated := false
	count := 0

	// Normalize search text for matching
	searchText := strings.ToLower(strings.TrimSpace(input.CheckboxText))

	for i, line := range lines {
		// Check if line is a checkbox
		if !strings.Contains(line, "- [") {
			continue
		}

		// Extract checkbox text
		matches := s.pattern.FindStringSubmatch(line)
		if len(matches) != 4 {
			continue
		}

		checkboxText := strings.ToLower(strings.TrimSpace(matches[3]))

		// Partial match: if search text is substring of checkbox text
		if !strings.Contains(checkboxText, searchText) {
			continue
		}

		// Update checkbox state
		indent := matches[1]
		text := matches[3]

		if input.Checked {
			lines[i] = indent + CheckboxChecked + " " + text
		} else {
			lines[i] = indent + CheckboxUnchecked + " " + text
		}

		updated = true
		count++
	}

	return UpdateCheckboxOutput{
		Content: strings.Join(lines, "\n"),
		Updated: updated,
		Count:   count,
	}, nil
}

// UpdateAllCheckboxes sets all checkboxes to specified state
func (s *service) UpdateAllCheckboxes(content string, checked bool) string {
	state := CheckboxUnchecked
	if checked {
		state = CheckboxChecked
	}

	// Replace all checkbox states
	result := s.pattern.ReplaceAllStringFunc(content, func(match string) string {
		matches := s.pattern.FindStringSubmatch(match)
		if len(matches) != 4 {
			return match
		}
		return matches[1] + state + " " + matches[3]
	})

	return result
}

// IsFullyCompleted checks if all checkboxes are checked
func (s *service) IsFullyCompleted(content string) bool {
	checkboxes := s.ParseCheckboxes(content)
	if len(checkboxes) == 0 {
		return false // No checkboxes = not a checklist task
	}

	for _, cb := range checkboxes {
		if !cb.Checked {
			return false
		}
	}
	return true
}
