package usecase

import (
	"context"
	"errors"
	"testing"

	"autonomous-task-management/internal/agent"
	"autonomous-task-management/internal/automation"
	"autonomous-task-management/internal/checklist"
	"autonomous-task-management/internal/model"
	"autonomous-task-management/internal/task/repository"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// Mock implementations
// ---------------------------------------------------------------------------

type mockLogger struct{}

func (m *mockLogger) Debug(_ context.Context, _ ...any)             {}
func (m *mockLogger) Debugf(_ context.Context, _ string, _ ...any)  {}
func (m *mockLogger) Info(_ context.Context, _ ...any)              {}
func (m *mockLogger) Infof(_ context.Context, _ string, _ ...any)   {}
func (m *mockLogger) Warn(_ context.Context, _ ...any)              {}
func (m *mockLogger) Warnf(_ context.Context, _ string, _ ...any)   {}
func (m *mockLogger) Error(_ context.Context, _ ...any)             {}
func (m *mockLogger) Errorf(_ context.Context, _ string, _ ...any)  {}
func (m *mockLogger) DPanic(_ context.Context, _ ...any)            {}
func (m *mockLogger) DPanicf(_ context.Context, _ string, _ ...any) {}
func (m *mockLogger) Panic(_ context.Context, _ ...any)             {}
func (m *mockLogger) Panicf(_ context.Context, _ string, _ ...any)  {}
func (m *mockLogger) Fatal(_ context.Context, _ ...any)             {}
func (m *mockLogger) Fatalf(_ context.Context, _ string, _ ...any)  {}

type mockMemosRepo struct {
	tasks     map[string]model.Task
	updates   map[string]string
	getErr    error
	updateErr error
}

func newMockMemosRepo() *mockMemosRepo {
	return &mockMemosRepo{
		tasks:   make(map[string]model.Task),
		updates: make(map[string]string),
	}
}

func (m *mockMemosRepo) CreateTask(_ context.Context, opt repository.CreateTaskOptions) (model.Task, error) {
	return model.Task{ID: "new-task", Content: opt.Content}, nil
}

func (m *mockMemosRepo) CreateTasksBatch(_ context.Context, opts []repository.CreateTaskOptions) ([]model.Task, error) {
	tasks := make([]model.Task, len(opts))
	for i, opt := range opts {
		tasks[i] = model.Task{ID: "batch-" + string(rune('0'+i)), Content: opt.Content}
	}
	return tasks, nil
}

func (m *mockMemosRepo) GetTask(_ context.Context, id string) (model.Task, error) {
	if m.getErr != nil {
		return model.Task{}, m.getErr
	}
	task, ok := m.tasks[id]
	if !ok {
		return model.Task{}, errors.New("task not found: " + id)
	}
	return task, nil
}

func (m *mockMemosRepo) ListTasks(_ context.Context, _ repository.ListTasksOptions) ([]model.Task, error) {
	return nil, nil
}

func (m *mockMemosRepo) UpdateTask(_ context.Context, id string, content string) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.updates[id] = content
	return nil
}

type mockVectorRepo struct {
	searchResults []repository.SearchResult
	filterResults []repository.SearchResult
	searchErr     error
	filterErr     error
}

func (m *mockVectorRepo) EmbedTask(_ context.Context, _ model.Task) error { return nil }

func (m *mockVectorRepo) SearchTasks(_ context.Context, _ repository.SearchTasksOptions) ([]repository.SearchResult, error) {
	return m.searchResults, m.searchErr
}

func (m *mockVectorRepo) SearchTasksWithFilter(_ context.Context, _ repository.SearchTasksOptions) ([]repository.SearchResult, error) {
	return m.filterResults, m.filterErr
}

func (m *mockVectorRepo) DeleteTask(_ context.Context, _ string) error { return nil }

type mockChecklistSvc struct {
	stats     checklist.ChecklistStats
	updateAll string
}

func (m *mockChecklistSvc) ParseCheckboxes(_ string) []checklist.Checkbox { return nil }
func (m *mockChecklistSvc) GetStats(_ string) checklist.ChecklistStats    { return m.stats }
func (m *mockChecklistSvc) UpdateCheckbox(_ context.Context, _ checklist.UpdateCheckboxInput) (checklist.UpdateCheckboxOutput, error) {
	return checklist.UpdateCheckboxOutput{}, nil
}
func (m *mockChecklistSvc) UpdateAllCheckboxes(content string, checked bool) string {
	if m.updateAll != "" {
		return m.updateAll
	}
	if checked {
		return content + " [UPDATED]"
	}
	return content
}
func (m *mockChecklistSvc) IsFullyCompleted(_ string) bool           { return false }
func (m *mockChecklistSvc) RegisterAgentTools(_ *agent.ToolRegistry) {}

func newTestAutomationUC(memos *mockMemosRepo, vector *mockVectorRepo, cl *mockChecklistSvc) *implUseCase {
	uc := New(memos, vector, cl, &mockLogger{})
	return uc.(*implUseCase)
}

// ---------------------------------------------------------------------------
// ProcessWebhook tests
// ---------------------------------------------------------------------------

func TestProcessWebhook_SkipsPR_NonMergedAction(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		action    string
	}{
		{"PR opened", "pull_request", "opened"},
		{"PR closed (not merged)", "pull_request", "closed"},
		{"PR synchronize", "pull_request", "synchronize"},
		{"MR opened", "merge_request", "opened"},
		{"MR closed (not merged)", "merge_request", "closed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := newTestAutomationUC(newMockMemosRepo(), &mockVectorRepo{}, &mockChecklistSvc{})

			output, err := uc.ProcessWebhook(context.Background(), model.Scope{UserID: "test"}, automation.ProcessWebhookInput{
				Event: model.WebhookEvent{
					EventType: tt.eventType,
					Action:    tt.action,
				},
			})

			assert.NoError(t, err)
			assert.Equal(t, 0, output.TasksUpdated)
		})
	}
}

func TestProcessWebhook_ProcessesMergedPR(t *testing.T) {
	memos := newMockMemosRepo()
	memos.tasks["task-1"] = model.Task{
		ID:      "task-1",
		Content: "Review PR #42\n- [ ] check code\n- [ ] approve",
		Tags:    []string{"#pr/42"},
	}

	vector := &mockVectorRepo{
		filterResults: []repository.SearchResult{
			{MemoID: "task-1", Score: 1.0},
		},
	}

	cl := &mockChecklistSvc{
		stats: checklist.ChecklistStats{Total: 2, Completed: 0, Pending: 2},
	}

	uc := newTestAutomationUC(memos, vector, cl)

	output, err := uc.ProcessWebhook(context.Background(), model.Scope{UserID: "test"}, automation.ProcessWebhookInput{
		Event: model.WebhookEvent{
			EventType:  "pull_request",
			Action:     "merged",
			Repository: "org/my-repo",
			PRNumber:   42,
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, output.TasksUpdated)
	assert.Contains(t, output.TaskIDs, "task-1")
	assert.NotEmpty(t, memos.updates["task-1"])
}

func TestProcessWebhook_ProcessesPushEvent(t *testing.T) {
	memos := newMockMemosRepo()
	memos.tasks["task-2"] = model.Task{
		ID:      "task-2",
		Content: "Deploy main branch\n- [ ] deploy",
		Tags:    []string{"#repo/my-repo"},
	}

	vector := &mockVectorRepo{
		filterResults: []repository.SearchResult{
			{MemoID: "task-2", Score: 1.0},
		},
	}

	cl := &mockChecklistSvc{
		stats: checklist.ChecklistStats{Total: 1, Completed: 0, Pending: 1},
	}

	uc := newTestAutomationUC(memos, vector, cl)

	output, err := uc.ProcessWebhook(context.Background(), model.Scope{UserID: "test"}, automation.ProcessWebhookInput{
		Event: model.WebhookEvent{
			EventType:  "push",
			Repository: "org/my-repo",
			Branch:     "main",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, output.TasksUpdated)
}

func TestProcessWebhook_NoMatchingTasks(t *testing.T) {
	uc := newTestAutomationUC(newMockMemosRepo(), &mockVectorRepo{}, &mockChecklistSvc{})

	output, err := uc.ProcessWebhook(context.Background(), model.Scope{UserID: "test"}, automation.ProcessWebhookInput{
		Event: model.WebhookEvent{
			EventType: "push",
			Branch:    "feature/xyz",
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, 0, output.TasksUpdated)
	assert.Contains(t, output.Message, "No matching tasks")
}

func TestProcessWebhook_SkipsTaskWithNoCheckboxes(t *testing.T) {
	memos := newMockMemosRepo()
	memos.tasks["task-3"] = model.Task{
		ID:      "task-3",
		Content: "Task with no checkboxes - just text",
	}

	vector := &mockVectorRepo{
		filterResults: []repository.SearchResult{
			{MemoID: "task-3", Score: 1.0},
		},
	}

	cl := &mockChecklistSvc{
		stats: checklist.ChecklistStats{Total: 0},
	}

	uc := newTestAutomationUC(memos, vector, cl)

	_, err := uc.ProcessWebhook(context.Background(), model.Scope{UserID: "test"}, automation.ProcessWebhookInput{
		Event: model.WebhookEvent{
			EventType:  "pull_request",
			Action:     "merged",
			Repository: "org/my-repo",
			PRNumber:   99,
		},
	})

	assert.NoError(t, err)
	assert.Empty(t, memos.updates)
}

func TestProcessWebhook_TaskAlreadyCompleted(t *testing.T) {
	content := "Review PR\n- [x] done"
	memos := newMockMemosRepo()
	memos.tasks["task-4"] = model.Task{ID: "task-4", Content: content}

	vector := &mockVectorRepo{
		filterResults: []repository.SearchResult{
			{MemoID: "task-4", Score: 1.0},
		},
	}

	cl := &mockChecklistSvc{
		stats:     checklist.ChecklistStats{Total: 1, Completed: 1},
		updateAll: content, // Same content → no change
	}

	uc := newTestAutomationUC(memos, vector, cl)

	output, err := uc.ProcessWebhook(context.Background(), model.Scope{UserID: "test"}, automation.ProcessWebhookInput{
		Event: model.WebhookEvent{
			EventType:  "pull_request",
			Action:     "merged",
			PRNumber:   1,
			Repository: "org/repo",
		},
	})

	assert.NoError(t, err)
	assert.Empty(t, memos.updates)
	// updateTaskChecklist returns nil when nothing changed,
	// but ProcessWebhook still counts it as "updated" (no error = success).
	// This is a known behavior: ProcessWebhook counts tasks it tried to update.
	assert.Equal(t, 1, output.TasksUpdated)
}

func TestProcessWebhook_UpdateTaskFails_ContinuesOthers(t *testing.T) {
	memos := newMockMemosRepo()
	memos.tasks["task-A"] = model.Task{ID: "task-A", Content: "- [ ] item"}
	memos.tasks["task-B"] = model.Task{ID: "task-B", Content: "- [ ] item b"}
	memos.updateErr = errors.New("memos API error")

	vector := &mockVectorRepo{
		filterResults: []repository.SearchResult{
			{MemoID: "task-A", Score: 1.0},
			{MemoID: "task-B", Score: 0.9},
		},
	}

	cl := &mockChecklistSvc{
		stats: checklist.ChecklistStats{Total: 1, Completed: 0, Pending: 1},
	}

	uc := newTestAutomationUC(memos, vector, cl)

	output, err := uc.ProcessWebhook(context.Background(), model.Scope{UserID: "test"}, automation.ProcessWebhookInput{
		Event: model.WebhookEvent{
			EventType:  "pull_request",
			Action:     "merged",
			Repository: "org/repo",
			PRNumber:   1,
		},
	})

	assert.NoError(t, err)
	assert.Equal(t, 0, output.TasksUpdated)
}

// ---------------------------------------------------------------------------
// CompleteTask tests
// ---------------------------------------------------------------------------

func TestCompleteTask_Success(t *testing.T) {
	memos := newMockMemosRepo()
	memos.tasks["task-1"] = model.Task{
		ID:      "task-1",
		Content: "task\n- [ ] item 1\n- [ ] item 2",
	}

	cl := &mockChecklistSvc{
		updateAll: "task\n- [x] item 1\n- [x] item 2",
	}

	uc := newTestAutomationUC(memos, &mockVectorRepo{}, cl)

	err := uc.CompleteTask(context.Background(), model.Scope{UserID: "test"}, "task-1")

	assert.NoError(t, err)
	assert.Contains(t, memos.updates["task-1"], "[x]")
}

func TestCompleteTask_TaskNotFound(t *testing.T) {
	memos := newMockMemosRepo()
	uc := newTestAutomationUC(memos, &mockVectorRepo{}, &mockChecklistSvc{})

	err := uc.CompleteTask(context.Background(), model.Scope{UserID: "test"}, "nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch task")
}

func TestCompleteTask_AlreadyCompleted(t *testing.T) {
	content := "task\n- [x] done"
	memos := newMockMemosRepo()
	memos.tasks["task-1"] = model.Task{ID: "task-1", Content: content}

	cl := &mockChecklistSvc{updateAll: content}
	uc := newTestAutomationUC(memos, &mockVectorRepo{}, cl)

	err := uc.CompleteTask(context.Background(), model.Scope{UserID: "test"}, "task-1")

	assert.NoError(t, err)
	assert.Empty(t, memos.updates)
}

func TestCompleteTask_MemosUpdateFails(t *testing.T) {
	memos := newMockMemosRepo()
	memos.tasks["task-1"] = model.Task{ID: "task-1", Content: "- [ ] item"}
	memos.updateErr = errors.New("memos down")

	cl := &mockChecklistSvc{}
	uc := newTestAutomationUC(memos, &mockVectorRepo{}, cl)

	err := uc.CompleteTask(context.Background(), model.Scope{UserID: "test"}, "task-1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update task")
}

// ---------------------------------------------------------------------------
// buildMatchCriteria tests
// ---------------------------------------------------------------------------

func TestBuildMatchCriteria_PR(t *testing.T) {
	matcher := &taskMatcher{l: &mockLogger{}}

	criteria := matcher.buildMatchCriteria(model.WebhookEvent{
		Repository: "org/my-repo",
		PRNumber:   42,
		Branch:     "feature/auth",
	})

	assert.Contains(t, criteria.Tags, "#repo/my-repo")
	assert.Contains(t, criteria.Tags, "#pr/42")
	assert.Contains(t, criteria.Keywords, "PR #42")
	assert.Contains(t, criteria.Keywords, "#42")
	assert.Contains(t, criteria.Keywords, "feature/auth")
}

func TestBuildMatchCriteria_Issue(t *testing.T) {
	matcher := &taskMatcher{l: &mockLogger{}}

	criteria := matcher.buildMatchCriteria(model.WebhookEvent{
		Repository:  "org/my-repo",
		IssueNumber: 15,
	})

	assert.Contains(t, criteria.Tags, "#issue/15")
	assert.Contains(t, criteria.Keywords, "Issue #15")
}

func TestBuildMatchCriteria_EmptyEvent(t *testing.T) {
	matcher := &taskMatcher{l: &mockLogger{}}

	criteria := matcher.buildMatchCriteria(model.WebhookEvent{})

	assert.Empty(t, criteria.Tags)
	assert.Empty(t, criteria.Keywords)
}

func TestBuildMatchCriteria_RepoWithoutOrg(t *testing.T) {
	matcher := &taskMatcher{l: &mockLogger{}}

	criteria := matcher.buildMatchCriteria(model.WebhookEvent{
		Repository: "my-repo",
	})

	for _, tag := range criteria.Tags {
		assert.NotEqual(t, "#repo/my-repo", tag)
	}
}

// ---------------------------------------------------------------------------
// mergeMatches tests
// ---------------------------------------------------------------------------

func TestMergeMatches_Deduplication(t *testing.T) {
	matcher := &taskMatcher{l: &mockLogger{}}

	tagMatches := []TaskMatch{
		{TaskID: "task-1", MatchScore: 1.0},
		{TaskID: "task-2", MatchScore: 1.0},
	}
	keywordMatches := []TaskMatch{
		{TaskID: "task-2", MatchScore: 0.8},
		{TaskID: "task-3", MatchScore: 0.7},
	}

	merged := matcher.mergeMatches(tagMatches, keywordMatches)

	assert.Len(t, merged, 3)
	ids := make(map[string]bool)
	for _, m := range merged {
		ids[m.TaskID] = true
	}
	assert.True(t, ids["task-1"])
	assert.True(t, ids["task-2"])
	assert.True(t, ids["task-3"])
}

func TestMergeMatches_TagsPrioritized(t *testing.T) {
	matcher := &taskMatcher{l: &mockLogger{}}

	tagMatches := []TaskMatch{
		{TaskID: "task-1", MatchScore: 1.0, MatchReason: "exact-tag"},
	}
	keywordMatches := []TaskMatch{
		{TaskID: "task-1", MatchScore: 0.5, MatchReason: "semantic"},
	}

	merged := matcher.mergeMatches(tagMatches, keywordMatches)

	assert.Len(t, merged, 1)
	assert.Equal(t, "exact-tag", merged[0].MatchReason)
}

func TestMergeMatches_BothEmpty(t *testing.T) {
	matcher := &taskMatcher{l: &mockLogger{}}

	merged := matcher.mergeMatches(nil, nil)

	assert.Empty(t, merged)
}
