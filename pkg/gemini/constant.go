package gemini

import "time"

const (
	// DefaultModel is the default Gemini model
	DefaultModel = "gemini-2.5-flash"

	// DefaultAPIURL is the default Gemini API endpoint
	DefaultAPIURL = "https://generativelanguage.googleapis.com/v1beta"

	// DefaultTimeout is the default HTTP client timeout
	DefaultTimeout = 30 * time.Second
)
