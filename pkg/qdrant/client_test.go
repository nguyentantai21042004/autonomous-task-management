package qdrant_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"autonomous-task-management/pkg/qdrant"
)

func TestQdrantClient(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Routing based on URL path and Method
		path := r.URL.Path

		if r.Method == http.MethodPut && strings.HasSuffix(path, "/points") {
			var req qdrant.UpsertPointsRequest
			json.NewDecoder(r.Body).Decode(&req)
			if len(req.Points) > 0 {
				payload := req.Points[0].Payload
				if val, ok := payload["cause_500"]; ok && val == true {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method == http.MethodPut && strings.Contains(path, "/collections/") {
			w.WriteHeader(http.StatusCreated)
			return
		}

		if r.Method == http.MethodPost && strings.Contains(path, "/points/search") {
			var req qdrant.SearchRequest
			json.NewDecoder(r.Body).Decode(&req)
			if req.Limit == 999 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"result": [
					{
						"id": "123",
						"version": 1,
						"score": 0.95,
						"payload": {"key": "value"}
					}
				],
				"status": "ok",
				"time": 0.05
			}`))
			return
		}

		if r.Method == http.MethodPost && strings.Contains(path, "/points/delete") {
			var req qdrant.DeletePointsRequest
			json.NewDecoder(r.Body).Decode(&req)
			if len(req.Points) > 0 && req.Points[0] == "cause_500" {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	client := qdrant.NewClient(ts.URL)

	t.Run("CreateCollection", func(t *testing.T) {
		err := client.CreateCollection(context.Background(), qdrant.CreateCollectionRequest{
			Name: "test_col",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("UpsertPoints Success", func(t *testing.T) {
		err := client.UpsertPoints(context.Background(), "test_col", qdrant.UpsertPointsRequest{
			Points: []qdrant.Point{
				{
					ID:      "123",
					Payload: map[string]interface{}{"key": "val"},
					Vector:  []float32{0.1, 0.2},
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("UpsertPoints Error", func(t *testing.T) {
		err := client.UpsertPoints(context.Background(), "test_col", qdrant.UpsertPointsRequest{
			Points: []qdrant.Point{
				{
					ID:      "123",
					Payload: map[string]interface{}{"cause_500": true},
					Vector:  []float32{0.1, 0.2},
				},
			},
		})
		if err == nil {
			t.Fatalf("expected error from 500 response")
		}
	})

	t.Run("SearchPoints Success", func(t *testing.T) {
		resp, err := client.SearchPoints(context.Background(), "test_col", qdrant.SearchRequest{
			Limit: 10,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp.Result) != 1 || resp.Result[0].ID != "123" {
			t.Errorf("unexpected search results: %v", resp)
		}
	})

	t.Run("SearchPoints Error", func(t *testing.T) {
		_, err := client.SearchPoints(context.Background(), "test_col", qdrant.SearchRequest{
			Limit: 999,
		})
		if err == nil {
			t.Fatalf("expected error from 500 response")
		}
	})

	t.Run("DeletePoints Success", func(t *testing.T) {
		err := client.DeletePoints(context.Background(), "test_col", []string{"123", "456"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("DeletePoints Error", func(t *testing.T) {
		err := client.DeletePoints(context.Background(), "test_col", []string{"cause_500"})
		if err == nil {
			t.Fatalf("expected error from 500 response")
		}
	})

	t.Run("Context Cancelation Error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately

		err := client.CreateCollection(ctx, qdrant.CreateCollectionRequest{Name: "test"})
		if err == nil {
			t.Errorf("expected error on canceled context")
		}

		_, err = client.SearchPoints(ctx, "test", qdrant.SearchRequest{})
		if err == nil {
			t.Errorf("expected error on canceled context")
		}
	})
}
