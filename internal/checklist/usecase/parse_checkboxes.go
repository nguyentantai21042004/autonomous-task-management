package usecase

import (
	"strings"

	"autonomous-task-management/internal/checklist"
)

// ParseCheckboxes extracts all checkboxes from markdown
func (s *implUseCase) ParseCheckboxes(content string) []checklist.Checkbox {
	// Sanitize first to remove code blocks
	sanitized := sanitizeContent(content)

	matches := s.pattern.FindAllStringSubmatch(sanitized, -1)
	checkboxes := make([]checklist.Checkbox, 0, len(matches))

	lineNum := 0
	for _, match := range matches {
		if len(match) != 4 {
			continue
		}

		checkbox := checklist.Checkbox{
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
