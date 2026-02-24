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
	"autonomous-task-management/pkg/gemini"
)

type ragMemosRepo struct {
	mockMemosRepo
	getTaskFunc func(id string) (model.Task, error)
}

func (m *ragMemosRepo) GetTask(ctx context.Context, id string) (model.Task, error) {
	if m.getTaskFunc != nil {
		return m.getTaskFunc(id)
	}
	return model.Task{}, nil
}

type ragVectorRepo struct {
	mockVectorRepo
	searchFunc func(opt repository.SearchTasksOptions) ([]repository.SearchResult, error)
}

func (m *ragVectorRepo) SearchTasks(ctx context.Context, opt repository.SearchTasksOptions) ([]repository.SearchResult, error) {
	if m.searchFunc != nil {
		return m.searchFunc(opt)
	}
	return nil, nil
}

func TestAnswerQuery(t *testing.T) {
	// Setup LLM Mock for RAG answering
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req gemini.GenerateRequest
		json.NewDecoder(r.Body).Decode(&req)
		prompt := req.Contents[0].Parts[0].Text

		if strings.Contains(prompt, "error_llm_500") {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"candidates": [
				{
					"content": {
						"parts": [
							{ "text": "This is the generated answer based on context." }
						]
					}
				}
			]
		}`))
	}))
	defer ts.Close()

	llmClient := gemini.NewClient("test-key")
	llmClient.SetAPIURL(ts.URL)

	emptyMux := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"candidates": []
		}`))
	}))
	defer emptyMux.Close()
	llmEmpty := gemini.NewClient("test-key")
	llmEmpty.SetAPIURL(emptyMux.URL)

	dateMath, _ := datemath.NewParser("Asia/Ho_Chi_Minh")

	t.Run("Empty Query Error", func(t *testing.T) {
		uc := usecase.New(&mockLogger{}, llmClient, &mockCalendarClient{}, &ragMemosRepo{}, &ragVectorRepo{}, dateMath, "Asia/Ho_Chi_Minh", "http://memos")
		_, err := uc.AnswerQuery(context.Background(), model.Scope{}, task.QueryInput{Query: ""})
		if !errors.Is(err, task.ErrEmptyQuery) {
			t.Errorf("expected ErrEmptyQuery, got %v", err)
		}
	})

	t.Run("Vector Search Error", func(t *testing.T) {
		vRepo := &ragVectorRepo{
			searchFunc: func(opt repository.SearchTasksOptions) ([]repository.SearchResult, error) {
				return nil, errors.New("search fail")
			},
		}
		uc := usecase.New(&mockLogger{}, llmClient, &mockCalendarClient{}, &ragMemosRepo{}, vRepo, dateMath, "Asia/Ho_Chi_Minh", "http://memos")
		_, err := uc.AnswerQuery(context.Background(), model.Scope{}, task.QueryInput{Query: "test"})
		if err == nil {
			t.Errorf("expected vector search error")
		}
	})

	t.Run("No Search Results", func(t *testing.T) {
		vRepo := &ragVectorRepo{
			searchFunc: func(opt repository.SearchTasksOptions) ([]repository.SearchResult, error) {
				return []repository.SearchResult{}, nil
			},
		}
		uc := usecase.New(&mockLogger{}, llmClient, &mockCalendarClient{}, &ragMemosRepo{}, vRepo, dateMath, "Asia/Ho_Chi_Minh", "http://memos")
		out, err := uc.AnswerQuery(context.Background(), model.Scope{}, task.QueryInput{Query: "test"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.SourceCount != 0 {
			t.Errorf("expected 0 sources, got %d", out.SourceCount)
		}
	})

	t.Run("Successful RAG Workflow", func(t *testing.T) {
		vRepo := &ragVectorRepo{
			searchFunc: func(opt repository.SearchTasksOptions) ([]repository.SearchResult, error) {
				return []repository.SearchResult{
					{MemoID: "memo/1"},
					{MemoID: "memo/2"}, // this one will fail to fetch
				}, nil
			},
		}
		mRepo := &ragMemosRepo{
			getTaskFunc: func(id string) (model.Task, error) {
				if id == "memo/2" {
					return model.Task{}, errors.New("fetch fail")
				}
				return model.Task{Content: "Task 1 content"}, nil
			},
		}
		uc := usecase.New(&mockLogger{}, llmClient, &mockCalendarClient{}, mRepo, vRepo, dateMath, "Asia/Ho_Chi_Minh", "http://memos")
		out, err := uc.AnswerQuery(context.Background(), model.Scope{}, task.QueryInput{Query: "test"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.SourceCount != 2 { // memo/2 excluded from content, but source ID list remains
			t.Errorf("expected length 2, got %d", out.SourceCount)
		}
		if !strings.Contains(out.Answer, "This is the generated answer") {
			t.Errorf("unexpected answer output: %s", out.Answer)
		}
	})

	t.Run("LLM Failure on Generate", func(t *testing.T) {
		vRepo := &ragVectorRepo{
			searchFunc: func(opt repository.SearchTasksOptions) ([]repository.SearchResult, error) {
				return []repository.SearchResult{{MemoID: "memo/1"}}, nil
			},
		}
		mRepo := &ragMemosRepo{
			getTaskFunc: func(id string) (model.Task, error) {
				return model.Task{Content: "Task 1 content"}, nil
			},
		}
		uc := usecase.New(&mockLogger{}, llmClient, &mockCalendarClient{}, mRepo, vRepo, dateMath, "Asia/Ho_Chi_Minh", "http://memos")
		_, err := uc.AnswerQuery(context.Background(), model.Scope{}, task.QueryInput{Query: "error_llm_500"})
		if err == nil {
			t.Errorf("expected llm failure error")
		}
	})

	t.Run("Empty LLM Return Block", func(t *testing.T) {
		vRepo := &ragVectorRepo{
			searchFunc: func(opt repository.SearchTasksOptions) ([]repository.SearchResult, error) {
				return []repository.SearchResult{{MemoID: "memo/1"}}, nil
			},
		}
		mRepo := &ragMemosRepo{
			getTaskFunc: func(id string) (model.Task, error) {
				return model.Task{Content: "Task 1 content"}, nil
			},
		}
		uc := usecase.New(&mockLogger{}, llmEmpty, &mockCalendarClient{}, mRepo, vRepo, dateMath, "Asia/Ho_Chi_Minh", "http://memos")
		_, err := uc.AnswerQuery(context.Background(), model.Scope{}, task.QueryInput{Query: "test"})
		if err == nil {
			t.Errorf("expected empty llm response err")
		}
	})
}
