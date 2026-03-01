package usecase

import (
	"context"
	"strings"

	"autonomous-task-management/internal/checklist"
)

// UpdateCheckbox updates checkbox state by text match (partial match)
func (s *implUseCase) UpdateCheckbox(ctx context.Context, input checklist.UpdateCheckboxInput) (checklist.UpdateCheckboxOutput, error) {
	if input.Content == "" {
		return checklist.UpdateCheckboxOutput{Content: input.Content, Updated: false}, nil
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

	return checklist.UpdateCheckboxOutput{
		Content: strings.Join(lines, "\n"),
		Updated: updated,
		Count:   count,
	}, nil
}
