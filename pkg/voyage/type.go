package voyage

import "net/http"

// VoyageConfig holds the configuration for the Voyage client.
type VoyageConfig struct {
	APIKey string
}

// Client implements IVoyage using the Voyage AI API.
type Client struct {
	apiKey     string
	httpClient *http.Client
}

// Request defines the request body for Embedding API
type Request struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

// Response defines the response body from Embedding API
type Response struct {
	Object string      `json:"object"`
	Data   []Embedding `json:"data"`
	Model  string      `json:"model"`
	Usage  Usage       `json:"usage"`
}

// Embedding represents a single embedding object
type Embedding struct {
	Object    string    `json:"object"`
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

// Usage represents token usage
type Usage struct {
	TotalTokens int `json:"total_tokens"`
}
