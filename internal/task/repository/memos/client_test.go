package memos_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"autonomous-task-management/internal/task/repository/memos"
)

func TestMemosClient(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/memos", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var m memos.Memo
			json.NewDecoder(r.Body).Decode(&m)
			m.ID = "1"
			m.UID = "uid-1"
			m.CreateTime = time.Now().String()
			m.UpdateTime = time.Now().String()
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(m)
			return
		}
		if r.Method == http.MethodGet {
			filter := r.URL.Query().Get("filter")
			if strings.Contains(filter, "error") {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			m := memos.Memo{ID: "1", UID: "uid-1", Content: "List item"}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{"memos": []memos.Memo{m}})
			return
		}
	})

	// Handler for resource name "memos/uid-1" → /api/v1/memos/uid-1
	mux.HandleFunc("/api/v1/memos/uid-1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			var req memos.UpdateMemoRequest
			json.NewDecoder(r.Body).Decode(&req)
			m := memos.Memo{ID: "1", UID: "uid-1", Name: "memos/uid-1", Content: req.Content}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(m)
			return
		}
		if r.Method == http.MethodGet {
			m := memos.Memo{ID: "1", UID: "uid-1", Name: "memos/uid-1", Content: "Got memo"}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(m)
			return
		}
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	client := memos.NewClient(ts.URL, "test-token")
	ctx := context.Background()

	t.Run("CreateMemo", func(t *testing.T) {
		res, err := client.CreateMemo(ctx, memos.CreateMemoRequest{Content: "Hello"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.ID != "1" || res.UID != "uid-1" {
			t.Errorf("unexpected memo response: %+v", res)
		}
	})

	t.Run("UpdateMemo", func(t *testing.T) {
		res, err := client.UpdateMemo(ctx, "memos/uid-1", memos.UpdateMemoRequest{
			Content:    "Updated",
			UpdateMask: "content",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Content != "Updated" {
			t.Errorf("unexpected memo content: %s", res.Content)
		}
	})

	t.Run("GetMemo", func(t *testing.T) {
		res, err := client.GetMemo(ctx, "memos/uid-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Content != "Got memo" {
			t.Errorf("unexpected content: %s", res.Content)
		}
	})

	t.Run("ListMemos", func(t *testing.T) {
		res, err := client.ListMemos(ctx, "test", 10, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res) != 1 || res[0].Content != "List item" {
			t.Errorf("unexpected list result: %+v", res)
		}

		// test error prop
		_, err = client.ListMemos(ctx, "error", 10, 0)
		if err == nil {
			t.Errorf("expected error from filter")
		}
	})

	// Server Down
	t.Run("Server Down", func(t *testing.T) {
		badClient := memos.NewClient("http://localhost:59999", "token")
		_, err := badClient.GetMemo(ctx, "memos/uid-1")
		if err == nil {
			t.Errorf("expected connection refused error")
		}
	})
}

// TestGetMemo_ResourceNamePath verifies that GetMemo builds the correct URL
// when called with resource name format "memos/{uid}" (the format used in production).
// This was BUG #1: the old code built /api/v1/memos/memos/{uid} (doubled path).
func TestGetMemo_ResourceNamePath(t *testing.T) {
	var capturedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		m := memos.Memo{ID: "1", UID: "abc123", Name: "memos/abc123", Content: "test"}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(m)
	}))
	defer ts.Close()

	client := memos.NewClient(ts.URL, "test-token")

	// Call with resource name format (as stored in model.Task.ID)
	_, err := client.GetMemo(context.Background(), "memos/abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "/api/v1/memos/abc123"
	if capturedPath != expected {
		t.Errorf("GetMemo built wrong URL path: got %q, want %q", capturedPath, expected)
	}
}

// TestUpdateMemo_ResourceNamePath verifies the same fix for UpdateMemo.
func TestUpdateMemo_ResourceNamePath(t *testing.T) {
	var capturedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		m := memos.Memo{ID: "1", UID: "abc123", Name: "memos/abc123", Content: "updated"}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(m)
	}))
	defer ts.Close()

	client := memos.NewClient(ts.URL, "test-token")

	_, err := client.UpdateMemo(context.Background(), "memos/abc123", memos.UpdateMemoRequest{
		Content:    "updated",
		UpdateMask: "content",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "/api/v1/memos/abc123"
	if capturedPath != expected {
		t.Errorf("UpdateMemo built wrong URL path: got %q, want %q", capturedPath, expected)
	}
}

// TestGetMemo_404ReturnsError ensures a 404 from Memos API is propagated as
// an error containing "404", which the self-healing logic depends on.
func TestGetMemo_404ReturnsError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"not found"}`))
	}))
	defer ts.Close()

	client := memos.NewClient(ts.URL, "test-token")
	_, err := client.GetMemo(context.Background(), "memos/nonexistent")
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error should contain '404', got: %v", err)
	}
}
