package voyage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// New creates a new Voyage client.
func New(apiKey string) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("voyage: API key is required")
	}
	return &Client{
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}, nil
}

// Embed generates embeddings for the given texts.
func (c *Client) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("voyage: API key is required")
	}
	if len(texts) == 0 {
		return nil, fmt.Errorf("voyage: at least one text is required")
	}

	reqBody := Request{
		Input: texts,
		Model: Model,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, Endpoint, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call Voyage API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Voyage API returned status: %d, body: %s", resp.StatusCode, string(raw))
	}

	var parsedResp Response
	if err := json.NewDecoder(resp.Body).Decode(&parsedResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Voyage response: %w", err)
	}

	embeddings := make([][]float32, len(parsedResp.Data))
	for i, item := range parsedResp.Data {
		embeddings[i] = item.Embedding
	}

	return embeddings, nil
}
