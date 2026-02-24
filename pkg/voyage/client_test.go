package voyage_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"autonomous-task-management/pkg/voyage"
)

func TestVoyageClient(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-voyage-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		var req struct {
			Input []string `json:"input"`
			Model string   `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if len(req.Input) > 0 && req.Input[0] == "cause_500" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Success flow
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"data": [
				{
					"embedding": [0.1, 0.2, 0.3],
					"index": 0
				}
			]
		}`))
	}))
	defer ts.Close()

	client, _ := voyage.New("test-voyage-key")
	client.WithBaseURL(ts.URL).WithModel("custom-model")

	t.Run("Success Flow", func(t *testing.T) {
		emb, err := client.Embed(context.Background(), []string{"Hello world"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(emb) != 1 || len(emb[0]) != 3 {
			t.Fatalf("expected 1 embed with 3 dims, got len=%d", len(emb))
		}
		if emb[0][0] != 0.1 || emb[0][1] != 0.2 || emb[0][2] != 0.3 {
			t.Errorf("unexpected embedding values: %v", emb[0])
		}
	})

	t.Run("Server Error Flow", func(t *testing.T) {
		_, err := client.Embed(context.Background(), []string{"cause_500"})
		if err == nil {
			t.Fatalf("expected error from 500 response")
		}
	})

	t.Run("Unauthorized Error Flow", func(t *testing.T) {
		badClient, _ := voyage.New("bad-key")
		badClient.WithBaseURL(ts.URL)
		_, err := badClient.Embed(context.Background(), []string{"Hello world"})
		if err == nil || !strings.Contains(err.Error(), "401") {
			t.Fatalf("expected 401 error, got %v", err)
		}
	})
}
