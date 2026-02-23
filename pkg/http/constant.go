package http

import "time"

const (
	// DefaultTimeout is the default HTTP client timeout.
	DefaultTimeout = 30 * time.Second
	// DefaultRetries is the default number of retries.
	DefaultRetries = 3
	// DefaultRetryWait is the default wait between retries.
	DefaultRetryWait = 1 * time.Second
)

// DefaultConfig returns default ClientConfig.
func DefaultConfig() ClientConfig {
	return ClientConfig{
		Timeout:   DefaultTimeout,
		Retries:   DefaultRetries,
		RetryWait: DefaultRetryWait,
	}
}
