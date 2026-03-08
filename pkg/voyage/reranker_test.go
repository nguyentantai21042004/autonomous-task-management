package voyage

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newTestReranker(t *testing.T, serverURL string) *Reranker {
	t.Helper()
	r, err := NewReranker("test-api-key")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	r.baseURL = serverURL
	return r
}

func TestNewReranker_EmptyAPIKey(t *testing.T) {
	_, err := NewReranker("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API key")
}

func TestReranker_HappyPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/rerank", r.URL.Path)
		assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

		// Verify request body
		var req RerankRequest
		json.NewDecoder(r.Body).Decode(&req)
		assert.Equal(t, "PR #123 deadline", req.Query)
		assert.Len(t, req.Documents, 3)

		// Tra ve mock response: doc 1 co score cao nhat
		resp := RerankResponse{
			Object: "list",
			Model:  DefaultRerankModel,
			Data: []RerankResult{
				{Index: 1, RelevanceScore: 0.95},
				{Index: 0, RelevanceScore: 0.72},
				{Index: 2, RelevanceScore: 0.31},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	r := newTestReranker(t, server.URL)
	results, err := r.Rerank(context.Background(), "PR #123 deadline", []string{
		"Deploy staging server",
		"Review PR #123 backend",
		"Meeting with client",
	}, 3)

	assert.NoError(t, err)
	assert.Len(t, results, 3)
	// Doc 1 (Review PR #123) phai dung dau
	assert.Equal(t, 1, results[0].Index)
	assert.InDelta(t, 0.95, results[0].RelevanceScore, 0.01)
}

func TestReranker_EmptyDocuments(t *testing.T) {
	r, _ := NewReranker("key")
	results, err := r.Rerank(context.Background(), "query", []string{}, 5)

	assert.NoError(t, err)
	assert.Empty(t, results)
}

func TestReranker_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error: struct {
				Message string `json:"message"`
				Type    string `json:"type"`
			}{Message: "Invalid API key", Type: "auth_error"},
		})
	}))
	defer server.Close()

	r := newTestReranker(t, server.URL)
	_, err := r.Rerank(context.Background(), "query", []string{"doc1"}, 1)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

func TestReranker_TopKRespected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req RerankRequest
		json.NewDecoder(r.Body).Decode(&req)
		assert.Equal(t, 3, req.TopK) // Verify TopK duoc gui dung

		json.NewEncoder(w).Encode(RerankResponse{
			Data: []RerankResult{
				{Index: 0, RelevanceScore: 0.9},
				{Index: 1, RelevanceScore: 0.8},
				{Index: 2, RelevanceScore: 0.7},
			},
		})
	}))
	defer server.Close()

	r := newTestReranker(t, server.URL)
	results, err := r.Rerank(context.Background(), "query", []string{"a", "b", "c", "d", "e"}, 3)

	assert.NoError(t, err)
	assert.Len(t, results, 3)
}

func TestReranker_WithRerankModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req RerankRequest
		json.NewDecoder(r.Body).Decode(&req)
		assert.Equal(t, "rerank-lite-1", req.Model)
		json.NewEncoder(w).Encode(RerankResponse{Data: []RerankResult{{Index: 0, RelevanceScore: 0.5}}})
	}))
	defer server.Close()

	r := newTestReranker(t, server.URL)
	r.WithRerankModel("rerank-lite-1")
	_, err := r.Rerank(context.Background(), "q", []string{"doc"}, 1)
	assert.NoError(t, err)
}
