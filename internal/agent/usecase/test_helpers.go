package usecase

// stopCleanupForTest stops the background cleanup goroutine for testing
// This should only be used in tests to prevent goroutine leaks
func (uc *implUseCase) stopCleanupForTest() {
	close(uc.stopCleanup)
}
