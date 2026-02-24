package voyage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	DefaultBaseURL = "https://api.voyageai.com/v1"
	DefaultModel   = "voyage-3" // Latest model with 1024 dimensions
)

// Client is the Voyage AI embedding API client.
type Client struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// New creates a new Voyage AI client.
func New(apiKey string) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("voyage API key is required")
	}

	return &Client{
		apiKey:     apiKey,
		baseURL:    DefaultBaseURL,
		model:      DefaultModel,
		httpClient: &http.Client{},
	}, nil
}

// WithModel sets a custom model (e.g., "voyage-3", "voyage-large-2").
func (c *Client) WithModel(model string) *Client {
	c.model = model
	return c
}

// WithBaseURL overrides the default Voyage API base URL.
func (c *Client) WithBaseURL(baseURL string) *Client {
	c.baseURL = baseURL
	return c
}

// Embed generates embeddings for the given texts.
func (c *Client) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("no texts provided")
	}

	// Build request
	reqBody := EmbedRequest{
		Input: texts,
		Model: c.model,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/embeddings", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	// Execute request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call Voyage API: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if jsonErr := json.NewDecoder(resp.Body).Decode(&errResp); jsonErr == nil {
			return nil, fmt.Errorf("voyage API error (%d): %s", resp.StatusCode, errResp.Error.Message)
		}
		return nil, fmt.Errorf("voyage API error: %d", resp.StatusCode)
	}

	// Parse response
	var embedResp EmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract embeddings
	embeddings := make([][]float32, len(embedResp.Data))
	for i, data := range embedResp.Data {
		embeddings[i] = data.Embedding
	}

	return embeddings, nil
}
