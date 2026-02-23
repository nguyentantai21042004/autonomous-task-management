package qdrant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// Client is the Qdrant HTTP API client.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new Qdrant client.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{},
	}
}

// CreateCollection creates a new collection with the given configuration.
func (c *Client) CreateCollection(ctx context.Context, req CreateCollectionRequest) error {
	url := fmt.Sprintf("%s/collections/%s", c.baseURL, req.Name)

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to call qdrant API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("qdrant API error: %d", resp.StatusCode)
	}

	return nil
}

// UpsertPoints inserts or updates points (vectors) in a collection.
func (c *Client) UpsertPoints(ctx context.Context, collectionName string, req UpsertPointsRequest) error {
	url := fmt.Sprintf("%s/collections/%s/points", c.baseURL, collectionName)

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to call qdrant API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("qdrant API error: %d", resp.StatusCode)
	}

	return nil
}

// SearchPoints performs semantic search in a collection.
func (c *Client) SearchPoints(ctx context.Context, collectionName string, req SearchRequest) (*SearchResponse, error) {
	url := fmt.Sprintf("%s/collections/%s/points/search", c.baseURL, collectionName)

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
		return nil, fmt.Errorf("failed to call qdrant API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("qdrant API error: %d", resp.StatusCode)
	}

	var result SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// DeletePoints deletes points by IDs.
func (c *Client) DeletePoints(ctx context.Context, collectionName string, ids []string) error {
	url := fmt.Sprintf("%s/collections/%s/points/delete", c.baseURL, collectionName)

	req := DeletePointsRequest{
		Points: ids,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to call qdrant API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("qdrant API error: %d", resp.StatusCode)
	}

	return nil
}
