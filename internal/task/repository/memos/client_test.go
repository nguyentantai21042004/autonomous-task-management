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

	mux.HandleFunc("/api/v1/memos/uid-1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			var req memos.UpdateMemoRequest
			json.NewDecoder(r.Body).Decode(&req)
			m := memos.Memo{ID: "1", UID: "uid-1", Content: req.Content}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(m)
			return
		}
		if r.Method == http.MethodGet {
			m := memos.Memo{ID: "1", UID: "uid-1", Content: "Got memo"}
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
		res, err := client.UpdateMemo(ctx, "uid-1", memos.UpdateMemoRequest{
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
		res, err := client.GetMemo(ctx, "uid-1")
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
		_, err := badClient.GetMemo(ctx, "uid-1")
		if err == nil {
			t.Errorf("expected connection refused error")
		}
	})
}
