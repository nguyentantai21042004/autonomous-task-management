package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"autonomous-task-management/internal/model"
	"autonomous-task-management/internal/task"
	"autonomous-task-management/internal/task/repository"
	"autonomous-task-management/pkg/gemini"
	"autonomous-task-management/pkg/llmprovider"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock Repository implementations

type mockMemosRepo struct {
	mock.Mock
}

func (m *mockMemosRepo) CreateTask(ctx context.Context, opt repository.CreateTaskOptions) (model.Task, error) {
	args := m.Called(ctx, opt)
	return args.Get(0).(model.Task), args.Error(1)
}

func (m *mockMemosRepo) CreateTasksBatch(ctx context.Context, opts []repository.CreateTaskOptions) ([]model.Task, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).([]model.Task), args.Error(1)
}

func (m *mockMemosRepo) GetTask(ctx context.Context, id string) (model.Task, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(model.Task), args.Error(1)
}

func (m *mockMemosRepo) ListTasks(ctx context.Context, opt repository.ListTasksOptions) ([]model.Task, error) {
	args := m.Called(ctx, opt)
	return args.Get(0).([]model.Task), args.Error(1)
}

func (m *mockMemosRepo) UpdateTask(ctx context.Context, id string, content string) error {
	args := m.Called(ctx, id, content)
	return args.Error(0)
}

type mockVectorRepo struct {
	mock.Mock
}

func (m *mockVectorRepo) EmbedTask(ctx context.Context, t model.Task) error {
	args := m.Called(ctx, t)
	return args.Error(0)
}

func (m *mockVectorRepo) SearchTasks(ctx context.Context, opt repository.SearchTasksOptions) ([]repository.SearchResult, error) {
	args := m.Called(ctx, opt)
	return args.Get(0).([]repository.SearchResult), args.Error(1)
}

func (m *mockVectorRepo) SearchTasksWithFilter(ctx context.Context, opt repository.SearchTasksOptions) ([]repository.SearchResult, error) {
	args := m.Called(ctx, opt)
	return args.Get(0).([]repository.SearchResult), args.Error(1)
}

func (m *mockVectorRepo) DeleteTask(ctx context.Context, taskID string) error {
	args := m.Called(ctx, taskID)
	return args.Error(0)
}

type mockDateMath struct{}

func (m *mockDateMath) Parse(expr string, ref time.Time) (time.Time, error) {
	return ref, nil
}

func (m *mockDateMath) EndOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, t.Location())
}

// Helper: create implUseCase with mocks

func newTestTaskUC(llmManager llmprovider.IManager, repo *mockMemosRepo, vectorRepo *mockVectorRepo) *implUseCase {
	return &implUseCase{
		l:          &mockLogger{},
		llm:        llmManager,
		calendar:   nil,
		repo:       repo,
		vectorRepo: vectorRepo,
		dateMath:   &mockDateMath{},
		reranker:   nil,
		timezone:   "Asia/Ho_Chi_Minh",
		memosURL:   "http://localhost:5230",
	}
}

func makeLLMManager(responseText string) llmprovider.IManager {
	client := &mockGeminiClient{
		response: &gemini.Response{
			Content: gemini.Content{
				Parts: []gemini.Part{{Text: responseText}},
			},
			Usage: &gemini.Usage{},
		},
	}
	return createManagerFromGeminiClient(client, &mockLogger{})
}

func makeLLMManagerErr(err error) llmprovider.IManager {
	client := &mockGeminiClient{err: err}
	return createManagerFromGeminiClient(client, &mockLogger{})
}

// Tests: sanitizeJSONResponse

func TestSanitizeJSONResponse_PlainJSON(t *testing.T) {
	input := `[{"title":"Test"}]`
	assert.Equal(t, input, sanitizeJSONResponse(input))
}

func TestSanitizeJSONResponse_CodeFence(t *testing.T) {
	input := "```json\n[{\"title\":\"Test\"}]\n```"
	assert.Equal(t, `[{"title":"Test"}]`, sanitizeJSONResponse(input))
}

func TestSanitizeJSONResponse_CodeFenceNoLang(t *testing.T) {
	input := "```\n{\"key\":\"val\"}\n```"
	assert.Equal(t, `{"key":"val"}`, sanitizeJSONResponse(input))
}

func TestSanitizeJSONResponse_ProseWrapped(t *testing.T) {
	input := "Here are the parsed tasks:\n[{\"title\":\"Test\"}]\nHope this helps!"
	assert.Equal(t, `[{"title":"Test"}]`, sanitizeJSONResponse(input))
}

func TestSanitizeJSONResponse_NoBrackets(t *testing.T) {
	input := "no json here"
	assert.Equal(t, input, sanitizeJSONResponse(input))
}

// Tests: buildMarkdownContent

func TestBuildMarkdownContent_Basic(t *testing.T) {
	tw := taskWithDate{
		Title:                    "Write report",
		Description:              "Quarterly numbers",
		DueDateAbsolute:          time.Date(2025, 3, 15, 23, 59, 59, 0, time.UTC),
		Priority:                 "p1",
		EstimatedDurationMinutes: 120,
	}

	content := buildMarkdownContent(tw)
	assert.Contains(t, content, "## Write report")
	assert.Contains(t, content, "Quarterly numbers")
	assert.Contains(t, content, "2025-03-15")
	assert.Contains(t, content, "#priority/p1")
	assert.Contains(t, content, "120 min")
}

func TestBuildMarkdownContent_NoDescription(t *testing.T) {
	tw := taskWithDate{
		Title:           "Quick task",
		DueDateAbsolute: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Priority:        "p2",
	}

	content := buildMarkdownContent(tw)
	assert.Contains(t, content, "## Quick task")
}

// Tests: priorityTag and allTags

func TestPriorityTag(t *testing.T) {
	assert.Equal(t, "#priority/p0", priorityTag("p0"))
	assert.Equal(t, "#priority/p3", priorityTag("p3"))
}

func TestAllTags(t *testing.T) {
	tw := taskWithDate{
		Priority: "p1",
		Tags:     []string{"#project/smap", "#type/review"},
	}
	tags := allTags(tw)
	assert.Equal(t, []string{"#priority/p1", "#project/smap", "#type/review"}, tags)
}

func TestAllTags_NoUserTags(t *testing.T) {
	tw := taskWithDate{Priority: "p2"}
	tags := allTags(tw)
	assert.Equal(t, []string{"#priority/p2"}, tags)
}

// Tests: truncateText

func TestTruncateText_Short(t *testing.T) {
	assert.Equal(t, "hello", truncateText("hello", 100))
}

func TestTruncateText_ExactLimit(t *testing.T) {
	assert.Equal(t, "hello", truncateText("hello", 5))
}

func TestTruncateText_Truncated(t *testing.T) {
	result := truncateText("hello world", 5)
	assert.Equal(t, "hello... ["+"\u0111\u00e3 c\u1eaft b\u1edbt]", result)
}

func TestTruncateText_Vietnamese(t *testing.T) {
	input := "Xin ch\u00e0o th\u1ebf gi\u1edbi"
	result := truncateText(input, 8)
	assert.Equal(t, "Xin ch\u00e0o... ["+"\u0111\u00e3 c\u1eaft b\u1edbt]", result)
}

// Tests: resolveDueDates

func TestResolveDueDates_ValidISO(t *testing.T) {
	uc := newTestTaskUC(nil, nil, nil)
	parsed := []ParsedTask{
		{Title: "T1", DueDateAbsolute: "2025-06-15T10:00:00+07:00", Priority: "p1"},
	}

	result := uc.resolveDueDates(parsed)
	assert.Len(t, result, 1)
	assert.Equal(t, "T1", result[0].Title)
	assert.Equal(t, 2025, result[0].DueDateAbsolute.Year())
	assert.Equal(t, time.Month(6), result[0].DueDateAbsolute.Month())
	assert.Equal(t, 15, result[0].DueDateAbsolute.Day())
}

func TestResolveDueDates_InvalidFallsBackToEndOfToday(t *testing.T) {
	uc := newTestTaskUC(nil, nil, nil)
	parsed := []ParsedTask{
		{Title: "T1", DueDateAbsolute: "not-a-date", Priority: "p2"},
	}

	result := uc.resolveDueDates(parsed)
	assert.Len(t, result, 1)
	assert.Equal(t, 23, result[0].DueDateAbsolute.Hour())
	assert.Equal(t, 59, result[0].DueDateAbsolute.Minute())
}

// Tests: CreateBulk

func TestCreateBulk_EmptyInput(t *testing.T) {
	uc := newTestTaskUC(nil, nil, nil)
	_, err := uc.CreateBulk(context.Background(), model.Scope{UserID: "u1"}, task.CreateBulkInput{RawText: ""})
	assert.ErrorIs(t, err, task.ErrEmptyInput)
}

func TestCreateBulk_WhitespaceOnlyInput(t *testing.T) {
	uc := newTestTaskUC(nil, nil, nil)
	_, err := uc.CreateBulk(context.Background(), model.Scope{UserID: "u1"}, task.CreateBulkInput{RawText: "   "})
	assert.ErrorIs(t, err, task.ErrEmptyInput)
}

func TestCreateBulk_LLMError(t *testing.T) {
	mgr := makeLLMManagerErr(errors.New("LLM down"))
	uc := newTestTaskUC(mgr, nil, nil)

	_, err := uc.CreateBulk(context.Background(), model.Scope{UserID: "u1"}, task.CreateBulkInput{RawText: "some task"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LLM")
}

func TestCreateBulk_LLMReturnsEmptyTasks(t *testing.T) {
	mgr := makeLLMManager("[]")
	uc := newTestTaskUC(mgr, nil, nil)

	_, err := uc.CreateBulk(context.Background(), model.Scope{UserID: "u1"}, task.CreateBulkInput{RawText: "some task"})
	assert.ErrorIs(t, err, task.ErrNoTasksParsed)
}

func TestCreateBulk_Success(t *testing.T) {
	llmResp := `[{"title":"Buy milk","description":"","due_date_absolute":"2025-06-15T23:59:59+07:00","priority":"p2","tags":["#type/shopping"],"estimated_duration_minutes":30}]`
	mgr := makeLLMManager(llmResp)

	repo := new(mockMemosRepo)
	repo.On("CreateTask", mock.Anything, mock.Anything).Return(model.Task{
		ID:      "memo-1",
		MemoURL: "http://localhost:5230/m/memo-1",
		Content: "## Buy milk",
	}, nil)

	vectorRepo := new(mockVectorRepo)
	vectorRepo.On("EmbedTask", mock.Anything, mock.Anything).Return(nil)

	uc := newTestTaskUC(mgr, repo, vectorRepo)
	output, err := uc.CreateBulk(context.Background(), model.Scope{UserID: "u1"}, task.CreateBulkInput{RawText: "Buy milk tomorrow"})

	assert.NoError(t, err)
	assert.Equal(t, 1, output.TaskCount)
	assert.Equal(t, "Buy milk", output.Tasks[0].Title)
	assert.Equal(t, "memo-1", output.Tasks[0].MemoID)
	repo.AssertExpectations(t)
	vectorRepo.AssertExpectations(t)
}

func TestCreateBulk_EmbedFails_StillSucceeds(t *testing.T) {
	llmResp := `[{"title":"Task","description":"","due_date_absolute":"2025-06-15T23:59:59+07:00","priority":"p2","tags":[],"estimated_duration_minutes":30}]`
	mgr := makeLLMManager(llmResp)

	repo := new(mockMemosRepo)
	repo.On("CreateTask", mock.Anything, mock.Anything).Return(model.Task{
		ID: "memo-1", MemoURL: "http://localhost:5230/m/memo-1",
	}, nil)

	vectorRepo := new(mockVectorRepo)
	vectorRepo.On("EmbedTask", mock.Anything, mock.Anything).Return(errors.New("qdrant down"))

	uc := newTestTaskUC(mgr, repo, vectorRepo)
	output, err := uc.CreateBulk(context.Background(), model.Scope{UserID: "u1"}, task.CreateBulkInput{RawText: "a task"})

	assert.NoError(t, err)
	assert.Equal(t, 1, output.TaskCount)
}

// Tests: Search

func TestSearch_EmptyQuery(t *testing.T) {
	uc := newTestTaskUC(nil, nil, nil)
	_, err := uc.Search(context.Background(), model.Scope{UserID: "u1"}, task.SearchInput{Query: ""})
	assert.ErrorIs(t, err, task.ErrEmptyQuery)
}

func TestSearch_NilVectorRepo(t *testing.T) {
	uc := newTestTaskUC(nil, nil, nil)
	uc.vectorRepo = nil
	_, err := uc.Search(context.Background(), model.Scope{UserID: "u1"}, task.SearchInput{Query: "test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unavailable")
}

func TestSearch_DefaultLimit(t *testing.T) {
	repo := new(mockMemosRepo)
	vectorRepo := new(mockVectorRepo)
	vectorRepo.On("SearchTasks", mock.Anything, mock.MatchedBy(func(opt repository.SearchTasksOptions) bool {
		return opt.Limit == 10
	})).Return([]repository.SearchResult{}, nil)

	uc := newTestTaskUC(nil, repo, vectorRepo)
	output, err := uc.Search(context.Background(), model.Scope{UserID: "u1"}, task.SearchInput{Query: "test", Limit: 0})

	assert.NoError(t, err)
	assert.Equal(t, 0, output.Count)
	vectorRepo.AssertExpectations(t)
}

func TestSearch_WithResults(t *testing.T) {
	repo := new(mockMemosRepo)
	vectorRepo := new(mockVectorRepo)

	vectorRepo.On("SearchTasks", mock.Anything, mock.Anything).Return([]repository.SearchResult{
		{MemoID: "memo-1", Score: 0.95},
		{MemoID: "memo-2", Score: 0.85},
	}, nil)

	repo.On("GetTask", mock.Anything, "memo-1").Return(model.Task{
		ID: "memo-1", MemoURL: "http://localhost:5230/m/memo-1", Content: "## Task 1",
	}, nil)
	repo.On("GetTask", mock.Anything, "memo-2").Return(model.Task{
		ID: "memo-2", MemoURL: "http://localhost:5230/m/memo-2", Content: "## Task 2",
	}, nil)

	uc := newTestTaskUC(nil, repo, vectorRepo)
	output, err := uc.Search(context.Background(), model.Scope{UserID: "u1"}, task.SearchInput{Query: "task", Limit: 5})

	assert.NoError(t, err)
	assert.Equal(t, 2, output.Count)
	assert.Equal(t, "## Task 1", output.Results[0].Content)
	assert.Equal(t, 0.95, output.Results[0].Score)
}

func TestSearch_QdrantError(t *testing.T) {
	vectorRepo := new(mockVectorRepo)
	vectorRepo.On("SearchTasks", mock.Anything, mock.Anything).Return([]repository.SearchResult(nil), errors.New("qdrant down"))

	uc := newTestTaskUC(nil, nil, vectorRepo)
	_, err := uc.Search(context.Background(), model.Scope{UserID: "u1"}, task.SearchInput{Query: "test"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to search")
}

func TestSearch_ZombieVectorCleanup(t *testing.T) {
	repo := new(mockMemosRepo)
	vectorRepo := new(mockVectorRepo)

	vectorRepo.On("SearchTasks", mock.Anything, mock.Anything).Return([]repository.SearchResult{
		{MemoID: "deleted-memo", Score: 0.9},
		{MemoID: "valid-memo", Score: 0.8},
	}, nil)

	repo.On("GetTask", mock.Anything, "deleted-memo").Return(model.Task{}, errors.New("404 Not Found"))
	repo.On("GetTask", mock.Anything, "valid-memo").Return(model.Task{
		ID: "valid-memo", MemoURL: "http://localhost:5230/m/valid-memo", Content: "## Valid",
	}, nil)

	vectorRepo.On("DeleteTask", mock.Anything, "deleted-memo").Return(nil)

	uc := newTestTaskUC(nil, repo, vectorRepo)
	output, err := uc.Search(context.Background(), model.Scope{UserID: "u1"}, task.SearchInput{Query: "test"})

	assert.NoError(t, err)
	assert.Equal(t, 1, output.Count)
	assert.Equal(t, "valid-memo", output.Results[0].MemoID)

	// Give the goroutine time to execute
	time.Sleep(50 * time.Millisecond)
	vectorRepo.AssertCalled(t, "DeleteTask", mock.Anything, "deleted-memo")
}

// Tests: AnswerQuery

func TestAnswerQuery_EmptyQuery(t *testing.T) {
	uc := newTestTaskUC(nil, nil, nil)
	_, err := uc.AnswerQuery(context.Background(), model.Scope{UserID: "u1"}, task.QueryInput{Query: ""})
	assert.ErrorIs(t, err, task.ErrEmptyQuery)
}

func TestAnswerQuery_NoResults(t *testing.T) {
	vectorRepo := new(mockVectorRepo)
	vectorRepo.On("SearchTasks", mock.Anything, mock.Anything).Return([]repository.SearchResult{}, nil)

	uc := newTestTaskUC(nil, nil, vectorRepo)
	output, err := uc.AnswerQuery(context.Background(), model.Scope{UserID: "u1"}, task.QueryInput{Query: "what tasks"})

	assert.NoError(t, err)
	assert.Contains(t, output.Answer, "Kh\u00f4ng t\u00ecm th\u1ea5y")
	assert.Equal(t, 0, output.SourceCount)
}

func TestAnswerQuery_Success(t *testing.T) {
	repo := new(mockMemosRepo)
	vectorRepo := new(mockVectorRepo)

	vectorRepo.On("SearchTasks", mock.Anything, mock.Anything).Return([]repository.SearchResult{
		{MemoID: "memo-1", Score: 0.9, Payload: map[string]interface{}{"content": "task1 content"}},
	}, nil)

	repo.On("GetTask", mock.Anything, "memo-1").Return(model.Task{
		ID: "memo-1", MemoURL: "http://localhost:5230/m/memo-1", Content: "## Task 1\nDetails here",
	}, nil)

	mgr := makeLLMManager("Task 1 details")
	uc := newTestTaskUC(mgr, repo, vectorRepo)

	output, err := uc.AnswerQuery(context.Background(), model.Scope{UserID: "u1"}, task.QueryInput{Query: "what tasks do I have?"})

	assert.NoError(t, err)
	assert.Contains(t, output.Answer, "Task 1")
	assert.Equal(t, 1, output.SourceCount)
	assert.Equal(t, "memo-1", output.SourceTasks[0].MemoID)
}

func TestAnswerQuery_VectorSearchError(t *testing.T) {
	vectorRepo := new(mockVectorRepo)
	vectorRepo.On("SearchTasks", mock.Anything, mock.Anything).Return([]repository.SearchResult(nil), errors.New("search error"))

	uc := newTestTaskUC(nil, nil, vectorRepo)
	_, err := uc.AnswerQuery(context.Background(), model.Scope{UserID: "u1"}, task.QueryInput{Query: "test"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to search")
}

func TestAnswerQuery_LLMError(t *testing.T) {
	repo := new(mockMemosRepo)
	vectorRepo := new(mockVectorRepo)

	vectorRepo.On("SearchTasks", mock.Anything, mock.Anything).Return([]repository.SearchResult{
		{MemoID: "memo-1", Score: 0.9, Payload: map[string]interface{}{"content": "c"}},
	}, nil)
	repo.On("GetTask", mock.Anything, "memo-1").Return(model.Task{
		ID: "memo-1", MemoURL: "http://localhost:5230/m/memo-1", Content: "content",
	}, nil)

	mgr := makeLLMManagerErr(errors.New("LLM error"))
	uc := newTestTaskUC(mgr, repo, vectorRepo)

	_, err := uc.AnswerQuery(context.Background(), model.Scope{UserID: "u1"}, task.QueryInput{Query: "test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LLM failed")
}

// Tests: buildTaskParsingPrompt

func TestBuildTaskParsingPrompt(t *testing.T) {
	prompt := buildTaskParsingPrompt("Buy milk", "2025-06-15T10:00:00+07:00")
	assert.Contains(t, prompt, "Buy milk")
	assert.Contains(t, prompt, "2025-06-15T10:00:00+07:00")
	assert.Contains(t, prompt, "JSON array")
}

// Tests: parseInputWithLLM

func TestParseInputWithLLM_ValidJSON(t *testing.T) {
	llmResp := `[{"title":"Buy milk","description":"from store","due_date_absolute":"2025-06-15T23:59:59+07:00","priority":"p2","tags":["#type/shopping"],"estimated_duration_minutes":30}]`
	mgr := makeLLMManager(llmResp)
	uc := newTestTaskUC(mgr, nil, nil)

	tasks, err := uc.parseInputWithLLM(context.Background(), "Buy milk")
	assert.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "Buy milk", tasks[0].Title)
	assert.Equal(t, "p2", tasks[0].Priority)
}

func TestParseInputWithLLM_WrappedInCodeFence(t *testing.T) {
	llmResp := "```json\n[{\"title\":\"Test\",\"description\":\"\",\"due_date_absolute\":\"2025-01-01T00:00:00Z\",\"priority\":\"p2\",\"tags\":[],\"estimated_duration_minutes\":60}]\n```"
	mgr := makeLLMManager(llmResp)
	uc := newTestTaskUC(mgr, nil, nil)

	tasks, err := uc.parseInputWithLLM(context.Background(), "Test")
	assert.NoError(t, err)
	assert.Len(t, tasks, 1)
}

func TestParseInputWithLLM_InvalidJSON(t *testing.T) {
	mgr := makeLLMManager("not JSON at all")
	uc := newTestTaskUC(mgr, nil, nil)

	_, err := uc.parseInputWithLLM(context.Background(), "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse LLM JSON")
}

func TestParseInputWithLLM_LLMError(t *testing.T) {
	mgr := makeLLMManagerErr(errors.New("timeout"))
	uc := newTestTaskUC(mgr, nil, nil)

	_, err := uc.parseInputWithLLM(context.Background(), "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LLM request failed")
}

func TestParseInputWithLLM_EmptyParts(t *testing.T) {
	client := &mockGeminiClient{
		response: &gemini.Response{
			Content: gemini.Content{Parts: []gemini.Part{}},
			Usage:   &gemini.Usage{},
		},
	}
	mgr := createManagerFromGeminiClient(client, &mockLogger{})
	uc := newTestTaskUC(mgr, nil, nil)

	_, err := uc.parseInputWithLLM(context.Background(), "test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty response")
}
