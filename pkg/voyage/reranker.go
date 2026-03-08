package voyage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const DefaultRerankModel = "rerank-2"

// RerankRequest la request body cho Voyage rerank API.
type RerankRequest struct {
	Model     string   `json:"model"`
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
	TopK      int      `json:"top_k,omitempty"`
}

// RerankResult chua ket qua reranking cho mot document.
type RerankResult struct {
	Index          int     `json:"index"`           // Vi tri trong documents goc
	RelevanceScore float64 `json:"relevance_score"` // Score cang cao cang lien quan
}

// RerankResponse la response tu Voyage rerank API.
type RerankResponse struct {
	Object string         `json:"object"`
	Model  string         `json:"model"`
	Data   []RerankResult `json:"data"`
	Usage  UsageInfo      `json:"usage"`
}

// Reranker thuc hien cross-encoder reranking bang Voyage AI.
//
// Khac voi bi-encoder embedding (chi so sanh vector doc lap):
//   - Cross-encoder doc query + document CUNG LUC → hieu relationship sau hon
//   - Chinh xac hon nhung cham hon (chay sau bước retrieve)
//
// Pipeline khuyen nghi:
//  1. Hybrid search → top 20 candidates (recall phase)
//  2. Reranker.Rerank() → top 5 (precision phase)
//  3. Dua top 5 vao LLM context
type Reranker struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// NewReranker tao Reranker moi su dung Voyage AI rerank-2.
func NewReranker(apiKey string) (*Reranker, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("voyage API key is required for reranker")
	}
	return &Reranker{
		apiKey:     apiKey,
		baseURL:    DefaultBaseURL,
		model:      DefaultRerankModel,
		httpClient: &http.Client{},
	}, nil
}

// WithRerankModel thay the model mac dinh.
func (r *Reranker) WithRerankModel(model string) *Reranker {
	r.model = model
	return r
}

// WithRerankBaseURL thay the base URL (dung cho testing).
func (r *Reranker) WithRerankBaseURL(baseURL string) *Reranker {
	r.baseURL = baseURL
	return r
}

// Rerank sap xep lai documents theo muc do lien quan voi query.
// Tra ve top-K results theo thu tu giam dan cua relevance_score.
// topK = 0 → tra ve tat ca results.
func (r *Reranker) Rerank(ctx context.Context, query string, documents []string, topK int) ([]RerankResult, error) {
	if len(documents) == 0 {
		return []RerankResult{}, nil
	}

	reqBody := RerankRequest{
		Model:     r.model,
		Query:     query,
		Documents: documents,
		TopK:      topK,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("reranker: failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/rerank", r.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("reranker: failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", r.apiKey))

	resp, err := r.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("reranker: API call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if jsonErr := json.NewDecoder(resp.Body).Decode(&errResp); jsonErr == nil {
			return nil, fmt.Errorf("reranker: API error (%d): %s", resp.StatusCode, errResp.Error.Message)
		}
		return nil, fmt.Errorf("reranker: API error: %d", resp.StatusCode)
	}

	var rerankResp RerankResponse
	if err := json.NewDecoder(resp.Body).Decode(&rerankResp); err != nil {
		return nil, fmt.Errorf("reranker: failed to decode response: %w", err)
	}

	return rerankResp.Data, nil
}
