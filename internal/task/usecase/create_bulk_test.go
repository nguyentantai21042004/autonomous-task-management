package usecase_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"autonomous-task-management/internal/model"
	"autonomous-task-management/internal/task"
	"autonomous-task-management/internal/task/repository"
	"autonomous-task-management/internal/task/usecase"
	"autonomous-task-management/pkg/datemath"
	"autonomous-task-management/pkg/gcalendar"
	"autonomous-task-management/pkg/gemini"
)

// mock dependencies

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

type mockMemosRepo struct {
	fail bool
}

func (m *mockMemosRepo) CreateTask(ctx context.Context, opt repository.CreateTaskOptions) (model.Task, error) {
	if m.fail {
		return model.Task{}, errors.New("db error")
	}
	return model.Task{ID: "memos/1", UID: "uid-1", Content: opt.Content, MemoURL: "http://local/m/uid-1"}, nil
}

func (m *mockMemosRepo) CreateTasksBatch(ctx context.Context, opts []repository.CreateTaskOptions) ([]model.Task, error) {
	if m.fail {
		return nil, errors.New("db error batch")
	}
	var res []model.Task
	for i, o := range opts {
		if o.Content == "error" {
			continue
		}
		res = append(res, model.Task{ID: "memos/1", UID: "uid-1", Content: o.Content})
		_ = i
	}
	return res, nil
}

func (m *mockMemosRepo) GetTask(ctx context.Context, id string) (model.Task, error) {
	return model.Task{}, nil
}

func (m *mockMemosRepo) ListTasks(ctx context.Context, opt repository.ListTasksOptions) ([]model.Task, error) {
	return nil, nil
}

func (m *mockMemosRepo) UpdateTask(ctx context.Context, id string, content string) error {
	return nil
}

type mockVectorRepo struct {
	fail bool
}

func (m *mockVectorRepo) EmbedTask(ctx context.Context, task model.Task) error {
	if m.fail {
		return errors.New("vector embed error")
	}
	return nil
}

func (m *mockVectorRepo) SearchTasks(ctx context.Context, opt repository.SearchTasksOptions) ([]repository.SearchResult, error) {
	return nil, nil
}

func (m *mockVectorRepo) SearchTasksWithFilter(ctx context.Context, opt repository.SearchTasksOptions) ([]repository.SearchResult, error) {
	return nil, nil
}

func (m *mockVectorRepo) DeleteTask(ctx context.Context, taskID string) error {
	return nil
}

type mockCalendarClient struct {
	fail bool
}

func (m *mockCalendarClient) CreateEvent(ctx context.Context, req gcalendar.CreateEventRequest) (*gcalendar.Event, error) {
	if m.fail {
		return nil, errors.New("cal error")
	}
	return &gcalendar.Event{HtmlLink: "http://cal.link"}, nil
}

func TestCreateBulk(t *testing.T) {
	// Setup LLM Mock
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req gemini.GenerateRequest
		json.NewDecoder(r.Body).Decode(&req)
		prompt := req.Contents[0].Parts[0].Text

		if strings.Contains(prompt, "error_llm_json") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"candidates": [
					{
						"content": {
							"parts": [
								{ "text": "invalid json format" }
							]
						}
					}
				]
			}`))
			return
		}

		if strings.Contains(prompt, "error_llm_500") {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Success flow
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"candidates": [
				{
					"content": {
						"parts": [
							{
								"text": "[\n  {\n    \"title\": \"Buy milk\",\n    \"description\": \"At the grocery store\",\n    \"date\": \"2024-05-10\",\n    \"time\": \"14:00\",\n    \"priority\": \"1\",\n    \"tags\": [\"shopping\", \"errands\"]\n  }\n]"
							}
						]
					}
				}
			]
		}`))
	}))
	defer ts.Close()

	llmClient := gemini.NewClient("test-key")
	llmClient.SetAPIURL(ts.URL)
	dateMath, _ := datemath.NewParser("Asia/Ho_Chi_Minh")

	t.Run("Success Path", func(t *testing.T) {
		memosRepo := &mockMemosRepo{}
		vectorRepo := &mockVectorRepo{}
		calClient := &mockCalendarClient{}

		uc := usecase.New(&mockLogger{}, llmClient, calClient, memosRepo, vectorRepo, dateMath, "Asia/Ho_Chi_Minh", "http://memos")

		out, err := uc.CreateBulk(context.Background(), model.Scope{UserID: "u1"}, task.CreateBulkInput{RawText: "Buy milk tomorrow at 2pm"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(out.Tasks) != 1 {
			t.Errorf("expected 1 task, got %d", len(out.Tasks))
		}
	})

	t.Run("LLM Failure Path", func(t *testing.T) {
		memosRepo := &mockMemosRepo{}
		vectorRepo := &mockVectorRepo{}
		calClient := &mockCalendarClient{}

		uc := usecase.New(&mockLogger{}, llmClient, calClient, memosRepo, vectorRepo, dateMath, "Asia/Ho_Chi_Minh", "http://memos")

		_, err := uc.CreateBulk(context.Background(), model.Scope{UserID: "u1"}, task.CreateBulkInput{RawText: "error_llm_500"})
		if err == nil {
			t.Errorf("expected llm 500 error")
		}

		_, err = uc.CreateBulk(context.Background(), model.Scope{UserID: "u1"}, task.CreateBulkInput{RawText: "error_llm_json"})
		if err == nil {
			t.Errorf("expected llm parsing error")
		}
	})

	t.Run("Memos DB Failure Path", func(t *testing.T) {
		memosRepo := &mockMemosRepo{fail: true}
		vectorRepo := &mockVectorRepo{}
		calClient := &mockCalendarClient{}

		uc := usecase.New(&mockLogger{}, llmClient, calClient, memosRepo, vectorRepo, dateMath, "Asia/Ho_Chi_Minh", "http://memos")

		out, err := uc.CreateBulk(context.Background(), model.Scope{UserID: "u1"}, task.CreateBulkInput{RawText: "Buy milk"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(out.Tasks) != 0 {
			t.Errorf("expected 0 tasks created on db failure, got %d", len(out.Tasks))
		}
	})

	t.Run("Vector Embed Failure path (Graceful degradation)", func(t *testing.T) {
		memosRepo := &mockMemosRepo{}
		vectorRepo := &mockVectorRepo{fail: true}
		calClient := &mockCalendarClient{}

		uc := usecase.New(&mockLogger{}, llmClient, calClient, memosRepo, vectorRepo, dateMath, "Asia/Ho_Chi_Minh", "http://memos")

		out, err := uc.CreateBulk(context.Background(), model.Scope{UserID: "u1"}, task.CreateBulkInput{RawText: "Buy milk"})
		if err != nil { // Still returns tasks
			t.Errorf("unexpected error on vector fail (should gracefully degrade): %v", err)
		}
		if len(out.Tasks) != 1 {
			t.Errorf("expected 1 task")
		}
	})

	t.Run("Calendar Failure path (Graceful degradation)", func(t *testing.T) {
		memosRepo := &mockMemosRepo{}
		vectorRepo := &mockVectorRepo{}
		calClient := &mockCalendarClient{fail: true}

		uc := usecase.New(&mockLogger{}, llmClient, calClient, memosRepo, vectorRepo, dateMath, "Asia/Ho_Chi_Minh", "http://memos")

		out, err := uc.CreateBulk(context.Background(), model.Scope{UserID: "u1"}, task.CreateBulkInput{RawText: "Buy milk"})
		if err != nil { // Still returns tasks
			t.Errorf("unexpected error on calendar fail (should gracefully degrade): %v", err)
		}
		if len(out.Tasks) != 1 {
			t.Errorf("expected 1 task")
		}
	})
}
