package gemini

import "context"

// IGemini defines the interface for Gemini API client.
// Implementations are safe for concurrent use.
type IGemini interface {
	// GenerateContent sends a generation request to Gemini API
	GenerateContent(ctx context.Context, req *Request) (*Response, error)

	// Model returns the model being used
	Model() string
}

// New creates a new Gemini client with the given configuration
func New(cfg Config) (IGemini, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return newGeminiImpl(cfg), nil
}
