package llmprovider

import (
	"context"
	"fmt"
	"time"

	"autonomous-task-management/pkg/log"
)

// Manager orchestrates provider selection, fallback, and retry logic
type Manager struct {
	providers []Provider
	config    *Config
	logger    log.Logger
}

// Config defines configuration for the Provider Manager
type Config struct {
	FallbackEnabled bool
	RetryAttempts   int
	RetryDelay      time.Duration
	MaxTotalTimeout time.Duration // NEW: Global timeout for entire fallback chain
}

// NewManager creates a new Provider Manager with the given providers, config, and logger
func NewManager(providers []Provider, config *Config, logger log.Logger) *Manager {
	return &Manager{
		providers: providers,
		config:    config,
		logger:    logger,
	}
}

// GenerateContent iterates through providers in priority order with fallback logic
func (m *Manager) GenerateContent(ctx context.Context, req *Request) (*Response, error) {
	if len(m.providers) == 0 {
		return nil, ErrNoProvidersConfigured
	}

	// Create context with global timeout for entire fallback chain
	var cancel context.CancelFunc
	if m.config.MaxTotalTimeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, m.config.MaxTotalTimeout)
		defer cancel()
	}

	var lastErr error

	// Iterate through providers in priority order
	for _, provider := range m.providers {
		// Check if context is already cancelled (timeout exceeded)
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("global timeout exceeded after trying %d provider(s): %w",
				len(m.providers), ctx.Err())
		default:
			// Continue
		}

		// Call generateWithRetry for each provider
		resp, err := m.generateWithRetry(ctx, provider, req)
		if err == nil {
			// On success, log metrics and return response
			m.logSuccess(ctx, provider, resp)
			return resp, nil
		}

		// On failure, log error and try next provider
		m.logFailure(ctx, provider, err)
		lastErr = err

		// If fallback is disabled, stop after first provider
		if !m.config.FallbackEnabled {
			break
		}
	}

	// Return error if all providers fail
	return nil, fmt.Errorf("%w: %v", ErrAllProvidersFailed, lastErr)
}

// generateWithRetry implements retry mechanism with exponential backoff
func (m *Manager) generateWithRetry(ctx context.Context, provider Provider, req *Request) (*Response, error) {
	var lastErr error

	for attempt := 0; attempt < m.config.RetryAttempts; attempt++ {
		// Add delay for retries (exponential backoff)
		if attempt > 0 {
			delay := time.Duration(attempt) * m.config.RetryDelay
			select {
			case <-time.After(delay):
				// Continue after delay
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		// Attempt generation
		resp, err := provider.GenerateContent(ctx, req)
		if err == nil {
			return resp, nil
		}

		lastErr = err
	}

	return nil, lastErr
}

// logSuccess logs successful LLM generation with metrics
func (m *Manager) logSuccess(ctx context.Context, provider Provider, resp *Response) {
	m.logger.Info(ctx, "LLM generation successful",
		"provider", provider.Name(),
		"model", provider.Model(),
		"input_tokens", resp.Usage.InputTokens,
		"output_tokens", resp.Usage.OutputTokens,
	)
}

// logFailure logs failed LLM generation attempts
func (m *Manager) logFailure(ctx context.Context, provider Provider, err error) {
	m.logger.Warn(ctx, "LLM generation failed",
		"provider", provider.Name(),
		"model", provider.Model(),
		"error", err.Error(),
	)
}
