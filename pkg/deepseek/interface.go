package deepseek

import (
	"context"
	"fmt"
)

// IDeepSeek defines the interface for DeepSeek LLM client
type IDeepSeek interface {
	GenerateContent(ctx context.Context, req *Request) (*Response, error)
}

// New creates a new DeepSeek client with the given configuration
func New(cfg Config) (IDeepSeek, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("pkg: API key is required")
	}
	return newDeepSeekImpl(cfg), nil
}
