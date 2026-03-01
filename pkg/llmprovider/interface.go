package llmprovider

import "context"

// Provider defines the interface for LLM providers.
// Implementations are safe for concurrent use.
type Provider interface {
	// GenerateContent sends a generation request and returns a response
	GenerateContent(ctx context.Context, req *Request) (*Response, error)

	// Name returns the provider name (e.g., "qwen", "gemini")
	Name() string

	// Model returns the model being used
	Model() string
}

// IManager defines the interface for the LLM Provider Manager.
// It handles fallback and retry logic across multiple providers.
type IManager interface {
	// GenerateContent iterates through providers in priority order with fallback logic
	GenerateContent(ctx context.Context, req *Request) (*Response, error)
}
