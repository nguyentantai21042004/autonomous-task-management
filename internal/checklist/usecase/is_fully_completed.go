package usecase

// IsFullyCompleted checks if all checkboxes are checked
func (s *implUseCase) IsFullyCompleted(content string) bool {
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
