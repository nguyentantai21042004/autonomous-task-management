package http

import (
	"net/http"
	"time"
)

// ClientConfig holds configuration for the HTTP client.
type ClientConfig struct {
	Timeout   time.Duration
	Retries   int
	RetryWait time.Duration
}

// clientImpl implements IClient.
type clientImpl struct {
	client *http.Client
	config ClientConfig
}
