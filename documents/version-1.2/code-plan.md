# 💻 CODE PLAN VERSION 1.2 - TESTING & OPTIMIZATION

## TỔNG QUAN

Version 1.2 tập trung vào **xây dựng test infrastructure** để đạt coverage ≥80% và đảm bảo code quality.

**Điểm quan trọng:** Session memory và sliding window ĐÃ ĐƯỢC IMPLEMENT và hoạt động tốt. Chúng ta KHÔNG cần refactor, chỉ cần viết tests để verify chúng hoạt động đúng.

---

## PHẦN 1: HIỆN TRẠNG IMPLEMENTATION (ĐÃ CÓ)

### 1.1. Session Memory Management ✅

**Location:** `internal/agent/usecase/new.go` và `helpers.go`

**Current Implementation:**
```go
type implUseCase struct {
    llm          llmprovider.IManager
    registry     *agent.ToolRegistry
    l            pkgLog.Logger
    timezone     string
    sessionCache map[string]*agent.SessionMemory  // ✅ Đã có
    cacheMutex   sync.RWMutex                     // ✅ Thread-safe
    cacheTTL     time.Duration                    // ✅ TTL = 10 phút
}

func New(...) agent.UseCase {
    uc := &implUseCase{
        // ...
        sessionCache: make(map[string]*agent.SessionMemory),
        cacheTTL:     10 * time.Minute,
    }
    
    go uc.cleanupExpiredSessions()  // ✅ Background cleanup
    
    return uc
}
```

**Features đã có:**
- ✅ Background goroutine chạy mỗi 5 phút
- ✅ TTL-based expiration (10 phút)
- ✅ Mutex-protected concurrent access
- ✅ Logging cleanup metrics

**Không cần thay đổi!**

### 1.2. Sliding Window ✅

**Location:** `internal/agent/usecase/process_query.go`

**Current Implementation:**
```go
// Save to session history
uc.cacheMutex.Lock()
session.Messages = append(session.Messages, userMessage)
session.Messages = append(session.Messages, llmprovider.Message{
    Role:  "assistant",
    Parts: []llmprovider.Part{{Text: part.Text}},
})

// ✅ Sliding window - giữ tối đa 10 messages (5 turns)
if len(session.Messages) > MaxSessionHistory {
    session.Messages = session.Messages[len(session.Messages)-MaxSessionHistory:]
}
session.LastUpdated = time.Now()
uc.cacheMutex.Unlock()
```

**Configuration:**
- `MaxSessionHistory = 10` (defined in `constant.go`)
- Applied after each successful agent response

**Không cần thay đổi!**

### 1.3. Multi-Provider LLM Manager ✅

**Location:** `pkg/llmprovider/llmprovider.go`

**Architecture:**
- Provider Manager với fallback logic
- 3 providers: DeepSeek (primary) → Gemini → Qwen
- Retry mechanism với exponential backoff
- Global timeout cho entire fallback chain

**Đã có tests:** `pkg/llmprovider/manager_test.go`

**Không cần thay đổi!**

---

## PHẦN 2: NHIỆM VỤ VERSION 1.2 (TODO)

### 🎯 Objective: Viết Tests Để Đạt Coverage ≥80%

**Ưu tiên:**
1. E2E tests cho Telegram webhook (HIGH)
2. Unit tests cho Agent ProcessQuery (HIGH)
3. Unit tests cho Router Classify (HIGH)
4. Integration tests (MEDIUM)
5. Performance benchmarks (LOW)

---

## 2.1. E2E TESTS CHO TELEGRAM WEBHOOK (Priority 1)

### 📁 File: `internal/task/delivery/telegram/handler_test.go` (🆕 NEW)

**Mục tiêu:** Test full flow từ webhook đến response

**Test Cases:**

#### Test 1: Router Classification - CREATE_TASK Intent

```go
package telegram_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"autonomous-task-management/internal/agent"
	"autonomous-task-management/internal/model"
	"autonomous-task-management/internal/router"
	"autonomous-task-management/internal/task/delivery/telegram"
	"autonomous-task-management/pkg/llmprovider"
	pkgLog "autonomous-task-management/pkg/log"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockBot simulates Telegram bot
type MockBot struct {
	mock.Mock
}

func (m *MockBot) SendMessageHTML(chatID int64, text string) error {
	args := m.Called(chatID, text)
	return args.Error(0)
}

// MockRouter simulates semantic router
type MockRouter struct {
	mock.Mock
}

func (m *MockRouter) Classify(ctx context.Context, message string, history []string) (router.RouterOutput, error) {
	args := m.Called(ctx, message, history)
	return args.Get(0).(router.RouterOutput), args.Error(1)
}

// MockAgent simulates agent usecase
type MockAgent struct {
	mock.Mock
}

func (m *MockAgent) ProcessQuery(ctx context.Context, sc model.Scope, query string) (string, error) {
	args := m.Called(ctx, sc, query)
	return args.String(0), args.Error(1)
}

func (m *MockAgent) GetSessionMessages(userID string) []llmprovider.Message {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]llmprovider.Message)
}

func (m *MockAgent) ClearSession(userID string) {
	m.Called(userID)
}

func TestTelegramWebhook_CreateTaskIntent(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	
	mockBot := new(MockBot)
	mockRouter := new(MockRouter)
	mockAgent := new(MockAgent)
	logger := pkgLog.NewLogger()
	
	handler := telegram.NewHandler(logger, mockAgent, mockBot, mockRouter)
	router.POST("/webhook/telegram", handler.HandleWebhook)
	
	// Mock expectations
	mockAgent.On("GetSessionMessages", "telegram_12345").Return([]llmprovider.Message{})
	mockRouter.On("Classify", mock.Anything, "Tạo task: Hoàn thành báo cáo", []string{}).
		Return(router.RouterOutput{
			Intent:     router.IntentCreateTask,
			Confidence: 95,
			Reasoning:  "User wants to create a task",
		}, nil)
	mockAgent.On("ProcessQuery", mock.Anything, mock.Anything, "Tạo task: Hoàn thành báo cáo").
		Return("Task đã được tạo thành công!", nil)
	mockBot.On("SendMessageHTML", int64(12345), mock.Anything).Return(nil)
	
	// Prepare webhook payload
	payload := map[string]interface{}{
		"update_id": 123456,
		"message": map[string]interface{}{
			"message_id": 1,
			"from": map[string]interface{}{
				"id": 12345, "first_name": "TestUser",
			},
			"chat": map[string]interface{}{
				"id": 12345, "type": "private",
			},
			"date": 1708780000,
			"text": "Tạo task: Hoàn thành báo cáo",
		},
	}
	body, _ := json.Marshal(payload)
	
	// Execute request
	req, _ := http.NewRequest(http.MethodPost, "/webhook/telegram", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	mockRouter.AssertExpectations(t)
	mockAgent.AssertExpectations(t)
	mockBot.AssertExpectations(t)
}
```

#### Test 2: Router Classification - SEARCH_TASK Intent

```go
func TestTelegramWebhook_SearchTaskIntent(t *testing.T) {
	// Setup (tương tự Test 1)
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	
	mockBot := new(MockBot)
	mockRouter := new(MockRouter)
	mockAgent := new(MockAgent)
	logger := pkgLog.NewLogger()
	
	handler := telegram.NewHandler(logger, mockAgent, mockBot, mockRouter)
	router.POST("/webhook/telegram", handler.HandleWebhook)
	
	// Mock expectations
	mockAgent.On("GetSessionMessages", "telegram_12345").Return([]llmprovider.Message{})
	mockRouter.On("Classify", mock.Anything, "Tìm task về báo cáo", []string{}).
		Return(router.RouterOutput{
			Intent:     router.IntentSearchTask,
			Confidence: 90,
			Reasoning:  "User wants to search tasks",
		}, nil)
	mockAgent.On("ProcessQuery", mock.Anything, mock.Anything, "Tìm task về báo cáo").
		Return("Tìm thấy 3 tasks liên quan đến báo cáo", nil)
	mockBot.On("SendMessageHTML", int64(12345), mock.Anything).Return(nil)
	
	// Prepare payload
	payload := map[string]interface{}{
		"update_id": 123457,
		"message": map[string]interface{}{
			"message_id": 2,
			"from":       map[string]interface{}{"id": 12345, "first_name": "TestUser"},
			"chat":       map[string]interface{}{"id": 12345, "type": "private"},
			"date":       1708780100,
			"text":       "Tìm task về báo cáo",
		},
	}
	body, _ := json.Marshal(payload)
	
	// Execute
	req, _ := http.NewRequest(http.MethodPost, "/webhook/telegram", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	// Assertions
	assert.Equal(t, http.StatusOK, w.Code)
	mockRouter.AssertExpectations(t)
	mockAgent.AssertExpectations(t)
	mockBot.AssertExpectations(t)
}
```

#### Test 3: Session Memory Persistence

```go
func TestTelegramWebhook_SessionPersistence(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	
	mockBot := new(MockBot)
	mockRouter := new(MockRouter)
	mockAgent := new(MockAgent)
	logger := pkgLog.NewLogger()
	
	handler := telegram.NewHandler(logger, mockAgent, mockBot, mockRouter)
	router.POST("/webhook/telegram", handler.HandleWebhook)
	
	// First message - no history
	mockAgent.On("GetSessionMessages", "telegram_12345").Return([]llmprovider.Message{}).Once()
	mockRouter.On("Classify", mock.Anything, "Tạo task mới", []string{}).
		Return(router.RouterOutput{Intent: router.IntentCreateTask, Confidence: 95}, nil).Once()
	mockAgent.On("ProcessQuery", mock.Anything, mock.Anything, "Tạo task mới").
		Return("Task created", nil).Once()
	mockBot.On("SendMessageHTML", int64(12345), mock.Anything).Return(nil).Once()
	
	// Send first message
	payload1 := map[string]interface{}{
		"update_id": 1,
		"message": map[string]interface{}{
			"message_id": 1,
			"from":       map[string]interface{}{"id": 12345},
			"chat":       map[string]interface{}{"id": 12345},
			"text":       "Tạo task mới",
		},
	}
	body1, _ := json.Marshal(payload1)
	req1, _ := http.NewRequest(http.MethodPost, "/webhook/telegram", bytes.NewBuffer(body1))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	
	assert.Equal(t, http.StatusOK, w1.Code)
	
	// Second message - with history
	history := []llmprovider.Message{
		{Role: "user", Parts: []llmprovider.Part{{Text: "Tạo task mới"}}},
		{Role: "assistant", Parts: []llmprovider.Part{{Text: "Task created"}}},
	}
	mockAgent.On("GetSessionMessages", "telegram_12345").Return(history).Once()
	mockRouter.On("Classify", mock.Anything, "Xem task vừa tạo", mock.Anything).
		Return(router.RouterOutput{Intent: router.IntentSearchTask, Confidence: 92}, nil).Once()
	mockAgent.On("ProcessQuery", mock.Anything, mock.Anything, "Xem task vừa tạo").
		Return("Đây là task vừa tạo", nil).Once()
	mockBot.On("SendMessageHTML", int64(12345), mock.Anything).Return(nil).Once()
	
	// Send second message
	payload2 := map[string]interface{}{
		"update_id": 2,
		"message": map[string]interface{}{
			"message_id": 2,
			"from":       map[string]interface{}{"id": 12345},
			"chat":       map[string]interface{}{"id": 12345},
			"text":       "Xem task vừa tạo",
		},
	}
	body2, _ := json.Marshal(payload2)
	req2, _ := http.NewRequest(http.MethodPost, "/webhook/telegram", bytes.NewBuffer(body2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	
	assert.Equal(t, http.StatusOK, w2.Code)
	mockAgent.AssertExpectations(t)
}
```

---

## 2.2. UNIT TESTS CHO AGENT PROCESSQUERY (Priority 2)

### 📁 File: `internal/agent/usecase/process_query_test.go` (🆕 NEW)

**Test Cases:**

#### Test 1: Agent ReAct Loop - Single Step

```go
package usecase_test

import (
	"context"
	"testing"

	"autonomous-task-management/internal/agent"
	"autonomous-task-management/internal/agent/usecase"
	"autonomous-task-management/internal/model"
	"autonomous-task-management/pkg/llmprovider"
	pkgLog "autonomous-task-management/pkg/log"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockLLMManager struct {
	mock.Mock
}

func (m *MockLLMManager) GenerateContent(ctx context.Context, req *llmprovider.Request) (*llmprovider.Response, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*llmprovider.Response), args.Error(1)
}

type MockTool struct {
	mock.Mock
}

func (m *MockTool) Name() string {
	return "test_tool"
}

func (m *MockTool) Description() string {
	return "A test tool"
}

func (m *MockTool) Parameters() map[string]interface{} {
	return map[string]interface{}{}
}

func (m *MockTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	args := m.Called(ctx, params)
	return args.Get(0), args.Error(1)
}

func TestProcessQuery_SingleStep_DirectAnswer(t *testing.T) {
	// Setup
	mockLLM := new(MockLLMManager)
	registry := agent.NewToolRegistry()
	logger := pkgLog.NewLogger()
	
	uc := usecase.New(mockLLM, registry, logger, "Asia/Ho_Chi_Minh")
	
	// Mock LLM response - direct answer without tool call
	mockLLM.On("GenerateContent", mock.Anything, mock.Anything).Return(&llmprovider.Response{
		Content: llmprovider.Content{
			Parts: []llmprovider.Part{
				{Text: "Xin chào! Tôi có thể giúp bạn quản lý tasks."},
			},
		},
	}, nil)
	
	// Execute
	sc := model.Scope{UserID: "test_user_123"}
	response, err := uc.ProcessQuery(context.Background(), sc, "Xin chào")
	
	// Assertions
	assert.NoError(t, err)
	assert.Contains(t, response, "Xin chào")
	mockLLM.AssertExpectations(t)
}
```

#### Test 2: Agent ReAct Loop - With Tool Call

```go
func TestProcessQuery_WithToolCall(t *testing.T) {
	// Setup
	mockLLM := new(MockLLMManager)
	mockTool := new(MockTool)
	registry := agent.NewToolRegistry()
	registry.Register(mockTool)
	logger := pkgLog.NewLogger()
	
	uc := usecase.New(mockLLM, registry, logger, "Asia/Ho_Chi_Minh")
	
	// Mock LLM first response - wants to call tool
	mockLLM.On("GenerateContent", mock.Anything, mock.Anything).Return(&llmprovider.Response{
		Content: llmprovider.Content{
			Parts: []llmprovider.Part{
				{
					FunctionCall: &llmprovider.FunctionCall{
						Name: "test_tool",
						Args: map[string]interface{}{"query": "test"},
					},
				},
			},
		},
	}, nil).Once()
	
	// Mock tool execution
	mockTool.On("Execute", mock.Anything, mock.Anything).Return(map[string]string{
		"result": "Tool executed successfully",
	}, nil)
	
	// Mock LLM second response - final answer after tool
	mockLLM.On("GenerateContent", mock.Anything, mock.Anything).Return(&llmprovider.Response{
		Content: llmprovider.Content{
			Parts: []llmprovider.Part{
				{Text: "Based on tool result: Tool executed successfully"},
			},
		},
	}, nil).Once()
	
	// Execute
	sc := model.Scope{UserID: "test_user_456"}
	response, err := uc.ProcessQuery(context.Background(), sc, "Run test tool")
	
	// Assertions
	assert.NoError(t, err)
	assert.Contains(t, response, "Tool executed successfully")
	mockLLM.AssertExpectations(t)
	mockTool.AssertExpectations(t)
}
```

#### Test 3: Agent Max Steps Exceeded

```go
func TestProcessQuery_MaxStepsExceeded(t *testing.T) {
	// Setup
	mockLLM := new(MockLLMManager)
	registry := agent.NewToolRegistry()
	logger := pkgLog.NewLogger()
	
	uc := usecase.New(mockLLM, registry, logger, "Asia/Ho_Chi_Minh")
	
	// Mock LLM to always return tool call (infinite loop scenario)
	mockLLM.On("GenerateContent", mock.Anything, mock.Anything).Return(&llmprovider.Response{
		Content: llmprovider.Content{
			Parts: []llmprovider.Part{
				{
					FunctionCall: &llmprovider.FunctionCall{
						Name: "non_existent_tool",
						Args: map[string]interface{}{},
					},
				},
			},
		},
	}, nil)
	
	// Execute
	sc := model.Scope{UserID: "test_user_789"}
	response, err := uc.ProcessQuery(context.Background(), sc, "Infinite loop test")
	
	// Assertions
	assert.NoError(t, err)
	assert.Contains(t, response, "vượt quá số bước cho phép")
	mockLLM.AssertNumberOfCalls(t, "GenerateContent", 5) // MaxAgentSteps = 5
}
```

---

## 2.3. UNIT TESTS CHO SESSION MEMORY (Priority 3)

### 📁 File: `internal/agent/usecase/session_test.go` (🆕 NEW)

```go
package usecase_test

import (
	"testing"
	"time"

	"autonomous-task-management/internal/agent"
	"autonomous-task-management/internal/agent/usecase"
	"autonomous-task-management/pkg/llmprovider"
	pkgLog "autonomous-task-management/pkg/log"

	"github.com/stretchr/testify/assert"
)

func TestGetSessionMessages_EmptySession(t *testing.T) {
	mockLLM := new(MockLLMManager)
	registry := agent.NewToolRegistry()
	logger := pkgLog.NewLogger()
	
	uc := usecase.New(mockLLM, registry, logger, "Asia/Ho_Chi_Minh")
	
	messages := uc.GetSessionMessages("new_user")
	assert.Nil(t, messages)
}

func TestClearSession(t *testing.T) {
	mockLLM := new(MockLLMManager)
	registry := agent.NewToolRegistry()
	logger := pkgLog.NewLogger()
	
	uc := usecase.New(mockLLM, registry, logger, "Asia/Ho_Chi_Minh")
	
	// Simulate session creation by calling ProcessQuery
	// (would need to mock LLM response)
	
	// Clear session
	uc.ClearSession("test_user")
	
	// Verify session is cleared
	messages := uc.GetSessionMessages("test_user")
	assert.Nil(t, messages)
}

func TestSlidingWindow_MaxHistory(t *testing.T) {
	// This test would verify that session history is limited to MaxSessionHistory
	// Would need to simulate multiple ProcessQuery calls and verify message count
	t.Skip("TODO: Implement sliding window test")
}
```

---

## 2.4. COVERAGE REPORTING

### Chạy Tests và Generate Coverage

```bash
# Run all tests with coverage
go test -v -coverprofile=coverage.out ./internal/... ./pkg/...

# View coverage summary
go tool cover -func=coverage.out

# View coverage in browser (visual)
go tool cover -html=coverage.out

# Check total coverage
go tool cover -func=coverage.out | grep total
```

### Target Coverage by Package

| Package | Target | Priority |
|---------|--------|----------|
| `internal/agent/usecase` | ≥85% | HIGH |
| `internal/router/usecase` | ≥85% | HIGH |
| `internal/task/delivery/telegram` | ≥80% | HIGH |
| `internal/checklist/usecase` | ≥75% | MEDIUM |
| `internal/task/usecase` | ≥75% | MEDIUM |
| `pkg/llmprovider` | ≥80% | MEDIUM |
| `pkg/datemath` | ≥90% | LOW (easy) |

---

## 2.5. CI/CD INTEGRATION (Optional)

### GitHub Actions Workflow

```yaml
# .github/workflows/test.yml
name: Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Run tests
        run: go test -v -coverprofile=coverage.out ./internal/... ./pkg/...
      
      - name: Check coverage
        run: |
          COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          echo "Total coverage: $COVERAGE%"
          if (( $(echo "$COVERAGE < 80" | bc -l) )); then
            echo "Coverage is below 80%"
            exit 1
          fi
      
      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./coverage.out
```

---

## TỔNG KẾT & NEXT STEPS

### ✅ Đã Có (Không cần làm)
- Session memory management
- Sliding window
- Background cleanup
- Multi-provider LLM manager

### 📝 Cần Làm (Version 1.2)
1. **E2E tests** cho Telegram webhook (3-5 test cases)
2. **Unit tests** cho Agent ProcessQuery (3-5 test cases)
3. **Unit tests** cho Router Classify (đã có template trong `classify_test.go`)
4. **Session tests** để verify cleanup và sliding window
5. **Coverage reporting** và CI/CD integration

### Thứ Tự Thực Hiện
1. Bắt đầu với E2E tests (quan trọng nhất)
2. Thêm unit tests cho agent logic
3. Verify coverage với `go tool cover`
4. Iterate cho đến khi đạt ≥80%
5. Setup CI/CD (optional)

**Lưu ý:** Tất cả code examples trên đều sử dụng cấu trúc thực tế của repo. Copy-paste và adjust theo nhu cầu!
