package telegram_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"autonomous-task-management/internal/agent"
	"autonomous-task-management/internal/agent/orchestrator"
	"autonomous-task-management/internal/checklist"
	"autonomous-task-management/internal/model"
	"autonomous-task-management/internal/task"
	"autonomous-task-management/internal/task/delivery/telegram"
	"autonomous-task-management/internal/task/repository"
	"autonomous-task-management/pkg/gemini"
	pkgTelegram "autonomous-task-management/pkg/telegram"
)

// ── Mocks ──────────────────────────────────────────────────────────────────

type mockLogger struct{}

func (m *mockLogger) Debug(ctx context.Context, args ...interface{})                  {}
func (m *mockLogger) Debugf(ctx context.Context, format string, args ...interface{})  {}
func (m *mockLogger) Info(ctx context.Context, args ...interface{})                   {}
func (m *mockLogger) Infof(ctx context.Context, format string, args ...interface{})   {}
func (m *mockLogger) Warn(ctx context.Context, args ...interface{})                   {}
func (m *mockLogger) Warnf(ctx context.Context, format string, args ...interface{})   {}
func (m *mockLogger) Error(ctx context.Context, args ...interface{})                  {}
func (m *mockLogger) Errorf(ctx context.Context, format string, args ...interface{})  {}
func (m *mockLogger) DPanic(ctx context.Context, args ...interface{})                 {}
func (m *mockLogger) DPanicf(ctx context.Context, format string, args ...interface{}) {}
func (m *mockLogger) Panic(ctx context.Context, args ...interface{})                  {}
func (m *mockLogger) Panicf(ctx context.Context, format string, args ...interface{})  {}
func (m *mockLogger) Fatal(ctx context.Context, args ...interface{})                  {}
func (m *mockLogger) Fatalf(ctx context.Context, format string, args ...interface{})  {}

type mockTaskUseCase struct {
	createBulkOutput task.CreateBulkOutput
	createBulkErr    error
	searchOutput     task.SearchOutput
	searchErr        error
	queryOutput      task.QueryOutput
	queryErr         error
}

func (m *mockTaskUseCase) CreateBulk(ctx context.Context, sc model.Scope, input task.CreateBulkInput) (task.CreateBulkOutput, error) {
	return m.createBulkOutput, m.createBulkErr
}
func (m *mockTaskUseCase) Search(ctx context.Context, sc model.Scope, input task.SearchInput) (task.SearchOutput, error) {
	return m.searchOutput, m.searchErr
}
func (m *mockTaskUseCase) AnswerQuery(ctx context.Context, sc model.Scope, input task.QueryInput) (task.QueryOutput, error) {
	return m.queryOutput, m.queryErr
}

// mockMemosRepo implements repository.MemosRepository
type mockMemosRepo struct {
	taskResult model.Task
	getErr     error
	updateErr  error
}

func (m *mockMemosRepo) CreateTask(ctx context.Context, opt repository.CreateTaskOptions) (model.Task, error) {
	return model.Task{}, nil
}
func (m *mockMemosRepo) CreateTasksBatch(ctx context.Context, opts []repository.CreateTaskOptions) ([]model.Task, error) {
	return nil, nil
}
func (m *mockMemosRepo) GetTask(ctx context.Context, id string) (model.Task, error) {
	return m.taskResult, m.getErr
}
func (m *mockMemosRepo) ListTasks(ctx context.Context, opt repository.ListTasksOptions) ([]model.Task, error) {
	return nil, nil
}
func (m *mockMemosRepo) UpdateTask(ctx context.Context, id string, content string) error {
	return m.updateErr
}

// mockChecklistSvc implements checklist.Service
type mockChecklistSvc struct {
	stats      checklist.ChecklistStats
	checkboxes []checklist.Checkbox
	updateOut  checklist.UpdateCheckboxOutput
	updateErr  error
}

func (m *mockChecklistSvc) GetStats(content string) checklist.ChecklistStats {
	return m.stats
}
func (m *mockChecklistSvc) ParseCheckboxes(content string) []checklist.Checkbox {
	return m.checkboxes
}
func (m *mockChecklistSvc) UpdateCheckbox(ctx context.Context, input checklist.UpdateCheckboxInput) (checklist.UpdateCheckboxOutput, error) {
	return m.updateOut, m.updateErr
}
func (m *mockChecklistSvc) UpdateAllCheckboxes(content string, checked bool) string {
	return "updated content"
}
func (m *mockChecklistSvc) IsFullyCompleted(content string) bool {
	return false
}

// ── Test Helpers ───────────────────────────────────────────────────────────

type testEnv struct {
	engine           *gin.Engine
	muc              *mockTaskUseCase
	memosRepo        *mockMemosRepo
	checklistSvc     *mockChecklistSvc
	capturedMessages *[]string
}

func newTestEnv(t *testing.T) (*testEnv, *httptest.Server, *httptest.Server) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	capturedMessages := &[]string{}

	tgServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/sendMessage") {
			var payload map[string]interface{}
			json.NewDecoder(r.Body).Decode(&payload)
			if text, ok := payload["text"].(string); ok {
				*capturedMessages = append(*capturedMessages, text)
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))

	llmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := gemini.GenerateResponse{
			Candidates: []gemini.Candidate{
				{
					Content: gemini.Content{
						Parts: []gemini.Part{{Text: "Agent response simulated"}},
					},
				},
			},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))

	l := &mockLogger{}
	bot := pkgTelegram.NewBot("test-token")
	bot.SetAPIURL(tgServer.URL)

	llm := gemini.NewClient("test-key")
	llm.SetAPIURL(llmServer.URL)

	registry := agent.NewToolRegistry()
	agentOrch := orchestrator.New(llm, registry, l, "Asia/Ho_Chi_Minh")

	muc := &mockTaskUseCase{}
	memosRepo := &mockMemosRepo{}
	checklistSvc := &mockChecklistSvc{}

	engine := gin.New()
	h := telegram.New(l, muc, bot, agentOrch, nil, checklistSvc, memosRepo)
	engine.POST("/webhook/telegram", h.HandleWebhook)

	return &testEnv{
		engine:           engine,
		muc:              muc,
		memosRepo:        memosRepo,
		checklistSvc:     checklistSvc,
		capturedMessages: capturedMessages,
	}, tgServer, llmServer
}

func sendWebhook(engine *gin.Engine, text string) *httptest.ResponseRecorder {
	update := pkgTelegram.Update{
		UpdateID: 1,
		Message: &pkgTelegram.Message{
			MessageID: 1,
			Chat:      &pkgTelegram.Chat{ID: 123},
			From:      &pkgTelegram.User{ID: 456},
			Text:      text,
		},
	}
	body, _ := json.Marshal(update)
	req, _ := http.NewRequest(http.MethodPost, "/webhook/telegram", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

func waitForMessages(msgs *[]string, atLeast int, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) && len(*msgs) < atLeast {
		time.Sleep(20 * time.Millisecond)
	}
}

func assertContains(t *testing.T, msgs []string, substr string) {
	t.Helper()
	for _, m := range msgs {
		if strings.Contains(m, substr) {
			return
		}
	}
	t.Errorf("expected a message containing %q, got: %v", substr, msgs)
}

// ── Tests ──────────────────────────────────────────────────────────────────

func TestHandleWebhook_InvalidJSON(t *testing.T) {
	env, tgSrv, llmSrv := newTestEnv(t)
	defer tgSrv.Close()
	defer llmSrv.Close()

	req, _ := http.NewRequest(http.MethodPost, "/webhook/telegram", bytes.NewBufferString("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.engine.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleWebhook_NonMessageUpdate(t *testing.T) {
	env, tgSrv, llmSrv := newTestEnv(t)
	defer tgSrv.Close()
	defer llmSrv.Close()

	update := pkgTelegram.Update{UpdateID: 1, Message: nil}
	body, _ := json.Marshal(update)
	req, _ := http.NewRequest(http.MethodPost, "/webhook/telegram", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	env.engine.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestHandleStart(t *testing.T) {
	env, tgSrv, llmSrv := newTestEnv(t)
	defer tgSrv.Close()
	defer llmSrv.Close()

	w := sendWebhook(env.engine, "/start")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	waitForMessages(env.capturedMessages, 1, 500*time.Millisecond)
	assertContains(t, *env.capturedMessages, "Chào mừng")
}

func TestHandleHelp(t *testing.T) {
	env, tgSrv, llmSrv := newTestEnv(t)
	defer tgSrv.Close()
	defer llmSrv.Close()

	w := sendWebhook(env.engine, "/help")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	waitForMessages(env.capturedMessages, 1, 500*time.Millisecond)
	assertContains(t, *env.capturedMessages, "Hướng dẫn")
}

func TestHandleReset(t *testing.T) {
	env, tgSrv, llmSrv := newTestEnv(t)
	defer tgSrv.Close()
	defer llmSrv.Close()

	w := sendWebhook(env.engine, "/reset")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	waitForMessages(env.capturedMessages, 1, 500*time.Millisecond)
	assertContains(t, *env.capturedMessages, "Đã xóa lịch sử")
}

func TestHandleCreateTask_Success(t *testing.T) {
	env, tgSrv, llmSrv := newTestEnv(t)
	defer tgSrv.Close()
	defer llmSrv.Close()

	env.muc.createBulkOutput = task.CreateBulkOutput{
		TaskCount: 2,
		Tasks: []task.CreatedTask{
			{Title: "Task 1", MemoURL: "http://memos/1"},
			{Title: "Task 2", CalendarLink: "http://cal/2"},
		},
	}
	w := sendWebhook(env.engine, "Họp team lúc 9h sáng mai")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	waitForMessages(env.capturedMessages, 2, 500*time.Millisecond)
	assertContains(t, *env.capturedMessages, "2 task(s)")
}

func TestHandleCreateTask_ZeroTasks(t *testing.T) {
	env, tgSrv, llmSrv := newTestEnv(t)
	defer tgSrv.Close()
	defer llmSrv.Close()

	env.muc.createBulkOutput = task.CreateBulkOutput{TaskCount: 0}
	w := sendWebhook(env.engine, "Blah blah")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	waitForMessages(env.capturedMessages, 2, 500*time.Millisecond)
	assertContains(t, *env.capturedMessages, "Không tìm thấy tasks")
}

func TestHandleCreateTask_ConversationalFallback(t *testing.T) {
	env, tgSrv, llmSrv := newTestEnv(t)
	defer tgSrv.Close()
	defer llmSrv.Close()

	env.muc.createBulkErr = task.ErrNoTasksParsed
	w := sendWebhook(env.engine, "Bạn làm được gì?")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	waitForMessages(env.capturedMessages, 3, 600*time.Millisecond)
	assertContains(t, *env.capturedMessages, "Agent response simulated")
}

func TestHandleCreateTask_OtherError(t *testing.T) {
	env, tgSrv, llmSrv := newTestEnv(t)
	defer tgSrv.Close()
	defer llmSrv.Close()

	env.muc.createBulkErr = task.ErrEmptyInput
	w := sendWebhook(env.engine, "something")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	waitForMessages(env.capturedMessages, 2, 500*time.Millisecond)
	assertContains(t, *env.capturedMessages, "Không thể xử lý")
}

func TestHandleSearch_Success(t *testing.T) {
	env, tgSrv, llmSrv := newTestEnv(t)
	defer tgSrv.Close()
	defer llmSrv.Close()

	env.muc.searchOutput = task.SearchOutput{
		Results: []task.SearchResultItem{
			{Content: "Test meeting task", MemoURL: "http://memos/1", Score: 0.95},
		},
	}
	w := sendWebhook(env.engine, "/search meeting")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	waitForMessages(env.capturedMessages, 2, 500*time.Millisecond)
	assertContains(t, *env.capturedMessages, "Tìm thấy")
}

func TestHandleSearch_EmptyQuery(t *testing.T) {
	env, tgSrv, llmSrv := newTestEnv(t)
	defer tgSrv.Close()
	defer llmSrv.Close()

	w := sendWebhook(env.engine, "/search ")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	waitForMessages(env.capturedMessages, 1, 500*time.Millisecond)
	assertContains(t, *env.capturedMessages, "Vui lòng nhập từ khóa")
}

func TestHandleSearch_Error(t *testing.T) {
	env, tgSrv, llmSrv := newTestEnv(t)
	defer tgSrv.Close()
	defer llmSrv.Close()

	env.muc.searchErr = task.ErrEmptyQuery
	w := sendWebhook(env.engine, "/search test")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	waitForMessages(env.capturedMessages, 2, 500*time.Millisecond)
	assertContains(t, *env.capturedMessages, "Lỗi tìm kiếm")
}

func TestHandleSearch_NoResults(t *testing.T) {
	env, tgSrv, llmSrv := newTestEnv(t)
	defer tgSrv.Close()
	defer llmSrv.Close()

	env.muc.searchOutput = task.SearchOutput{Results: nil}
	w := sendWebhook(env.engine, "/search nothing")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	waitForMessages(env.capturedMessages, 2, 500*time.Millisecond)
	assertContains(t, *env.capturedMessages, "Không tìm thấy task nào")
}

func TestHandleAsk_Success(t *testing.T) {
	env, tgSrv, llmSrv := newTestEnv(t)
	defer tgSrv.Close()
	defer llmSrv.Close()

	w := sendWebhook(env.engine, "/ask lịch trình tuần này")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	waitForMessages(env.capturedMessages, 2, 600*time.Millisecond)
	assertContains(t, *env.capturedMessages, "Agent response simulated")
}

func TestHandleAsk_EmptyQuery(t *testing.T) {
	env, tgSrv, llmSrv := newTestEnv(t)
	defer tgSrv.Close()
	defer llmSrv.Close()

	w := sendWebhook(env.engine, "/ask ")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	waitForMessages(env.capturedMessages, 2, 500*time.Millisecond)
	assertContains(t, *env.capturedMessages, "Vui lòng nhập câu hỏi")
}

func TestHandleProgress_EmptyID(t *testing.T) {
	env, tgSrv, llmSrv := newTestEnv(t)
	defer tgSrv.Close()
	defer llmSrv.Close()

	w := sendWebhook(env.engine, "/progress ")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	waitForMessages(env.capturedMessages, 1, 500*time.Millisecond)
	assertContains(t, *env.capturedMessages, "Vui lòng nhập task ID")
}

func TestHandleProgress_Success(t *testing.T) {
	env, tgSrv, llmSrv := newTestEnv(t)
	defer tgSrv.Close()
	defer llmSrv.Close()

	env.memosRepo.taskResult = model.Task{ID: "task-1", Content: "- [ ] item1\n- [x] item2"}
	env.checklistSvc.stats = checklist.ChecklistStats{Total: 2, Completed: 1, Pending: 1, Progress: 50}
	env.checklistSvc.checkboxes = []checklist.Checkbox{
		{Text: "item1", Checked: false},
		{Text: "item2", Checked: true},
	}
	w := sendWebhook(env.engine, "/progress task-1")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	waitForMessages(env.capturedMessages, 2, 500*time.Millisecond)
	assertContains(t, *env.capturedMessages, "Tiến độ")
}

func TestHandleProgress_TaskNotFound(t *testing.T) {
	env, tgSrv, llmSrv := newTestEnv(t)
	defer tgSrv.Close()
	defer llmSrv.Close()

	env.memosRepo.getErr = task.ErrMemoCreate
	w := sendWebhook(env.engine, "/progress bad-id")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	waitForMessages(env.capturedMessages, 2, 500*time.Millisecond)
	assertContains(t, *env.capturedMessages, "Không thể lấy tiến độ")
}

func TestHandleComplete_EmptyID(t *testing.T) {
	env, tgSrv, llmSrv := newTestEnv(t)
	defer tgSrv.Close()
	defer llmSrv.Close()

	w := sendWebhook(env.engine, "/complete ")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	waitForMessages(env.capturedMessages, 1, 500*time.Millisecond)
	assertContains(t, *env.capturedMessages, "Vui lòng nhập task ID")
}

func TestHandleComplete_Success(t *testing.T) {
	env, tgSrv, llmSrv := newTestEnv(t)
	defer tgSrv.Close()
	defer llmSrv.Close()

	env.memosRepo.taskResult = model.Task{ID: "task-1", Content: "- [ ] item1"}
	w := sendWebhook(env.engine, "/complete task-1")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	waitForMessages(env.capturedMessages, 2, 500*time.Millisecond)
	assertContains(t, *env.capturedMessages, "Đã đánh dấu")
}

func TestHandleComplete_TaskNotFound(t *testing.T) {
	env, tgSrv, llmSrv := newTestEnv(t)
	defer tgSrv.Close()
	defer llmSrv.Close()

	env.memosRepo.getErr = task.ErrMemoCreate
	w := sendWebhook(env.engine, "/complete bad-id")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	waitForMessages(env.capturedMessages, 2, 500*time.Millisecond)
	assertContains(t, *env.capturedMessages, "Không thể đánh dấu")
}

func TestHandleComplete_UpdateError(t *testing.T) {
	env, tgSrv, llmSrv := newTestEnv(t)
	defer tgSrv.Close()
	defer llmSrv.Close()

	env.memosRepo.taskResult = model.Task{ID: "task-1", Content: "- [ ] item1"}
	env.memosRepo.updateErr = task.ErrMemoCreate
	w := sendWebhook(env.engine, "/complete task-1")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	waitForMessages(env.capturedMessages, 2, 500*time.Millisecond)
	assertContains(t, *env.capturedMessages, "Không thể hoàn thành task")
}

func TestHandleCheckItem_BadFormat(t *testing.T) {
	env, tgSrv, llmSrv := newTestEnv(t)
	defer tgSrv.Close()
	defer llmSrv.Close()

	w := sendWebhook(env.engine, "/check task-1")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	waitForMessages(env.capturedMessages, 1, 500*time.Millisecond)
	assertContains(t, *env.capturedMessages, "Vui lòng nhập đầy đủ")
}

func TestHandleCheckItem_TaskNotFound(t *testing.T) {
	env, tgSrv, llmSrv := newTestEnv(t)
	defer tgSrv.Close()
	defer llmSrv.Close()

	env.memosRepo.getErr = task.ErrMemoCreate
	w := sendWebhook(env.engine, "/check bad-id Write tests")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	waitForMessages(env.capturedMessages, 2, 500*time.Millisecond)
	assertContains(t, *env.capturedMessages, "Không thấy task")
}

func TestHandleCheckItem_Success(t *testing.T) {
	env, tgSrv, llmSrv := newTestEnv(t)
	defer tgSrv.Close()
	defer llmSrv.Close()

	env.memosRepo.taskResult = model.Task{ID: "task-1", Content: "- [ ] Write tests"}
	env.checklistSvc.updateOut = checklist.UpdateCheckboxOutput{Content: "- [x] Write tests", Updated: true, Count: 1}
	w := sendWebhook(env.engine, "/check task-1 Write tests")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	waitForMessages(env.capturedMessages, 2, 500*time.Millisecond)
	assertContains(t, *env.capturedMessages, "Đã cập nhật")
}

func TestHandleCheckItem_NotFound(t *testing.T) {
	env, tgSrv, llmSrv := newTestEnv(t)
	defer tgSrv.Close()
	defer llmSrv.Close()

	env.memosRepo.taskResult = model.Task{ID: "task-1", Content: "- [ ] Other item"}
	env.checklistSvc.updateOut = checklist.UpdateCheckboxOutput{Updated: false, Count: 0}
	w := sendWebhook(env.engine, "/check task-1 Write tests")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	waitForMessages(env.capturedMessages, 2, 500*time.Millisecond)
	assertContains(t, *env.capturedMessages, "Không tìm thấy checkbox")
}

func TestHandleCheckItem_MultipleMatches(t *testing.T) {
	env, tgSrv, llmSrv := newTestEnv(t)
	defer tgSrv.Close()
	defer llmSrv.Close()

	env.memosRepo.taskResult = model.Task{ID: "task-1", Content: "- [ ] item\n- [ ] item"}
	env.checklistSvc.updateOut = checklist.UpdateCheckboxOutput{Content: "- [x] item\n- [x] item", Updated: true, Count: 2}
	w := sendWebhook(env.engine, "/check task-1 item")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	waitForMessages(env.capturedMessages, 2, 500*time.Millisecond)
	assertContains(t, *env.capturedMessages, "Lưu ý")
}

func TestHandleCheckItem_UpdateError(t *testing.T) {
	env, tgSrv, llmSrv := newTestEnv(t)
	defer tgSrv.Close()
	defer llmSrv.Close()

	env.memosRepo.taskResult = model.Task{ID: "task-1", Content: "- [ ] Write tests"}
	env.checklistSvc.updateErr = task.ErrMemoCreate
	w := sendWebhook(env.engine, "/check task-1 Write tests")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	waitForMessages(env.capturedMessages, 2, 500*time.Millisecond)
	assertContains(t, *env.capturedMessages, "Không thể cập nhật")
}

func TestHandleCheckItem_SaveError(t *testing.T) {
	env, tgSrv, llmSrv := newTestEnv(t)
	defer tgSrv.Close()
	defer llmSrv.Close()

	env.memosRepo.taskResult = model.Task{ID: "task-1", Content: "- [ ] Write tests"}
	env.checklistSvc.updateOut = checklist.UpdateCheckboxOutput{Content: "updated", Updated: true, Count: 1}
	env.memosRepo.updateErr = task.ErrMemoCreate
	w := sendWebhook(env.engine, "/check task-1 Write tests")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	waitForMessages(env.capturedMessages, 2, 500*time.Millisecond)
	assertContains(t, *env.capturedMessages, "Không thể hoàn thành check task")
}

func TestHandleUncheckItem_BadFormat(t *testing.T) {
	env, tgSrv, llmSrv := newTestEnv(t)
	defer tgSrv.Close()
	defer llmSrv.Close()

	w := sendWebhook(env.engine, "/uncheck task-1")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	waitForMessages(env.capturedMessages, 1, 500*time.Millisecond)
	assertContains(t, *env.capturedMessages, "Vui lòng nhập đầy đủ")
}

func TestHandleUncheckItem_Success(t *testing.T) {
	env, tgSrv, llmSrv := newTestEnv(t)
	defer tgSrv.Close()
	defer llmSrv.Close()

	env.memosRepo.taskResult = model.Task{ID: "task-1", Content: "- [x] Write tests"}
	env.checklistSvc.updateOut = checklist.UpdateCheckboxOutput{Content: "- [ ] Write tests", Updated: true, Count: 1}
	w := sendWebhook(env.engine, "/uncheck task-1 Write tests")
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	waitForMessages(env.capturedMessages, 2, 500*time.Millisecond)
	assertContains(t, *env.capturedMessages, "Đã cập nhật")
}
