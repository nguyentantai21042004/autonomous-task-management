package usecase

// UpdateAllCheckboxes sets all checkboxes to specified state
func (s *implUseCase) UpdateAllCheckboxes(content string, checked bool) string {
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
