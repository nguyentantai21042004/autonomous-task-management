package http

import "context"

// IClient defines the interface for HTTP client with retry and timeout.
// Implementations are safe for concurrent use.
type IClient interface {
	Get(ctx context.Context, url string, headers map[string]string) ([]byte, int, error)
	Post(ctx context.Context, url string, body interface{}, headers map[string]string) ([]byte, int, error)
}

// NewClient creates a new HTTP client. Returns the interface.
func NewClient(cfg ClientConfig) IClient {
	return &clientImpl{
		client: defaultHTTPClient(cfg.Timeout),
		config: cfg,
	}
}
