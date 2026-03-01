package usecase

import (
	"autonomous-task-management/internal/checklist"
)

// GetStats calculates checklist statistics
func (s *implUseCase) GetStats(content string) checklist.ChecklistStats {
	checkboxes := s.ParseCheckboxes(content)
	total := len(checkboxes)

	if total == 0 {
		return checklist.ChecklistStats{
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

	return checklist.ChecklistStats{
		Total:     total,
		Completed: completed,
		Pending:   pending,
		Progress:  progress,
	}
}
