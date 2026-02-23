package usecase

// coalesce returns the first non-empty string — used for partial updates.
// If new value is provided, use it; otherwise fall back to the existing value.
func (uc *implUseCase) coalesce(newVal, existing string) string {
	if newVal != "" {
		return newVal
	}
	return existing
}
