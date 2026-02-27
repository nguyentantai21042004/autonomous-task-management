package llmprovider

import (
	"errors"
	"fmt"
)

var (
	// ErrAllProvidersFailed indicates all providers failed to generate content
	ErrAllProvidersFailed = errors.New("all providers failed")

	// ErrNoProvidersConfigured indicates no providers are enabled
	ErrNoProvidersConfigured = errors.New("no providers configured")

	// ErrInvalidRequest indicates the request is malformed
	ErrInvalidRequest = errors.New("invalid request")

	// ErrProviderTimeout indicates a provider request timed out
	ErrProviderTimeout = errors.New("provider timeout")

	// ErrProviderRateLimited indicates rate limit exceeded
	ErrProviderRateLimited = errors.New("provider rate limited")
)

// ProviderError wraps provider-specific errors
type ProviderError struct {
	Provider string
	Err      error
}

func (e *ProviderError) Error() string {
	return fmt.Sprintf("provider %s: %v", e.Provider, e.Err)
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}
