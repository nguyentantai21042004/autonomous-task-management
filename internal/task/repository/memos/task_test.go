package memos_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"autonomous-task-management/internal/task/repository"
	"autonomous-task-management/internal/task/repository/memos"
)

type mockLogger struct{}

func (m *mockLogger) Debug(ctx context.Context, args ...any)                  {}
func (m *mockLogger) Debugf(ctx context.Context, format string, args ...any)  {}
func (m *mockLogger) Info(ctx context.Context, args ...any)                   {}
func (m *mockLogger) Infof(ctx context.Context, format string, args ...any)   {}
func (m *mockLogger) Warn(ctx context.Context, args ...any)                   {}
func (m *mockLogger) Warnf(ctx context.Context, format string, args ...any)   {}
func (m *mockLogger) Error(ctx context.Context, args ...any)                  {}
func (m *mockLogger) Errorf(ctx context.Context, format string, args ...any)  {}
func (m *mockLogger) DPanic(ctx context.Context, args ...any)                 {}
func (m *mockLogger) DPanicf(ctx context.Context, format string, args ...any) {}
func (m *mockLogger) Panic(ctx context.Context, args ...any)                  {}
func (m *mockLogger) Panicf(ctx context.Context, format string, args ...any)  {}
func (m *mockLogger) Fatal(ctx context.Context, args ...any)                  {}
func (m *mockLogger) Fatalf(ctx context.Context, format string, args ...any)  {}

func TestMemosRepository(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/memos", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var m memos.Memo
			json.NewDecoder(r.Body).Decode(&m)
			if strings.Contains(m.Content, "error") {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			m.ID = "1"
			m.UID = "uid-1"
			m.Name = "memos/uid-1"
			m.CreateTime = time.Now().String()
			m.UpdateTime = time.Now().String()
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(m)
			return
		}
		if r.Method == http.MethodGet {
			m := memos.Memo{ID: "1", UID: "uid-1", Name: "memos/uid-1", Content: "# Test Task\n\nSome body"}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{"memos": []memos.Memo{m}})
			return
		}
	})

	mux.HandleFunc("/api/v1/memos/uid-1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch {
			var req memos.UpdateMemoRequest
			json.NewDecoder(r.Body).Decode(&req)
			if strings.Contains(req.Content, "error") {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			m := memos.Memo{ID: "1", UID: "uid-1", Name: "memos/uid-1", Content: req.Content}
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
			m := memos.Memo{ID: "1", UID: "uid-1", Name: "memos/uid-1", Content: "# Old Task"}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(m)
			return
		}
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	client := memos.NewClient(ts.URL, "test-token")
	repo := memos.New(client, "http://memos.local", &mockLogger{})
	ctx := context.Background()

	t.Run("CreateTask", func(t *testing.T) {
		opts := repository.CreateTaskOptions{
			Content:    "# Test Task\n\nBody",
			Visibility: "PRIVATE",
		}
		task, err := repo.CreateTask(ctx, opts)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if task.UID != "uid-1" {
			t.Errorf("unexpected UID: %s", task.UID)
		}

		// Check markdown generation fields
		if !strings.Contains(task.Content, "# Test Task") {
			t.Errorf("missing title in content")
		}

		// Error path
		optsFail := repository.CreateTaskOptions{Content: "error"}
		_, err = repo.CreateTask(ctx, optsFail)
		if err == nil {
			t.Errorf("expected error")
		}
	})

	t.Run("CreateTasksBatch", func(t *testing.T) {
		opts := []repository.CreateTaskOptions{
			{Content: "# Task 1"},
			{Content: "# Task 2"},
		}
		tasks, err := repo.CreateTasksBatch(ctx, opts)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(tasks) != 2 {
			t.Errorf("expected 2 tasks, got %d", len(tasks))
		}

		// Error path returns empty array instead of panic
		optsFail := []repository.CreateTaskOptions{{Content: "error"}}
		tasksFail, err := repo.CreateTasksBatch(ctx, optsFail)
		if err != nil {
			t.Fatalf("unexpected batch error: %v", err)
		}
		if len(tasksFail) != 0 {
			t.Errorf("expected 0 completed tasks for batch failure, got %d", len(tasksFail))
		}
	})

	t.Run("GetTask", func(t *testing.T) {
		task, err := repo.GetTask(ctx, "uid-1")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if task.UID != "uid-1" {
			t.Errorf("unexpected UID: %s", task.UID)
		}
	})

	t.Run("UpdateTask", func(t *testing.T) {
		err := repo.UpdateTask(ctx, "uid-1", "new content")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		err = repo.UpdateTask(ctx, "uid-1", "error content")
		if err == nil {
			t.Errorf("expected update error")
		}
	})

	t.Run("ListTasks", func(t *testing.T) {
		tasks, err := repo.ListTasks(ctx, repository.ListTasksOptions{Tag: "p1"})
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(tasks) != 1 || tasks[0].UID != "uid-1" {
			t.Errorf("unexpected tasks: %+v", tasks)
		}
	})
}
