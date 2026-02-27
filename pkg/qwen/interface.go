package qwen

import "context"

// IQwen defines the interface for Qwen API client.
// Implementations are safe for concurrent use.
type IQwen interface {
	// GenerateContent sends a generation request to Qwen API
	GenerateContent(ctx context.Context, req *Request) (*Response, error)

	// Model returns the model being used
	Model() string
}

// New creates a new Qwen client with the given configuration
func New(cfg Config) (IQwen, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return newQwenImpl(cfg), nil
}
