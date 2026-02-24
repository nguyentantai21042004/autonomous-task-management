package tools_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"autonomous-task-management/internal/agent/tools"
	"autonomous-task-management/internal/checklist"
	"autonomous-task-management/internal/model"
	"autonomous-task-management/internal/task"
	"autonomous-task-management/internal/task/repository"
	"autonomous-task-management/pkg/gcalendar"
)

// mockLogger
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

// mockTaskUseCase
type mockTaskUseCase struct {
	searchOutput task.SearchOutput
	searchErr    error
}

func (m *mockTaskUseCase) Search(ctx context.Context, sc model.Scope, input task.SearchInput) (task.SearchOutput, error) {
	return m.searchOutput, m.searchErr
}
func (m *mockTaskUseCase) CreateBulk(ctx context.Context, sc model.Scope, input task.CreateBulkInput) (task.CreateBulkOutput, error) {
	return task.CreateBulkOutput{}, nil
}
func (m *mockTaskUseCase) AnswerQuery(ctx context.Context, sc model.Scope, input task.QueryInput) (task.QueryOutput, error) {
	return task.QueryOutput{}, nil
}

// mockCalendarClient
type mockCalendarClient struct {
	events []gcalendar.Event
	err    error
}

func (m *mockCalendarClient) ListEvents(ctx context.Context, req gcalendar.ListEventsRequest) ([]gcalendar.Event, error) {
	return m.events, m.err
}

// mockMemosRepo
type mockMemosRepo struct {
	task model.Task
	err  error
}

func (m *mockMemosRepo) CreateTask(ctx context.Context, opts repository.CreateTaskOptions) (model.Task, error) {
	return model.Task{}, nil
}
func (m *mockMemosRepo) CreateTasksBatch(ctx context.Context, opts []repository.CreateTaskOptions) ([]model.Task, error) {
	return nil, nil
}
func (m *mockMemosRepo) GetTask(ctx context.Context, id string) (model.Task, error) {
	if id == "" {
		return model.Task{}, errors.New("missing id")
	}
	return m.task, m.err
}
func (m *mockMemosRepo) UpdateTask(ctx context.Context, id, content string) error { return m.err }
func (m *mockMemosRepo) DeleteTask(ctx context.Context, id string) error          { return m.err }
func (m *mockMemosRepo) ListTasks(ctx context.Context, opts repository.ListTasksOptions) ([]model.Task, error) {
	return nil, m.err
}

// mockChecklistService
type mockChecklistService struct {
	stats     checklist.ChecklistStats
	updateOut checklist.UpdateCheckboxOutput
	updateErr error
}

func (m *mockChecklistService) ParseCheckboxes(content string) []checklist.Checkbox { return nil }
func (m *mockChecklistService) GetStats(content string) checklist.ChecklistStats    { return m.stats }
func (m *mockChecklistService) UpdateCheckbox(ctx context.Context, input checklist.UpdateCheckboxInput) (checklist.UpdateCheckboxOutput, error) {
	return m.updateOut, m.updateErr
}
func (m *mockChecklistService) UpdateAllCheckboxes(content string, checked bool) string { return "" }
func (m *mockChecklistService) IsFullyCompleted(content string) bool                    { return false }

func TestAgentTools(t *testing.T) {
	ctx := context.Background()
	l := &mockLogger{}

	t.Run("CheckCalendarTool", func(t *testing.T) {
		client := &mockCalendarClient{
			events: []gcalendar.Event{{Summary: "Meeting", StartTime: time.Now()}},
		}
		tool := tools.NewCheckCalendarTool(client, l)

		if tool.Name() != "check_calendar" {
			t.Errorf("unexpected name: %s", tool.Name())
		}
		if tool.Description() == "" || len(tool.Parameters()) == 0 {
			t.Errorf("missing desc or params")
		}

		res, err := tool.Execute(ctx, map[string]interface{}{"start_date": "2026-02-24", "end_date": "2026-02-25"})
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		out, ok := res.(tools.CheckCalendarOutput)
		if !ok || len(out.Events) != 1 {
			t.Errorf("unexpected result: %v", res)
		}

		// failure
		client.err = errors.New("cal error")
		_, err = tool.Execute(ctx, map[string]interface{}{})
		if err == nil {
			t.Errorf("expected error")
		}
	})

	t.Run("GetChecklistProgressTool", func(t *testing.T) {
		repo := &mockMemosRepo{
			task: model.Task{Content: "- [x] task"},
		}
		svc := &mockChecklistService{
			stats: checklist.ChecklistStats{Total: 1, Completed: 1, Pending: 0, Progress: 100},
		}
		tool := tools.NewGetChecklistProgressTool(repo, svc, l)

		if tool.Name() != "get_checklist_progress" {
			t.Errorf("unexpected name: %s", tool.Name())
		}
		if tool.Description() == "" || len(tool.Parameters()) == 0 {
			t.Errorf("missing desc or params")
		}

		res, err := tool.Execute(ctx, map[string]interface{}{"task_id": "123"})
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}

		stats, ok := res.(tools.GetChecklistProgressOutput)
		if !ok || stats.Stats.Progress != 100 {
			t.Errorf("unexpected result: %v", res)
		}

		// Invalid arg
		_, err = tool.Execute(ctx, map[string]interface{}{})
		if err == nil {
			t.Errorf("expected error missing task_id")
		}
	})

	t.Run("SearchTasksTool", func(t *testing.T) {
		uc := &mockTaskUseCase{
			searchOutput: task.SearchOutput{
				Count:   1,
				Results: []task.SearchResultItem{{MemoID: "1", Content: "Find me"}},
			},
		}
		tool := tools.NewSearchTasksTool(uc)

		if tool.Name() != "search_tasks" {
			t.Errorf("unexpected name: %s", tool.Name())
		}
		if tool.Description() == "" || len(tool.Parameters()) == 0 {
			t.Errorf("missing desc or params")
		}

		res, err := tool.Execute(ctx, map[string]interface{}{"query": "find", "limit": float64(2)})
		if err != nil {
			t.Fatalf("unexpected test error: %v", err)
		}

		resMap, ok := res.(map[string]interface{})
		if !ok || resMap["count"] != 1 {
			t.Errorf("unexpected err result: %v", res)
		}
	})

	t.Run("UpdateChecklistItemTool", func(t *testing.T) {
		repo := &mockMemosRepo{
			task: model.Task{ID: "1", Content: "- [ ] step 1"},
		}
		svc := &mockChecklistService{
			updateOut: checklist.UpdateCheckboxOutput{Updated: true},
		}
		tool := tools.NewUpdateChecklistItemTool(repo, nil, svc, l)

		if tool.Name() != "update_checklist_item" {
			t.Errorf("unexpected name: %s", tool.Name())
		}
		if tool.Description() == "" || len(tool.Parameters()) == 0 {
			t.Errorf("missing desc or params")
		}

		res, err := tool.Execute(ctx, map[string]interface{}{
			"task_id":   "123",
			"item_text": "step 1",
			"checked":   true,
		})
		if err != nil {
			t.Fatalf("unexpected var: %v", err)
		}

		out, ok := res.(tools.UpdateChecklistItemOutput)
		if !ok || !out.Updated {
			t.Errorf("expected update to succeed")
		}
	})
}
