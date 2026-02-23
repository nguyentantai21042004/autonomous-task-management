package memos

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Client is the HTTP wrapper for the Memos REST API.
type Client struct {
	baseURL     string
	accessToken string
	httpClient  *http.Client
}

// NewClient creates a new Memos HTTP client.
func NewClient(baseURL, accessToken string) *Client {
	return &Client{
		baseURL:     baseURL,
		accessToken: accessToken,
		httpClient:  &http.Client{},
	}
}

// CreateMemo creates a new memo via POST /api/v1/memos.
func (c *Client) CreateMemo(ctx context.Context, req CreateMemoRequest) (*Memo, error) {
	url := fmt.Sprintf("%s/api/v1/memos", c.baseURL)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create memo request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to build create memo request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call memos create API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("memos API create error %d: %s", resp.StatusCode, string(raw))
	}

	var memo Memo
	if err := json.NewDecoder(resp.Body).Decode(&memo); err != nil {
		return nil, fmt.Errorf("failed to decode memos create response: %w", err)
	}
	return &memo, nil
}

// GetMemo fetches a single memo by its ID.
func (c *Client) GetMemo(ctx context.Context, id string) (*Memo, error) {
	url := fmt.Sprintf("%s/api/v1/memos/%s", c.baseURL, id)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build get memo request: %w", err)
	}
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call memos get API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("memos API get error %d: %s", resp.StatusCode, string(raw))
	}

	var memo Memo
	if err := json.NewDecoder(resp.Body).Decode(&memo); err != nil {
		return nil, fmt.Errorf("failed to decode memos get response: %w", err)
	}
	return &memo, nil
}

// ListMemos lists memos with an optional tag filter.
func (c *Client) ListMemos(ctx context.Context, tag string, limit, offset int) ([]Memo, error) {
	url := fmt.Sprintf("%s/api/v1/memos?pageSize=%d", c.baseURL, limit)
	if tag != "" {
		url += fmt.Sprintf("&filter=tag='%s'", tag)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build list memos request: %w", err)
	}
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call memos list API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("memos API list error %d: %s", resp.StatusCode, string(raw))
	}

	var listResp struct {
		Memos []Memo `json:"memos"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode memos list response: %w", err)
	}
	return listResp.Memos, nil
}

// ---- Request/Response types scoped to this package ----

// CreateMemoRequest is the body for POST /api/v1/memos.
type CreateMemoRequest struct {
	Content    string `json:"content"`
	Visibility string `json:"visibility"`
}

// Memo is the Memos API memo object.
type Memo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	UID        string `json:"uid"`
	Content    string `json:"content"`
	Visibility string `json:"visibility"`
	CreateTime string `json:"createTime"`
	UpdateTime string `json:"updateTime"`
}
