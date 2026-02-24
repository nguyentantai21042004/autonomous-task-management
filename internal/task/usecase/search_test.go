package usecase_test

import (
	"context"
	"errors"
	"testing"

	"autonomous-task-management/internal/model"
	"autonomous-task-management/internal/task"
	"autonomous-task-management/internal/task/repository"
	"autonomous-task-management/internal/task/usecase"
	"autonomous-task-management/pkg/datemath"
	"autonomous-task-management/pkg/gemini"
)

func TestSearch(t *testing.T) {
	// Initialize minimal dependencies
	llmClient := gemini.NewClient("test-key")
	calClient := &mockCalendarClient{}
	dateMath, _ := datemath.NewParser("Asia/Ho_Chi_Minh")

	t.Run("Empty Query Error", func(t *testing.T) {
		uc := usecase.New(&mockLogger{}, llmClient, calClient, &ragMemosRepo{}, &ragVectorRepo{}, dateMath, "Asia/Ho_Chi_Minh", "http://memos")
		_, err := uc.Search(context.Background(), model.Scope{}, task.SearchInput{Query: ""})
		if !errors.Is(err, task.ErrEmptyQuery) {
			t.Errorf("expected ErrEmptyQuery, got %v", err)
		}
	})

	t.Run("Vector Repository Missing Error", func(t *testing.T) {
		uc := usecase.New(&mockLogger{}, llmClient, calClient, &ragMemosRepo{}, nil, dateMath, "Asia/Ho_Chi_Minh", "http://memos")
		_, err := uc.Search(context.Background(), model.Scope{}, task.SearchInput{Query: "test"})
		if err == nil {
			t.Errorf("expected error when vector repo is nil")
		}
	})

	t.Run("Vector Search Error", func(t *testing.T) {
		vRepo := &ragVectorRepo{
			searchFunc: func(opt repository.SearchTasksOptions) ([]repository.SearchResult, error) {
				return nil, errors.New("vector search failed")
			},
		}
		uc := usecase.New(&mockLogger{}, llmClient, calClient, &ragMemosRepo{}, vRepo, dateMath, "Asia/Ho_Chi_Minh", "http://memos")
		_, err := uc.Search(context.Background(), model.Scope{}, task.SearchInput{Query: "test"})
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
		uc := usecase.New(&mockLogger{}, llmClient, calClient, &ragMemosRepo{}, vRepo, dateMath, "Asia/Ho_Chi_Minh", "http://memos")
		out, err := uc.Search(context.Background(), model.Scope{}, task.SearchInput{Query: "test"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.Count != 0 {
			t.Errorf("expected 0 count array on empty search, got %d", out.Count)
		}
	})

	t.Run("Successful Search Flow", func(t *testing.T) {
		vRepo := &ragVectorRepo{
			searchFunc: func(opt repository.SearchTasksOptions) ([]repository.SearchResult, error) {
				return []repository.SearchResult{
					{MemoID: "memo/1", Score: 0.95},
					{MemoID: "memo/2", Score: 0.88}, // fail fetch
				}, nil
			},
		}
		mRepo := &ragMemosRepo{
			getTaskFunc: func(id string) (model.Task, error) {
				if id == "memo/2" {
					return model.Task{}, errors.New("memo fetch fail")
				}
				return model.Task{ID: "memo/1", Content: "# Task 1"}, nil
			},
		}
		uc := usecase.New(&mockLogger{}, llmClient, calClient, mRepo, vRepo, dateMath, "Asia/Ho_Chi_Minh", "http://memos")
		out, err := uc.Search(context.Background(), model.Scope{}, task.SearchInput{Query: "test", Limit: 5})
		if err != nil {
			t.Fatalf("unexpected search engine error: %v", err)
		}
		if out.Count != 1 {
			t.Errorf("expected 1 result out of 2 due to db fetch fail, got %d", out.Count)
		}
		if out.Results[0].MemoID != "memo/1" {
			t.Errorf("unexpected task result mapping")
		}
	})
}
