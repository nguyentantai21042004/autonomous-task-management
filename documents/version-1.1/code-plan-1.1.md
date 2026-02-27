# CODE PLAN VERSION 1.1 - DETAILED IMPLEMENTATION GUIDE

> **Mục tiêu**: Triển khai 5 cải tiến cốt lõi để nâng cấp UX và độ tin cậy của hệ thống ATM lên tầm cao mới.

---

## TỔNG QUAN KIẾN TRÚC

### Cấu trúc Code hiện tại (Baseline)

```
internal/
├── agent/
│   ├── orchestrator/
│   │   ├── orchestrator.go      # ProcessQuery() - Entry point cho agent
│   │   ├── types.go              # SessionMemory struct
│   │   └── new.go
│   └── tools/                    # Agent tools (search, calendar, checklist)
├── task/
│   ├── delivery/telegram/
│   │   ├── handler.go            # processMessage() - Routing logic
│   │   └── new.go
│   └── usecase/
│       ├── search.go             # Search() - Đã có self-healing (HOTFIX 4)
│       └── answer_query.go       # AnswerQuery() - RAG logic
pkg/
├── telegram/
│   └── bot.go                    # SendMessage(), SendMessageWithMode()
└── gemini/
    └── client.go                 # GenerateContent()
```

### Các thành phần mới cần tạo

```
internal/
├── router/                       # 🆕 Semantic Router
│   ├── router.go                 # SemanticRouter struct + Classify()
│   ├── types.go                  # Intent enum, RouterOutput
│   └── new.go                    # Constructor
└── agent/orchestrator/
    └── time_context.go           # 🆕 Time injection utilities
```

---

## 1. OMNI-ROUTER (Semantic Routing)

### 📋 Checklist Implementation

- [ ] Tạo package `internal/router`
- [ ] Định nghĩa Intent types và RouterOutput struct
- [ ] Implement SemanticRouter với Gemini Flash + Structured Output
- [ ] Tích hợp vào Telegram handler
- [ ] Viết unit tests
- [ ] Update handler để fallback slash commands (backward compatibility)

### 🔍 Industry Standard

Pattern này được gọi là **Semantic Routing** (như thư viện `semantic-router`). Thay vì dùng regex, họ đưa tin nhắn qua một LLM siêu nhanh (Gemini Flash hoặc Claude Haiku) với **Structured Outputs (Ép kiểu JSON)**. LLM bị API ép buộc trả về đúng một struct JSON định sẵn. Điều này đảm bảo tốc độ < 500ms và độ chính xác > 98%.


### 📁 File: `internal/router/types.go` (🆕 New File)

**Convention**: All Input/Output structs must be in `types.go` at module root (per `convention.md`)

```go
package router

// Intent represents user's intention
type Intent string

const (
	IntentCreateTask      Intent = "CREATE_TASK"
	IntentSearchTask      Intent = "SEARCH_TASK"
	IntentManageChecklist Intent = "MANAGE_CHECKLIST"
	IntentConversation    Intent = "CONVERSATION"
)

// RouterOutput is the structured response from Semantic Router
type RouterOutput struct {
	Intent     Intent `json:"intent"`
	Confidence int    `json:"confidence"` // 0-100
	Reasoning  string `json:"reasoning"`  // Optional: Why this intent was chosen
}
```

### 📁 File: `internal/router/router.go` (🆕 New File)

**Convention**: Logic files contain ONLY method implementations, no type definitions

```go
package router

import (
	"context"
	"encoding/json"
	"fmt"

	"autonomous-task-management/pkg/gemini"
	"autonomous-task-management/pkg/log"
)

// SemanticRouter classifies user intent using LLM
type SemanticRouter struct {
	llm *gemini.Client
	l   log.Logger
}

// Classify determines user intent from message
// Convention: Method accepts context.Context as first parameter
func (r *SemanticRouter) Classify(ctx context.Context, message string, conversationHistory []string) (RouterOutput, error) {
	// Build prompt with conversation history
	historyContext := ""
	if len(conversationHistory) > 0 {
		historyContext = "Lịch sử hội thoại gần đây:\n"
		for i, msg := range conversationHistory {
			historyContext += fmt.Sprintf("%d. %s\n", i+1, msg)
		}
		historyContext += "\n"
	}

	prompt := fmt.Sprintf(`%sBạn là Semantic Router. Phân tích tin nhắn sau và xác định ý định (intent) của người dùng.

Tin nhắn hiện tại: "%s"

Các intent có thể:
1. CREATE_TASK: Tạo task mới, thêm công việc, nhắc nhở, deadline
2. SEARCH_TASK: Tìm kiếm, tra cứu, xem task cũ
3. MANAGE_CHECKLIST: Đánh dấu hoàn thành, check/uncheck, xem tiến độ
4. CONVERSATION: Chào hỏi, hỏi về tính năng, chat thông thường

Trả về JSON với format:
{
  "intent": "CREATE_TASK|SEARCH_TASK|MANAGE_CHECKLIST|CONVERSATION",
  "confidence": 0-100,
  "reasoning": "Giải thích ngắn gọn"
}`, historyContext, message)

	// Call Gemini with structured output
	resp, err := r.llm.GenerateContent(ctx, gemini.GenerateRequest{
		Contents: []gemini.Content{
			{
				Role: "user",
				Parts: []gemini.Part{
					{Text: prompt},
				},
			},
		},
		GenerationConfig: &gemini.GenerationConfig{
			Temperature:      0.1, // Low temperature for consistent routing
			ResponseMIMEType: "application/json",
		},
	})
	if err != nil {
		return RouterOutput{}, fmt.Errorf("router: LLM call failed: %w", err)
	}

	// Parse JSON response
	var output RouterOutput
	if err := json.Unmarshal([]byte(resp.Text), &output); err != nil {
		r.l.Warnf(ctx, "router: Failed to parse JSON, falling back to CONVERSATION: %v", err)
		// 🔧 PRO-TIP #2: Fallback to CONVERSATION (safer than CREATE_TASK)
		// Reason: If JSON parsing fails, better to let agent handle conversationally
		// than force into CREATE_TASK which may cause "no tasks parsed" error
		// This prevents Race Condition where ambiguous messages get forced into task creation
		return RouterOutput{
			Intent:     IntentConversation,
			Confidence: 50,
			Reasoning:  "Fallback due to parsing error - route to conversational agent",
		}, nil
	}

	r.l.Infof(ctx, "router: Classified as %s (confidence: %d%%)", output.Intent, output.Confidence)
	return output, nil
}
```

### 📁 File: `internal/router/new.go` (🆕 New File)

**Convention**: `new.go` is strictly a factory - contains ONLY struct + New() + setters

```go
package router

import (
	"autonomous-task-management/pkg/gemini"
	"autonomous-task-management/pkg/log"
)

// New creates a new SemanticRouter
// Convention: Factory function returns concrete type (not interface) for internal packages
func New(llm *gemini.Client, l log.Logger) *SemanticRouter {
	return &SemanticRouter{
		llm: llm,
		l:   l,
	}
}
```


### 📁 Update: `internal/task/delivery/telegram/handler.go`

**Convention**: Delivery layer handles "How data gets IN and OUT", no business logic

**Thay đổi 1: Add router field to handler struct**

```go
// handler.go - Around line 20
type handler struct {
	l            pkgLog.Logger
	uc           task.UseCase
	bot          *pkgTelegram.Bot
	orchestrator *orchestrator.Orchestrator
	automationUC automation.UseCase
	checklistSvc checklist.Service
	memosRepo    repository.MemosRepository
	router       *router.SemanticRouter // 🆕 Add this field
}
```

**Thay đổi 2: Update processMessage to use router**

**Convention**: 
- Delivery validates strictly, passes quickly, maps errors
- Context.Context as first parameter
- Extract scope from context (not parameter)

```go
// handler.go - processMessage method (around line 71)
func (h *handler) processMessage(ctx context.Context, msg *pkgTelegram.Message) error {
	// Convention: Construct scope from message
	sc := model.Scope{UserID: fmt.Sprintf("telegram_%d", msg.From.ID)}

	// Handle explicit slash commands first (backward compatibility)
	// Convention: Simple switch-case for command routing
	switch {
	case msg.Text == "/start":
		return h.handleStart(ctx, msg.Chat.ID)
	case msg.Text == "/help":
		return h.handleHelp(ctx, msg.Chat.ID)
	case msg.Text == "/reset":
		h.orchestrator.ClearSession(sc.UserID)
		return h.bot.SendMessage(msg.Chat.ID, "✅ Đã xóa lịch sử hội thoại. Bắt đầu lại từ đầu!")
	case strings.HasPrefix(msg.Text, "/search "):
		query := strings.TrimSpace(strings.TrimPrefix(msg.Text, "/search"))
		return h.handleSearch(ctx, sc, query, msg.Chat.ID)
	case strings.HasPrefix(msg.Text, "/ask "):
		query := strings.TrimSpace(strings.TrimPrefix(msg.Text, "/ask"))
		return h.handleAgentOrchestrator(ctx, sc, query, msg.Chat.ID)
	case strings.HasPrefix(msg.Text, "/progress "):
		taskID := strings.TrimSpace(strings.TrimPrefix(msg.Text, "/progress"))
		return h.handleProgress(ctx, sc, taskID, msg.Chat.ID)
	case strings.HasPrefix(msg.Text, "/complete "):
		taskID := strings.TrimSpace(strings.TrimPrefix(msg.Text, "/complete"))
		return h.handleComplete(ctx, sc, taskID, msg.Chat.ID)
	case strings.HasPrefix(msg.Text, "/check "):
		return h.handleCheck(ctx, sc, msg.Text, msg.Chat.ID)
	case strings.HasPrefix(msg.Text, "/uncheck "):
		return h.handleUncheck(ctx, sc, msg.Text, msg.Chat.ID)
	}

	// 🆕 Use Semantic Router for natural language messages
	// Convention: Get conversation history for context
	session := h.orchestrator.GetSession(sc.UserID)
	history := []string{}
	if session != nil && len(session.Messages) > 0 {
		// Get last 3 messages (6 content items = 3 turns)
		start := len(session.Messages) - 6
		if start < 0 {
			start = 0
		}
		for i := start; i < len(session.Messages); i++ {
			if len(session.Messages[i].Parts) > 0 {
				history = append(history, session.Messages[i].Parts[0].Text)
			}
		}
	}

	// Classify intent using router
	// Convention: Pass context as first parameter
	routerOutput, err := h.router.Classify(ctx, msg.Text, history)
	if err != nil {
		h.l.Errorf(ctx, "router: Classification failed, falling back to CONVERSATION: %v", err)
		// 🔧 PRO-TIP #2: Fallback to CONVERSATION (safer than CREATE_TASK)
		routerOutput.Intent = router.IntentConversation
	}

	// Route based on intent
	// Convention: Simple switch-case, delegate to specific handlers
	switch routerOutput.Intent {
	case router.IntentCreateTask:
		return h.handleCreateTask(ctx, sc, msg)
	
	case router.IntentSearchTask:
		return h.handleSearch(ctx, sc, msg.Text, msg.Chat.ID)
	
	case router.IntentManageChecklist:
		// Route to agent for intelligent handling
		return h.handleAgentOrchestrator(ctx, sc, msg.Text, msg.Chat.ID)
	
	case router.IntentConversation:
		return h.handleAgentOrchestrator(ctx, sc, msg.Text, msg.Chat.ID)
	
	default:
		// Fallback to create task
		return h.handleCreateTask(ctx, sc, msg)
	}
}
```

### 📁 Update: `internal/task/delivery/telegram/new.go`

**Convention**: Factory function for dependency injection

```go
// new.go - Update New function signature
func New(
	l pkgLog.Logger,
	uc task.UseCase,
	bot *pkgTelegram.Bot,
	orchestrator *orchestrator.Orchestrator,
	automationUC automation.UseCase,
	checklistSvc checklist.Service,
	memosRepo repository.MemosRepository,
	router *router.SemanticRouter, // 🆕 Add this parameter
) Handler {
	return &handler{
		l:            l,
		uc:           uc,
		bot:          bot,
		orchestrator: orchestrator,
		automationUC: automationUC,
		checklistSvc: checklistSvc,
		memosRepo:    memosRepo,
		router:       router, // 🆕 Inject router
	}
}
```

### 📁 File: `cmd/api/Dockerfile` (Update for Timezone Support)

**🔧 PRO-TIP #1: Add tzdata for timezone support in Alpine**

**Vấn đề**: Trong `time_context.go`, khi gọi `time.LoadLocation(timezone)`, nếu backend chạy trong Docker với image `golang:1.21-alpine`, image alpine mặc định KHÔNG có data múi giờ. Hàm này sẽ trả về lỗi và fallback về UTC, khiến time context không chính xác.

**Giải pháp**: Đảm bảo trong Dockerfile có cài đặt gói `tzdata`.

Tìm dòng `RUN apk --no-cache add ca-certificates tzdata curl wget` (around line 15) và verify tzdata đã được cài:

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/api

FROM alpine:latest

# 🔧 CRITICAL: Install tzdata for timezone support (Asia/Ho_Chi_Minh)
# Without this, time.LoadLocation() will fail and fallback to UTC
# PRO-TIP #1: This fixes "Temporal Blindness" in Docker environments
RUN apk --no-cache add ca-certificates tzdata curl wget

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/main .

# Copy config directory
COPY --from=builder /app/config ./config

EXPOSE 8080

CMD ["./main"]
```

**Verification**:
```bash
# After building, verify tzdata is installed
docker run --rm atm-backend ls /usr/share/zoneinfo/Asia/Ho_Chi_Minh
# Should output: /usr/share/zoneinfo/Asia/Ho_Chi_Minh
```

---

### 📁 Update: `cmd/api/main.go` (Dependency Injection)

```go
// After initializing geminiClient (around line 80)...

// 🆕 Initialize Semantic Router
semanticRouter := router.New(geminiClient, logger)
logger.Info(ctx, "Semantic Router initialized")

// Update telegram handler initialization (around line 150)
telegramHandler := telegram.NewHandler(
	logger,
	taskUC,
	telegramBot,
	orchestrator,
	automationUC,
	checklistSvc,
	memosRepo,
	semanticRouter, // 🆕 Add this parameter
)
```

---

## 2. HARD TIME INJECTION (Temporal Context)

### 📋 Checklist Implementation

- [ ] Tạo utility functions cho time context
- [ ] Update ProcessQuery để inject time context
- [ ] Expose GetSession method cho router
- [ ] Test với các query về "tuần này", "ngày mai"
- [ ] Verify agent không hỏi lại ngày tháng

### 🔍 Industry Standard

Pattern này trong ngành gọi là **Context Hydration** hoặc **Prompt Enrichment**. Các hệ thống production không bao giờ hy vọng LLM "tự biết" ngày giờ. Họ luôn dùng kỹ thuật **Hidden Context Prepending/Appending**: Lấy thời gian thực ở backend, format lại, và lén dán (append) vào ngay phía sau tin nhắn của user trước khi đưa cho LLM.

### 📁 File: `internal/agent/orchestrator/time_context.go` (🆕 New File)

```go
package orchestrator

import (
	"fmt"
	"time"
)

// buildTimeContext creates a temporal context string for LLM
func buildTimeContext(timezone string) string {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}
	
	now := time.Now().In(loc)
	
	// Calculate week boundaries (Monday-Sunday)
	weekday := int(now.Weekday())
	if weekday == 0 { // Sunday
		weekday = 7
	}
	weekStart := now.AddDate(0, 0, -(weekday - 1)) // Monday
	weekEnd := weekStart.AddDate(0, 0, 6)          // Sunday
	tomorrow := now.AddDate(0, 0, 1)
	
	// Build context string
	context := fmt.Sprintf(`

[SYSTEM CONTEXT - Thông tin thời gian hiện tại]
- Hôm nay: %s (%s)
- Tuần này: từ %s đến %s
- Ngày mai: %s

QUY TẮC QUAN TRỌNG:
1. Nếu user hỏi về "tuần này", hãy TỰ ĐỘNG sử dụng start_date='%s' và end_date='%s'
2. Nếu user hỏi về "ngày mai", dùng date='%s'
3. KHÔNG BAO GIỜ hỏi ngược lại user về ngày tháng cụ thể
4. Format ngày LUÔN LUÔN là YYYY-MM-DD
5. Tự động nội suy các mốc thời gian tương đối`,
		now.Format("2006-01-02"),
		now.Weekday().String(),
		weekStart.Format("2006-01-02"),
		weekEnd.Format("2006-01-02"),
		tomorrow.Format("2006-01-02"),
		weekStart.Format("2006-01-02"),
		weekEnd.Format("2006-01-02"),
		tomorrow.Format("2006-01-02"),
	)
	
	return context
}
```

### 📁 Update: `internal/agent/orchestrator/orchestrator.go`

**Thay đổi 1: Inject time context in ProcessQuery**

Tìm dòng này (around line 69):
```go
func (o *Orchestrator) ProcessQuery(ctx context.Context, userID string, query string) (string, error) {
```

Thay đổi thành:
```go
func (o *Orchestrator) ProcessQuery(ctx context.Context, userID string, query string) (string, error) {
	// 🆕 Inject time context into query
	timeContext := buildTimeContext(o.timezone)
	enhancedQuery := query + timeContext
	
	// Get session
	session := o.getSession(userID)
	
	// Add user message with enhanced query
	session.Messages = append(session.Messages, gemini.Content{
		Role: "user",
		Parts: []gemini.Part{
			{Text: enhancedQuery},
		},
	})
	session.LastUpdated = time.Now()
	
	// Rest of the ReAct loop remains the same...
	// (existing code continues from line ~100)
```

**Thay đổi 2: Expose GetSession method**

Thêm method này vào cuối file (sau ClearSession):

```go
// 🆕 GetSession exposes session for router to access conversation history
func (o *Orchestrator) GetSession(userID string) *SessionMemory {
	return o.getSession(userID)
}
```

---

## 3. TELEGRAM MARKDOWN SANITIZER

### 📋 Checklist Implementation

- [ ] Add HTML mode support to bot.SendMessage
- [ ] Add fallback to plain text if HTML fails
- [ ] Update all SendMessage calls (already done via default behavior)
- [ ] Test with special characters

### 🔍 Industry Standard

Telegram `MarkdownV2` là một "ác mộng" parsing vì nó yêu cầu escape (thêm `\`) cho 18 ký tự đặc biệt. Các framework bot lớn (như `python-telegram-bot` hoặc `telegraf`) thường khuyến nghị: **Chuyển từ `MarkdownV2` sang `HTML`**. Telegram HTML parser "hiền" hơn rất nhiều và LLM sinh ra text bọc trong thẻ `<b>`, `<i>`, `<code>` ít khi bị lỗi.

### 📁 Update: `pkg/telegram/bot.go`

**Thay đổi 1: Add SendMessageHTML method**

Thêm vào sau method SendMessageWithMode (around line 80):

```go
// 🆕 SendMessageHTML sends message with HTML formatting (safer than MarkdownV2)
func (b *Bot) SendMessageHTML(chatID int64, text string) error {
	url := fmt.Sprintf("%s/sendMessage", b.apiURL)
	
	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "HTML",
	}
	
	body, _ := json.Marshal(payload)
	resp, err := b.httpClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()
	
	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	
	if !apiResp.OK {
		// Fallback to plain text if HTML parsing fails
		return b.SendMessagePlain(chatID, text)
	}
	
	return nil
}

// 🆕 SendMessagePlain sends message without any formatting
func (b *Bot) SendMessagePlain(chatID int64, text string) error {
	url := fmt.Sprintf("%s/sendMessage", b.apiURL)
	
	payload := map[string]interface{}{
		"chat_id": chatID,
		"text":    text,
		// No parse_mode = plain text
	}
	
	body, _ := json.Marshal(payload)
	resp, err := b.httpClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()
	
	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	
	if !apiResp.OK {
		return fmt.Errorf("telegram API error: %s", apiResp.Description)
	}
	
	return nil
}
```

**Thay đổi 2: Update default SendMessage to use HTML**

Tìm method SendMessage (around line 57) và thay đổi:

```go
// SendMessage sends a text message to the specified chat (uses HTML mode by default)
func (b *Bot) SendMessage(chatID int64, text string) error {
	return b.SendMessageHTML(chatID, text) // 🆕 Changed from plain to HTML
}
```

---

## 4. SELF-HEALING RAG

### ✅ ĐÃ ĐƯỢC IMPLEMENT (HOTFIX 4)

Self-healing RAG đã được implement trong `internal/task/usecase/search.go` (lines 56-77). Chỉ cần áp dụng pattern tương tự cho `answer_query.go`.

### 📋 Checklist

- [x] Self-healing logic implemented in search.go
- [ ] Add same logic to answer_query.go (RAG)
- [ ] Test with deleted tasks

### 📁 Update: `internal/task/usecase/answer_query.go`

Tìm phần fetch source tasks (around line 50-70) và thêm self-healing logic:

```go
// Fetch full task details from Memos
sourceTasks := make([]repository.SearchResult, 0)
zombieVectors := make([]string, 0) // 🆕 Track zombie vectors

for _, sr := range searchResults {
	memoTask, err := uc.repo.GetTask(ctx, sr.MemoID)
	if err != nil {
		// 🆕 Self-healing: cleanup zombie vectors
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "Not Found") {
			uc.l.Warnf(ctx, "AnswerQuery: Task %s deleted in Memos. Self-healing: removing from Qdrant", sr.MemoID)
			zombieVectors = append(zombieVectors, sr.MemoID)
			
			// Async cleanup (don't block RAG)
			go func(memoID string) {
				cleanupCtx := context.Background()
				if err := uc.vectorRepo.DeleteTask(cleanupCtx, memoID); err != nil {
					uc.l.Errorf(cleanupCtx, "Self-healing: Failed to cleanup zombie vector %s: %v", memoID, err)
				} else {
					uc.l.Infof(cleanupCtx, "Self-healing: Successfully cleaned up zombie vector %s", memoID)
				}
			}(sr.MemoID)
			
			continue
		}
		
		uc.l.Warnf(ctx, "AnswerQuery: failed to fetch task %s: %v", sr.MemoID, err)
		continue
	}
	
	sourceTasks = append(sourceTasks, sr)
}

// 🆕 Log self-healing stats
if len(zombieVectors) > 0 {
	uc.l.Infof(ctx, "AnswerQuery: Self-healing cleaned up %d zombie vectors", len(zombieVectors))
}
```

---

## 5. SESSION MEMORY INTEGRATION

### ✅ COMPLETED

Session memory integration đã được hoàn thành trong Section 1 và 2:
- GetSession method exposed trong orchestrator.go
- Router sử dụng conversation history trong handler.go

---

## TESTING STRATEGY

### Unit Tests

#### 📁 File: `internal/router/router_test.go` (🆕 New File)

```go
package router

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSemanticRouter_Classify(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		history  []string
		expected Intent
	}{
		{
			name:     "Create task - explicit",
			message:  "Nhắc tôi họp lúc 3pm",
			history:  []string{},
			expected: IntentCreateTask,
		},
		{
			name:     "Create task - deadline",
			message:  "Deadline dự án ABC vào 15/3",
			history:  []string{},
			expected: IntentCreateTask,
		},
		{
			name:     "Search task",
			message:  "Tìm task về meeting",
			history:  []string{},
			expected: IntentSearchTask,
		},
		{
			name:     "Search task - alternative",
			message:  "Có task nào về dự án SMAP không?",
			history:  []string{},
			expected: IntentSearchTask,
		},
		{
			name:     "Conversation - greeting",
			message:  "Chào bạn",
			history:  []string{},
			expected: IntentConversation,
		},
		{
			name:     "Conversation - help",
			message:  "Bạn có thể giúp tôi những gì?",
			history:  []string{},
			expected: IntentConversation,
		},
		{
			name:     "Context-aware create",
			message:  "Đổi lại lúc 9h nhé",
			history:  []string{"User: Tạo task họp lúc 3pm"},
			expected: IntentCreateTask,
		},
		{
			name:     "Manage checklist",
			message:  "Đánh dấu hoàn thành task abc123",
			history:  []string{},
			expected: IntentManageChecklist,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This requires actual Gemini API call
			// For true unit test, mock the LLM client
			// For now, this serves as integration test
			t.Skip("Requires Gemini API - run manually")
		})
	}
}
```

#### 📁 File: `internal/agent/orchestrator/time_context_test.go` (🆕 New File)

```go
package orchestrator

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBuildTimeContext(t *testing.T) {
	timezone := "Asia/Ho_Chi_Minh"
	context := buildTimeContext(timezone)
	
	// Verify context contains key elements
	assert.Contains(t, context, "SYSTEM CONTEXT")
	assert.Contains(t, context, "Hôm nay:")
	assert.Contains(t, context, "Tuần này:")
	assert.Contains(t, context, "Ngày mai:")
	assert.Contains(t, context, "YYYY-MM-DD")
	
	// Verify date format
	now := time.Now()
	todayStr := now.Format("2006-01-02")
	assert.Contains(t, context, todayStr)
}

func TestBuildTimeContext_WeekBoundaries(t *testing.T) {
	context := buildTimeContext("Asia/Ho_Chi_Minh")
	
	// Should contain Monday and Sunday dates
	lines := strings.Split(context, "\n")
	var weekLine string
	for _, line := range lines {
		if strings.Contains(line, "Tuần này:") {
			weekLine = line
			break
		}
	}
	
	assert.NotEmpty(t, weekLine)
	assert.Contains(t, weekLine, "từ")
	assert.Contains(t, weekLine, "đến")
}
```

### Integration Tests (Manual)

Sử dụng các Milestones từ Master Plan:

#### 🏆 Milestone 1: "Smooth Talker" (Giao tiếp không rào cản)

```bash
# Test Case 1: Greeting
Input: "Chào bạn, bạn có thể giúp tôi những gì?"
Expected: 
  - Router classifies as CONVERSATION
  - Bot responds with friendly message
  - No "no tasks parsed" error
  - Log shows: "router: Classified as CONVERSATION"

# Test Case 2: Natural create
Input: "Nhắc tôi họp team lúc 3pm ngày mai"
Expected:
  - Router classifies as CREATE_TASK
  - Task created successfully
  - Log shows: "router: Classified as CREATE_TASK"
```

#### 🏆 Milestone 2: "Time Master" (Bậc thầy thời gian)

```bash
# Test Case 1: Week query
Input: "Kiểm tra lịch tuần này xem có vướng gì không?"
Expected:
  - Agent does NOT ask for dates
  - Automatically calculates Monday-Sunday
  - Calls check_calendar with correct dates
  - Log shows: "SYSTEM CONTEXT" with week boundaries

# Test Case 2: Tomorrow query
Input: "Tôi có meeting nào ngày mai?"
Expected:
  - Agent uses tomorrow's date automatically
  - No date clarification questions
  - Log shows tomorrow's date in YYYY-MM-DD format
```

#### 🏆 Milestone 3: "Self-Healing RAG" (Không còn bóng ma)

```bash
# Test Case 1: Zombie vector cleanup
Steps:
  1. Create task: "Mua sữa lúc 5h chiều"
  2. Note the memo ID (e.g., abc123)
  3. Delete task in Memos web UI
  4. Search: "Tìm task về việc mua sữa"

Expected:
  - Bot responds: "Không tìm thấy task"
  - Log shows: "Self-healing: Successfully cleaned up zombie vector abc123"
  - Verify in Qdrant: vector abc123 is deleted

# Test Case 2: Multiple zombie cleanup
Steps:
  1. Create 3 tasks
  2. Delete all 3 in Memos
  3. Search for them

Expected:
  - All 3 vectors cleaned up
  - Log shows: "Self-healing cleaned up 3 zombie vectors"
```

#### 🏆 Milestone 4: "Bulletproof Messaging" (Chống đạn API)

```bash
# Test Case 1: Special characters
Input: "Tạo task: Code hàm func()_test[]!"
Expected:
  - Message sent successfully
  - Special chars display correctly
  - No "400 Bad Request" error
  - Log shows: "SendMessageHTML" (not MarkdownV2)

# Test Case 2: LLM with code blocks
Input: "/ask Giải thích code này: `const x = [1, 2, 3]`"
Expected:
  - Bot responds with explanation
  - Code block renders correctly
  - No parsing errors
```

---

## DEPLOYMENT CHECKLIST

### Pre-deployment

- [ ] All unit tests pass: `make test`
- [ ] Integration tests pass (4 milestones)
- [ ] Code review completed
- [ ] Documentation updated
- [ ] Backup current system

### Deployment Steps

```bash
# 1. Backup current system
make backup

# 2. Pull latest code
git pull origin main

# 3. Build new binary
make build

# 4. Restart services
make restart

# 5. Verify services
make logs

# 6. Check router initialization
grep "Semantic Router initialized" logs/app.log

# 7. Check time context injection
grep "SYSTEM CONTEXT" logs/app.log

# 8. Monitor for errors
tail -f logs/app.log | grep -i error
```

### Smoke Tests

```bash
# Test 1: Router working
curl -X POST http://localhost:8080/webhook/telegram \
  -H "Content-Type: application/json" \
  -d '{"message": {"text": "Chào bạn", "from": {"id": 123}, "chat": {"id": 123}}}'

# Test 2: Time context
# Send message via Telegram: "Kiểm tra lịch tuần này"
# Check logs for SYSTEM CONTEXT

# Test 3: HTML mode
# Send message with special chars: "Test: func()_test[]"
# Verify no 400 errors
```

### Rollback Plan

If issues occur:

```bash
# 1. Checkout previous version
git log --oneline -10  # Find last working commit
git checkout <commit-hash>

# 2. Rebuild
make build

# 3. Restart
make restart

# 4. Verify
make logs
```

---

## PERFORMANCE CONSIDERATIONS

### Latency Impact

| Component | Added Latency | Mitigation |
|-----------|---------------|------------|
| Router call | +200-500ms | Use Gemini Flash (fastest model) |
| Time context | <1ms | Negligible |
| Self-healing | 0ms (async) | Runs in background goroutine |
| HTML parsing | 0ms | Same as MarkdownV2 |

### Optimization Tips

1. **Cache router results**
   ```go
   // Cache identical messages for 5 minutes
   type RouterCache struct {
       cache map[string]RouterOutput
       ttl   time.Duration
   }
   ```

2. **Batch cleanup**
   ```go
   // Collect zombie vectors and delete in batches
   if len(zombieVectors) > 10 {
       go batchDeleteVectors(zombieVectors)
   }
   ```

3. **Monitor Gemini quota**
   ```bash
   # Track API usage
   grep "router: LLM call" logs/app.log | wc -l
   ```

---

## MONITORING & METRICS

### Key Metrics to Track

```go
// Add to internal/router/router.go
var (
	routerCallsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atm_router_calls_total",
			Help: "Total number of router classifications",
		},
		[]string{"intent"},
	)
	
	routerLatency = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "atm_router_latency_seconds",
			Help:    "Router classification latency",
			Buckets: []float64{0.1, 0.2, 0.5, 1.0, 2.0},
		},
	)
	
	selfHealingCleanups = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "atm_self_healing_cleanups_total",
			Help: "Total number of zombie vectors cleaned up",
		},
	)
)
```

### Log Patterns to Monitor

```bash
# Router classifications
grep "router: Classified as" logs/app.log | tail -20

# Self-healing cleanups
grep "Self-healing: Successfully cleaned up" logs/app.log | wc -l

# Time context injections
grep "SYSTEM CONTEXT" logs/app.log | head -5

# Telegram API errors
grep "400 Bad Request" logs/app.log

# Router errors
grep "router: Classification failed" logs/app.log
```

### Dashboard Queries (Prometheus)

```promql
# Router classification rate
rate(atm_router_calls_total[5m])

# Router latency p95
histogram_quantile(0.95, rate(atm_router_latency_seconds_bucket[5m]))

# Self-healing rate
rate(atm_self_healing_cleanups_total[1h])
```

---

## TROUBLESHOOTING

### Router not working

**Symptom**: All messages treated as CREATE_TASK

**Debug steps**:
```bash
# 1. Check Gemini API key
echo $GEMINI_API_KEY

# 2. Check router initialization
grep "Semantic Router initialized" logs/app.log

# 3. Check router logs
grep "router:" logs/app.log | tail -20

# 4. Test router directly (if debug endpoint exists)
curl -X POST http://localhost:8080/debug/router \
  -d '{"message": "Chào bạn"}'
```

**Common fixes**:
- Verify Gemini API key is set
- Check Gemini API quota
- Verify router is injected in handler

### Time context not injected

**Symptom**: Agent still asks for dates

**Debug steps**:
```bash
# 1. Check timezone config
grep "timezone" config/config.yaml

# 2. Verify time context in logs
grep "SYSTEM CONTEXT" logs/app.log | head -1

# 3. Check orchestrator logs
grep "ProcessQuery" logs/app.log | tail -10
```

**Common fixes**:
- Verify timezone is set in config
- Check buildTimeContext is called
- Verify enhancedQuery is used

### Self-healing not triggering

**Symptom**: Zombie vectors not cleaned up

**Debug steps**:
```bash
# 1. Check Qdrant connection
curl http://localhost:6333/collections/tasks

# 2. Verify 404 detection
grep "404" logs/app.log | grep "Self-healing"

# 3. Check cleanup logs
grep "Self-healing: Successfully cleaned up" logs/app.log
```

**Common fixes**:
- Verify Qdrant is running
- Check error message contains "404" or "Not Found"
- Verify vectorRepo.DeleteTask is called

### Telegram 400 errors

**Symptom**: Messages fail to send

**Debug steps**:
```bash
# 1. Check parse mode
grep "SendMessage" logs/app.log | grep "parse_mode"

# 2. Check error details
grep "400 Bad Request" logs/app.log | tail -5

# 3. Test with plain text
# Temporarily disable HTML mode
```

**Common fixes**:
- Verify SendMessageHTML is used
- Check fallback to SendMessagePlain works
- Test with simple messages first

---

## NEXT STEPS (Post v1.1)

### Short-term (v1.2)

1. **Router improvements**
   - Add confidence threshold (skip routing if < 70%)
   - Implement caching for identical messages
   - A/B test different prompts

2. **Enhanced metrics**
   - Add Prometheus metrics
   - Create Grafana dashboard
   - Set up alerts for high error rates

3. **Testing automation**
   - Add E2E tests with Playwright
   - Automate milestone tests
   - CI/CD integration

### Mid-term (v1.3)

1. **Advanced time handling**
   - Support more relative dates ("next month", "in 2 weeks")
   - Multi-timezone support for teams
   - Recurring tasks

2. **Enhanced self-healing**
   - Periodic full sync job (nightly)
   - Metrics dashboard for drift rate
   - Alert on high drift (> 5%)

3. **Telegram UX**
   - Inline keyboards for quick actions
   - Rich formatting with HTML
   - Voice message support

### Long-term (v2.0)

1. **Multi-user support**
   - Shared workspaces
   - Task assignment
   - Collaboration features

2. **Advanced AI**
   - Multi-agent collaboration
   - Proactive suggestions
   - Learning from user patterns

3. **Platform expansion**
   - Web UI
   - Mobile app
   - Slack integration

---

## APPENDIX

### A. File Checklist

**New Files**:
- [ ] `internal/router/types.go`
- [ ] `internal/router/router.go`
- [ ] `internal/router/new.go`
- [ ] `internal/router/router_test.go`
- [ ] `internal/agent/orchestrator/time_context.go`
- [ ] `internal/agent/orchestrator/time_context_test.go`

**Modified Files**:
- [ ] `internal/task/delivery/telegram/handler.go`
- [ ] `internal/task/delivery/telegram/new.go`
- [ ] `internal/agent/orchestrator/orchestrator.go`
- [ ] `internal/task/usecase/answer_query.go`
- [ ] `pkg/telegram/bot.go`
- [ ] `cmd/api/main.go`

### B. Dependencies

No new external dependencies required. All changes use existing packages:
- `autonomous-task-management/pkg/gemini`
- `autonomous-task-management/pkg/log`
- Standard library: `time`, `fmt`, `strings`, `encoding/json`

### C. Configuration Changes

No configuration file changes required. All features work with existing config.

Optional: Add router-specific config in future versions:
```yaml
router:
  enabled: true
  confidence_threshold: 70
  cache_ttl: 5m
```

---

## LEGACY CODE MANAGEMENT

### Philosophy: Graceful Deprecation

Version 1.1 theo triết lý **"Add, Don't Remove"** - Thêm tính năng mới mà không phá vỡ workflow cũ. Điều này đảm bảo:
- Zero downtime deployment
- User có thời gian làm quen với UX mới
- Rollback dễ dàng nếu có vấn đề
- A/B testing giữa old và new behavior

### What Becomes Legacy

#### 1. Slash Commands (Partial Legacy)

**Status**: DEPRECATED but SUPPORTED

**Current behavior** (v1.0):
```go
// Hard-coded routing in handler.go
case strings.HasPrefix(msg.Text, "/ask "):
    return h.handleAgentOrchestrator(...)
case strings.HasPrefix(msg.Text, "/search "):
    return h.handleSearch(...)
```

**New behavior** (v1.1):
```go
// Slash commands still work (backward compatibility)
// But natural language also works via router
switch {
case strings.HasPrefix(msg.Text, "/ask "):
    // Legacy path - still supported
    return h.handleAgentOrchestrator(...)
default:
    // New path - semantic routing
    routerOutput := h.router.Classify(...)
}
```

**Deprecation timeline**:
- **v1.1**: Both slash commands and natural language work
- **v1.2**: Add deprecation warning in `/help`
- **v1.3**: Log usage metrics to decide removal
- **v2.0**: Consider removing if usage < 5%

**Migration guide for users**:
```
Old way:  /ask Tôi có deadline nào tuần này?
New way:  Tôi có deadline nào tuần này?  (no slash needed)

Old way:  /search meeting
New way:  Tìm task về meeting  (natural language)
```

#### 2. Direct Task Creation (No Change)

**Status**: NOT LEGACY - Still primary path

```go
// This remains the same
case router.IntentCreateTask:
    return h.handleCreateTask(ctx, sc, msg)
```

Task creation logic (`handleCreateTask`) không thay đổi, chỉ cách routing đến nó thay đổi.

#### 3. MarkdownV2 Mode (Deprecated)

**Status**: DEPRECATED and REPLACED

**Old code** (v1.0):
```go
// pkg/telegram/bot.go
func (b *Bot) SendMessage(chatID int64, text string) error {
    // Implicitly uses MarkdownV2 or plain text
}
```

**New code** (v1.1):
```go
// Default to HTML mode with fallback
func (b *Bot) SendMessage(chatID int64, text string) error {
    return b.SendMessageHTML(chatID, text)
}

// Legacy method still exists but not used
func (b *Bot) SendMessageWithMode(chatID int64, text string, parseMode string) error {
    // Keep for backward compatibility
}
```

**Removal timeline**:
- **v1.1**: HTML mode becomes default
- **v1.2**: Remove SendMessageWithMode if no external usage
- **v2.0**: Clean up completely

#### 4. Manual Time Context (Removed)

**Status**: REMOVED - Replaced by automatic injection

**Old code** (Phase 5 HOTFIX 2):
```go
// orchestrator.go - Lines 69-100
// Manual time context in SystemInstruction
timeContext := fmt.Sprintf(
    "\n\n[SYSTEM CONTEXT - Thông tin thời gian hiện tại:"+
    "\n- Hôm nay: %s (%s)"+
    // ... rest of context
)
```

**New code** (v1.1):
```go
// time_context.go - Extracted to separate file
func buildTimeContext(timezone string) string {
    // Same logic but cleaner
}

// orchestrator.go - Injected into query
enhancedQuery := query + buildTimeContext(o.timezone)
```

**Why removed**:
- Old approach: Time context in SystemInstruction (LLM often ignores)
- New approach: Time context appended to user query (LLM always sees)
- No backward compatibility needed - internal implementation detail

### Code Cleanup Checklist

#### Phase 1: v1.1 Release (Current)

**Keep everything, add new features**:
- [x] Keep all slash command handlers
- [x] Keep SendMessageWithMode method
- [x] Add new router alongside old routing
- [x] Add HTML mode alongside old modes

**Mark as deprecated** (in code comments):
```go
// Deprecated: Use natural language instead of /ask command
// This will be removed in v2.0
case strings.HasPrefix(msg.Text, "/ask "):
```

#### Phase 2: v1.2 (3 months later)

**Collect metrics**:
```go
// Add metrics to track usage
var (
    slashCommandUsage = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "atm_slash_command_usage_total",
        },
        []string{"command"}, // /ask, /search, etc.
    )
    
    naturalLanguageUsage = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "atm_natural_language_usage_total",
        },
    )
)
```

**Add deprecation warnings**:
```go
func (h *handler) handleHelp(ctx context.Context, chatID int64) error {
    helpText := `
🤖 Autonomous Task Management Bot

Cách sử dụng:
✨ Mới: Chat tự nhiên (khuyến nghị)
  "Tạo task họp lúc 3pm"
  "Tìm task về meeting"
  "Tôi có deadline nào tuần này?"

⚠️ Cũ: Slash commands (sẽ bị loại bỏ trong v2.0)
  /ask <câu hỏi>
  /search <từ khóa>
  /progress <taskID>
`
    return h.bot.SendMessage(chatID, helpText)
}
```

#### Phase 3: v1.3 (6 months later)

**Analyze metrics and decide**:
```bash
# Query Prometheus
sum(rate(atm_slash_command_usage_total[30d])) / 
sum(rate(atm_natural_language_usage_total[30d]))

# If ratio < 0.05 (5%), proceed with removal
```

**Add final warning**:
```go
case strings.HasPrefix(msg.Text, "/ask "):
    // Send deprecation notice
    h.bot.SendMessage(msg.Chat.ID, 
        "⚠️ Slash commands sẽ bị loại bỏ trong v2.0. "+
        "Hãy chat tự nhiên thay vì dùng /ask")
    
    // Still process the command
    return h.handleAgentOrchestrator(...)
```

#### Phase 4: v2.0 (12 months later)

**Remove legacy code**:

```go
// REMOVE these cases from processMessage:
// case strings.HasPrefix(msg.Text, "/ask "):
// case strings.HasPrefix(msg.Text, "/search "):

// KEEP essential commands:
case msg.Text == "/start":
case msg.Text == "/help":
case msg.Text == "/reset":
case strings.HasPrefix(msg.Text, "/progress "):
case strings.HasPrefix(msg.Text, "/complete "):
case strings.HasPrefix(msg.Text, "/check "):
case strings.HasPrefix(msg.Text, "/uncheck "):
```

**Remove unused methods**:
```go
// pkg/telegram/bot.go
// REMOVE: SendMessageWithMode (if not used externally)
// KEEP: SendMessage, SendMessageHTML, SendMessagePlain
```

### Migration Script (Optional)

For users who have bookmarks or scripts using old commands:

```bash
#!/bin/bash
# scripts/migrate-commands.sh

echo "🔄 Migrating from slash commands to natural language..."

# Show examples
cat << EOF
Old Command              → New Natural Language
─────────────────────────────────────────────────
/ask Deadline tuần này?  → Deadline tuần này?
/search meeting          → Tìm task về meeting
/ask Lịch ngày mai       → Lịch ngày mai như thế nào?

✅ Commands that stay the same:
/start, /help, /reset, /progress, /complete, /check, /uncheck
EOF
```

### Documentation Updates

#### Update README.md

```markdown
## Cách sử dụng

### ✨ Chat tự nhiên (Khuyến nghị - v1.1+)

Chỉ cần chat bình thường, AI sẽ tự hiểu:

\`\`\`
"Deadline dự án SMAP vào 15/3"
"Tìm task về meeting"
"Tôi có deadline nào tuần này?"
\`\`\`

### 📝 Slash Commands (Legacy - Sẽ bị loại bỏ trong v2.0)

⚠️ **Deprecated**: Các lệnh này vẫn hoạt động nhưng sẽ bị loại bỏ trong tương lai.

\`\`\`bash
/ask <câu hỏi>    # Thay bằng: Chat tự nhiên
/search <từ khóa> # Thay bằng: "Tìm task về <từ khóa>"
\`\`\`

### 🔧 Utility Commands (Vẫn giữ nguyên)

\`\`\`bash
/start      # Bắt đầu
/help       # Trợ giúp
/reset      # Xóa lịch sử hội thoại
/progress   # Xem tiến độ
/complete   # Đánh dấu hoàn thành
/check      # Check item
/uncheck    # Uncheck item
\`\`\`
```

### Rollback Strategy

If v1.1 causes issues, rollback is simple because old code paths still exist:

```go
// Emergency rollback: Disable router
const ROUTER_ENABLED = false // Set to false to disable

func (h *handler) processMessage(...) {
    // ... slash command handling ...
    
    if !ROUTER_ENABLED {
        // Fallback to old behavior
        return h.handleCreateTask(ctx, sc, msg)
    }
    
    // New router logic
    routerOutput := h.router.Classify(...)
}
```

### Testing Legacy Paths

```go
// internal/task/delivery/telegram/handler_test.go

func TestBackwardCompatibility(t *testing.T) {
    tests := []struct {
        name    string
        message string
        handler string
    }{
        {
            name:    "Slash ask still works",
            message: "/ask Deadline tuần này?",
            handler: "handleAgentOrchestrator",
        },
        {
            name:    "Slash search still works",
            message: "/search meeting",
            handler: "handleSearch",
        },
        {
            name:    "Natural language works",
            message: "Tìm task về meeting",
            handler: "handleSearch (via router)",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test that both old and new paths work
        })
    }
}
```

---

## PRO-TIPS: Critical Fixes from Architecture Review

Dựa trên phân tích chi tiết từ System Architect, hai điểm rủi ro sau đã được tích hợp vào Code Plan:

### 🔧 PRO-TIP #1: Rủi ro múi giờ (Timezone) trong Docker

**Vấn đề**: 
Trong `time_context.go`, khi gọi `time.LoadLocation(timezone)`, nếu backend chạy trong Docker với image `golang:1.21-alpine`, image alpine mặc định KHÔNG có data múi giờ. Hàm này sẽ trả về lỗi và fallback về UTC, khiến Agent vẫn bị "mù thời gian".

**Nguyên nhân**:
Alpine Linux là distro tối giản, không bao gồm timezone database (`tzdata`) mặc định để giảm kích thước image.

**Giải pháp**:
Đảm bảo trong `cmd/api/Dockerfile` có lệnh cài đặt gói `tzdata`:

```dockerfile
# 🔧 CRITICAL: Install tzdata for timezone support (Asia/Ho_Chi_Minh)
# Without this, time.LoadLocation() will fail and fallback to UTC
RUN apk --no-cache add ca-certificates tzdata curl wget
```

**Verification**:
```bash
docker run --rm atm-backend ls /usr/share/zoneinfo/Asia/Ho_Chi_Minh
# Should output: /usr/share/zoneinfo/Asia/Ho_Chi_Minh
```

**Tham khảo**: Section "1. OMNI-ROUTER" → File `cmd/api/Dockerfile`

---

### 🔧 PRO-TIP #2: Rủi ro Race Condition trong Router Fallback

**Vấn đề**:
Trong `router.go`, nếu parse JSON lỗi (do Gemini trả về format không đúng), việc fallback về `IntentCreateTask` sẽ gây ra lỗi `no tasks parsed from input` khi user đang hỏi một câu bâng quơ (VD: "Bạn có thể giúp tôi những gì?").

**Nguyên nhân**:
- JSON parsing fail → Fallback `IntentCreateTask`
- Message không phải task → `handleCreateTask` parse fail
- User nhận lỗi kỹ thuật thay vì câu trả lời thân thiện

**Giải pháp**:
Đổi luồng fallback an toàn mặc định thành `IntentConversation`:

```go
if err := json.Unmarshal([]byte(resp.Text), &output); err != nil {
    // 🔧 PRO-TIP #2: Fallback to CONVERSATION (safer than CREATE_TASK)
    return RouterOutput{
        Intent:     IntentConversation,  // NOT IntentCreateTask
        Confidence: 50,
        Reasoning:  "Fallback due to parsing error - route to conversational agent",
    }, nil
}
```

**Lý do**:
Nếu bot không hiểu, thà để nó trả lời "Tôi chưa hiểu ý bạn, bạn nói rõ hơn được không?" (Conversation) còn hơn là văng lỗi kỹ thuật.

**Tham khảo**: Section "1. OMNI-ROUTER" → File `internal/router/router.go` (line ~60)

---

## SUMMARY: Legacy Management Strategy

| Component | v1.0 | v1.1 | v1.2 | v1.3 | v2.0 |
|-----------|------|------|------|------|------|
| Slash commands (/ask, /search) | ✅ Primary | ✅ Supported | ⚠️ Deprecated | ⚠️ Warning | ❌ Removed |
| Natural language | ❌ None | ✅ Primary | ✅ Primary | ✅ Primary | ✅ Only way |
| MarkdownV2 mode | ✅ Default | ⚠️ Fallback | ⚠️ Fallback | ❌ Removed | ❌ Removed |
| HTML mode | ❌ None | ✅ Default | ✅ Default | ✅ Default | ✅ Default |
| Manual time context | ✅ Used | ❌ Removed | ❌ Removed | ❌ Removed | ❌ Removed |
| Auto time injection | ❌ None | ✅ Used | ✅ Used | ✅ Used | ✅ Used |

**Key Principles**:
1. **Add, don't remove** (v1.1)
2. **Deprecate with warnings** (v1.2-v1.3)
3. **Remove after metrics confirm** (v2.0)
4. **Always keep rollback path** (all versions)

---

**Document Version:** 1.1  
**Last Updated:** 2026-02-27  
**Author:** AI Assistant  
**Status:** Ready for Implementation  
**Estimated Effort:** 2-3 days (1 developer)

