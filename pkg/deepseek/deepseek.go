package deepseek

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// deepseekImpl implements IDeepSeek interface
type deepseekImpl struct {
	apiKey  string
	model   string
	baseURL string
	client  *http.Client
}

func newDeepSeekImpl(cfg Config) *deepseekImpl {
	if cfg.Model == "" {
		cfg.Model = DefaultModel
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}

	return &deepseekImpl{
		apiKey:  cfg.APIKey,
		model:   cfg.Model,
		baseURL: cfg.BaseURL,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// GenerateContent sends a request to DeepSeek API
func (c *deepseekImpl) GenerateContent(ctx context.Context, req *Request) (*Response, error) {
	// Set model if not specified
	if req.Model == "" {
		req.Model = c.model
	}

	// Marshal request
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("pkg: failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("pkg: failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	// Send request
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("pkg: failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("pkg: failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err != nil {
			return nil, fmt.Errorf("pkg: API error %d: %s", resp.StatusCode, string(respBody))
		}
		return nil, fmt.Errorf("pkg: API error %d: %s", resp.StatusCode, errResp.Error.Message)
	}

	// Parse response
	var result Response
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("pkg: failed to parse response: %w", err)
	}

	return &result, nil
}
