package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const defaultModel = "gemini-1.5-flash"

// Client is the Gemini Generative Language API client.
type Client struct {
	apiKey     string
	apiURL     string
	model      string
	httpClient *http.Client
}

// NewClient creates a new Gemini API client with the given API key.
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:     apiKey,
		apiURL:     "https://generativelanguage.googleapis.com/v1",
		model:      defaultModel,
		httpClient: &http.Client{},
	}
}

// GenerateContent sends a content generation request to the Gemini API.
func (c *Client) GenerateContent(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", c.apiURL, c.model, c.apiKey)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call gemini API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gemini API error %d: %s", resp.StatusCode, string(raw))
	}

	var result GenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode gemini response: %w", err)
	}

	return &result, nil
}
