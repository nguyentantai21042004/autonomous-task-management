package telegram

// errorMessage returns a user-facing error string for the given error.
func errorMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
