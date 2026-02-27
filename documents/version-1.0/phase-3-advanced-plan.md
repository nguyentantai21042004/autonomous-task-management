## PHASE 3 ADVANCED: AGENTIC INTELLIGENCE & RAG OPTIMIZATION

### ✅ Phase 3 Basic Prerequisites

**Must be completed before starting Advanced:**

- ✅ Qdrant client working (Task 3.1)
- ✅ Embedding service operational (Task 3.2)
- ✅ Vector repository with UUID conversion (Task 3.3)
- ✅ Auto-embedding after task creation (Task 3.4)
- ✅ Semantic search working (Task 3.5)
- ✅ Telegram `/search` command (Task 3.6)
- ✅ Agent tools framework foundation (Task 3.7)

**Verified functionality:**

- Tasks auto-embedded to Qdrant
- Search returns relevant results
- No false positive intent detection
- Graceful degradation on failures

---

## 🎯 Mục tiêu Phase 3 Advanced

Nâng cấp từ "reactive search" sang "intelligent agent":

1. **ReAct Agent Loop:** LLM tự quyết định gọi tools (Reason → Act → Observe)
2. **RAG with Context Truncation:** Prevent token overflow, maintain quality
3. **Webhook Sync with Retry:** Auto-update Qdrant khi Memos thay đổi
4. **Multi-mode Interface:** `/search` (fast), `/ask` (intelligent)

**Key Improvements:**

- Agent có khả năng multi-step reasoning
- RAG không bị tràn token
- Qdrant luôn sync với Memos (eventual consistency)
- User có lựa chọn giữa speed vs intelligence

---

## 🚨 Critical Risks & Solutions

### 🔥 CRITICAL BUGS FIXED (Expert Review)

**Bug 1: Unicode Slicing Bug (Vietnamese Text Corruption)**

**Problem:** `text[:maxLen]` operates on bytes, not characters. Vietnamese UTF-8 chars (2-3 bytes) can be split mid-byte → corrupted output → LLM hallucination.

**Example:**
```go
// ❌ WRONG: Byte-based slicing
text := "Họp với đội ngũ"
truncated := text[:10]  // May cut "đ" in half → "Họp với �"
```

**Solution:** Convert to rune array (Unicode characters)
```go
// ✅ CORRECT: Character-based slicing
func truncateText(text string, maxLen int) string {
	runes := []rune(text)  // Convert to Unicode characters
	if len(runes) <= maxLen {
		return text
	}
	return string(runes[:maxLen]) + "... [đã cắt bớt]"
}
```

---

**Bug 2: Goroutine Leak (Context Timeout Missing)**

**Problem:** `context.Background()` in webhook goroutine has no timeout. If Memos API hangs → goroutine stuck forever → memory leak.

**Impact:** 1000 webhook calls with hung requests = 1000 leaked goroutines = OOM crash

**Solution:** Add timeout to background context
```go
// ❌ WRONG: No timeout
go func(bgCtx context.Context, p MemosWebhookPayload) {
	h.syncWithRetry(bgCtx, memoID)  // Can hang forever
}(context.Background(), payload)

// ✅ CORRECT: 2-minute timeout
go func(p MemosWebhookPayload) {
	bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()  // Always cleanup
	
	h.syncWithRetry(bgCtx, memoID)
}(payload)
```

---

### Risk 1: Infinite Agent Loop ⚠️ CRITICAL

**Problem:** LLM có thể gọi tool liên tục không dừng.

**Example:**

```
Step 1: LLM calls search_tasks
Step 2: LLM calls search_tasks again (same query)
Step 3: LLM calls search_tasks again...
→ Infinite loop → API quota exhausted
```

**Solution:** Max steps limit (5 steps)

```go
const MaxAgentSteps = 5

for step := 0; step < MaxAgentSteps; step++ {
	// Process one step
	if resp.FunctionCall == nil {
		return resp.Text, nil  // Done
	}
	// Execute tool and continue
}

return "Agent exceeded max steps", nil
```

---

### Risk 2: Token Overflow in RAG ⚠️ HIGH

**Problem:** Nhét 10 tasks (mỗi task 2000 chars) vào prompt → 20k chars → Vượt token limit.

**Impact:**

- API error (context too long)
- High cost
- Diluted context quality

**Solution:** Truncate each task to 800 chars (Unicode-safe)

```go
func truncateText(text string, maxLen int) string {
	runes := []rune(text)  // ✅ Unicode-safe for Vietnamese
	if len(runes) <= maxLen {
		return text
	}
	return string(runes[:maxLen]) + "... [truncated]"
}

// Usage
safeContent := truncateText(memoTask.Content, 800)
```

**Math:**

- 5 tasks × 800 chars = 4000 chars
- + System prompt (500 chars)
- + User query (200 chars)
- = ~4700 chars (~1200 tokens)
- Safe for Gemini (32k token limit)

---

### Risk 3: Webhook Sync Failure ⚠️ MEDIUM

**Problem:** Network glitch → Embed fails → Qdrant out of sync.

**Impact:** Search returns stale data.

**Solution:** Exponential backoff retry (3 attempts) + timeout protection

```go
// In handler: Create context with timeout
go func(p MemosWebhookPayload) {
	bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()  // ✅ Prevent goroutine leak
	
	h.syncWithRetry(bgCtx, memoID)
}(payload)

// In syncWithRetry: Exponential backoff
func (h *WebhookHandler) syncWithRetry(ctx context.Context, memoID string) {
	maxRetries := 3
	backoff := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		if err := h.vectorRepo.EmbedTask(ctx, task); err == nil {
			return  // Success
		}
		time.Sleep(backoff)
		backoff *= 2  // 2s → 4s → 8s
	}

	h.l.Errorf(ctx, "Sync failed after %d retries", maxRetries)
}
```

---

## Task Breakdown

### Task 3A.1: ReAct Agent Orchestrator

**Mục tiêu:** LLM tự quyết định gọi tools trong vòng lặp multi-step

**File:** `internal/agent/orchestrator/orchestrator.go`

```go
package orchestrator

import (
	"context"
	"fmt"

	"autonomous-task-management/internal/agent"
	"autonomous-task-management/pkg/gemini"
	pkgLog "autonomous-task-management/pkg/log"
)

const MaxAgentSteps = 5

type Orchestrator struct {
	llm      *gemini.Client
	registry *agent.ToolRegistry
	l        pkgLog.Logger
}

func New(llm *gemini.Client, registry *agent.ToolRegistry, l pkgLog.Logger) *Orchestrator {
	return &Orchestrator{
		llm:      llm,
		registry: registry,
		l:        l,
	}
}

// ProcessQuery runs ReAct loop: Reason → Act → Observe
func (o *Orchestrator) ProcessQuery(ctx context.Context, query string) (string, error) {
	req := gemini.GenerateRequest{
		Contents: []gemini.Content{
			{Role: "user", Parts: []gemini.Part{{Text: query}}},
		},
		Tools: o.registry.ToFunctionDefinitions(),
	}

	for step := 0; step < MaxAgentSteps; step++ {
		o.l.Infof(ctx, "Agent step %d/%d", step+1, MaxAgentSteps)

		// 1. Reason: Ask LLM what to do
		resp, err := o.llm.GenerateContent(ctx, req)
		if err != nil {
			return "", fmt.Errorf("agent LLM error at step %d: %w", step, err)
		}

		// 2. Check if LLM wants to call a tool
		if resp.FunctionCall == nil {
			// LLM has final answer
			o.l.Infof(ctx, "Agent finished at step %d", step+1)
			return resp.Text, nil
		}

		// 3. Act: Execute the tool
		toolName := resp.FunctionCall.Name
		o.l.Infof(ctx, "Agent calling tool: %s with args: %+v", toolName, resp.FunctionCall.Args)

		tool, ok := o.registry.Get(toolName)
		var toolResult interface{}

		if !ok {
			o.l.Errorf(ctx, "Tool %s not found", toolName)
			toolResult = map[string]string{"error": "tool not found"}
		} else {
			// Execute tool
			res, err := tool.Execute(ctx, resp.FunctionCall.Args)
			if err != nil {
				o.l.Errorf(ctx, "Tool %s failed: %v", toolName, err)
				toolResult = map[string]string{"error": err.Error()}
			} else {
				toolResult = res
			}
		}

		// 4. Observe: Add tool result to conversation history
		req.Contents = append(req.Contents, gemini.Content{
			Role:  "model",
			Parts: []gemini.Part{{FunctionCall: resp.FunctionCall}},
		})
		req.Contents = append(req.Contents, gemini.Content{
			Role:  "function",
			Parts: []gemini.Part{{FunctionResponse: toolResult}},
		})
	}

	// Max steps exceeded
	o.l.Warnf(ctx, "Agent exceeded max steps (%d)", MaxAgentSteps)
	return "Trợ lý đã suy nghĩ quá lâu (vượt quá số bước cho phép). Vui lòng thử chia nhỏ câu hỏi.", nil
}
```

**File:** `internal/agent/orchestrator/new.go`

```go
package orchestrator

import (
	"autonomous-task-management/internal/agent"
	"autonomous-task-management/pkg/gemini"
	pkgLog "autonomous-task-management/pkg/log"
)

type Orchestrator struct {
	llm      *gemini.Client
	registry *agent.ToolRegistry
	l        pkgLog.Logger
}

func New(llm *gemini.Client, registry *agent.ToolRegistry, l pkgLog.Logger) *Orchestrator {
	return &Orchestrator{
		llm:      llm,
		registry: registry,
		l:        l,
	}
}
```

---

### Task 3A.2: RAG with Context Truncation

**Mục tiêu:** Prevent token overflow, maintain context quality

**File:** `internal/task/usecase/answer_query.go` (NEW)

```go
package usecase

import (
	"context"
	"fmt"
	"strings"

	"autonomous-task-management/internal/model"
	"autonomous-task-management/internal/task"
	"autonomous-task-management/internal/task/repository"
)

const (
	MaxTasksInContext = 5    // Top-5 most relevant tasks
	MaxCharsPerTask   = 800  // Truncate each task to 800 chars
)

// AnswerQuery uses RAG to answer questions about tasks.
func (uc *implUseCase) AnswerQuery(ctx context.Context, sc model.Scope, input task.QueryInput) (task.QueryOutput, error) {
	if input.Query == "" {
		return task.QueryOutput{}, task.ErrEmptyQuery
	}

	uc.l.Infof(ctx, "AnswerQuery: user=%s query=%q", sc.UserID, input.Query)

	// Step 1: Search for relevant tasks
	searchResults, err := uc.vectorRepo.SearchTasks(ctx, repository.SearchTasksOptions{
		Query: input.Query,
		Limit: MaxTasksInContext,
	})
	if err != nil {
		return task.QueryOutput{}, fmt.Errorf("failed to search tasks: %w", err)
	}

	if len(searchResults) == 0 {
		return task.QueryOutput{
			Answer:      "Không tìm thấy task nào liên quan đến câu hỏi của bạn.",
			SourceCount: 0,
		}, nil
	}

	// Step 2: Build context with truncation
	var contextBuilder strings.Builder
	contextBuilder.WriteString("Ngữ cảnh (Các task liên quan):\n\n")

	for i, sr := range searchResults {
		memoTask, err := uc.memosRepo.GetTask(ctx, sr.MemoID)
		if err != nil {
			uc.l.Warnf(ctx, "Failed to fetch task %s: %v", sr.MemoID, err)
			continue
		}

		// ✅ CRITICAL: Truncate to prevent token overflow
		safeContent := truncateText(memoTask.Content, MaxCharsPerTask)

		contextBuilder.WriteString(fmt.Sprintf("-- Task %d (Độ phù hợp: %.0f%%, Link: %s) --\n%s\n\n",
			i+1, sr.Score*100, memoTask.MemoURL, safeContent))
	}

	// Step 3: Build prompt
	prompt := fmt.Sprintf(`%s

Nhiệm vụ: Trả lời câu hỏi sau dựa trên ngữ cảnh được cung cấp.
- Nếu ngữ cảnh không có thông tin, hãy nói rõ là không biết.
- Luôn đính kèm link task nếu có trích dẫn.
- Trả lời bằng tiếng Việt, ngắn gọn và rõ ràng.

Câu hỏi: "%s"`, contextBuilder.String(), input.Query)

	// Step 4: Call LLM
	req := gemini.GenerateRequest{
		Contents: []gemini.Content{
			{Parts: []gemini.Part{{Text: prompt}}},
		},
		GenerationConfig: &gemini.GenerationConfig{
			Temperature:     0.3, // Lower temperature for factual answers
			MaxOutputTokens: 1024,
		},
	}

	resp, err := uc.llm.GenerateContent(ctx, req)
	if err != nil {
		return task.QueryOutput{}, fmt.Errorf("LLM failed: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return task.QueryOutput{}, fmt.Errorf("empty LLM response")
	}

	answerText := resp.Candidates[0].Content.Parts[0].Text

	return task.QueryOutput{
		Answer:      answerText,
		SourceTasks: searchResults,
		SourceCount: len(searchResults),
	}, nil
}

// truncateText safely truncates text to maxLen (Unicode-safe for Vietnamese).
func truncateText(text string, maxLen int) string {
	runes := []rune(text)  // ✅ Convert to Unicode characters (not bytes)
	if len(runes) <= maxLen {
		return text
	}
	return string(runes[:maxLen]) + "... [đã cắt bớt]"
}
```

**File:** `internal/task/types.go` (update)

```go
// QueryInput is the input for RAG-based question answering.
type QueryInput struct {
	Query string // Natural language question
}

// QueryOutput is the result of RAG-based question answering.
type QueryOutput struct {
	Answer      string                      // LLM-generated answer
	SourceTasks []repository.SearchResult   // Source tasks used
	SourceCount int                         // Number of sources
}
```

**File:** `internal/task/interface.go` (update)

```go
type UseCase interface {
	CreateBulk(ctx context.Context, sc model.Scope, input CreateBulkInput) (CreateBulkOutput, error)
	Search(ctx context.Context, sc model.Scope, input SearchInput) (SearchOutput, error)
	AnswerQuery(ctx context.Context, sc model.Scope, input QueryInput) (QueryOutput, error) // ✅ NEW
}
```

---

### Task 3A.3: Webhook Sync with Exponential Backoff

**Mục tiêu:** Auto-update Qdrant khi Memos thay đổi

**File:** `internal/sync/handler.go` (NEW)

```go
package sync

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"autonomous-task-management/internal/task/repository"
	pkgLog "autonomous-task-management/pkg/log"
	pkgResponse "autonomous-task-management/pkg/response"
)

type WebhookHandler struct {
	memosRepo  repository.MemosRepository
	vectorRepo repository.VectorRepository
	l          pkgLog.Logger
}

func NewWebhookHandler(memosRepo repository.MemosRepository, vectorRepo repository.VectorRepository, l pkgLog.Logger) *WebhookHandler {
	return &WebhookHandler{
		memosRepo:  memosRepo,
		vectorRepo: vectorRepo,
		l:          l,
	}
}

// MemosWebhookPayload matches Memos API v1 webhook format.
type MemosWebhookPayload struct {
	ActivityType string `json:"activityType"` // e.g., "memos.memo.created"
	Memo         struct {
		Name string `json:"name"` // e.g., "memos/123"
		UID  string `json:"uid"`  // Short UID (Base58)
	} `json:"memo"`
}

// HandleMemosWebhook processes Memos webhook events.
func (h *WebhookHandler) HandleMemosWebhook(c *gin.Context) {
	ctx := c.Request.Context()

	var payload MemosWebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		h.l.Errorf(ctx, "webhook: failed to parse payload: %v", err)
		pkgResponse.Error(c, err, nil)
		return
	}

	h.l.Infof(ctx, "webhook: received %s for memo %s", payload.ActivityType, payload.Memo.UID)

	// Process in background to avoid blocking Memos
	go func(p MemosWebhookPayload) {
		// ✅ CRITICAL: Add timeout to prevent goroutine leak
		bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		memoID := p.Memo.UID

		switch p.ActivityType {
		case "memos.memo.created", "memos.memo.updated":
			// Re-embed task (upsert)
			h.syncWithRetry(bgCtx, memoID)

		case "memos.memo.deleted":
			// Delete from Qdrant
			if err := h.vectorRepo.DeleteTask(bgCtx, memoID); err != nil {
				h.l.Errorf(bgCtx, "webhook: failed to delete task %s: %v", memoID, err)
			} else {
				h.l.Infof(bgCtx, "webhook: deleted task %s from Qdrant", memoID)
			}
		}
	}(payload)

	// Acknowledge immediately
	pkgResponse.OK(c, map[string]string{"status": "accepted"})
}

// syncWithRetry embeds task to Qdrant with exponential backoff.
func (h *WebhookHandler) syncWithRetry(ctx context.Context, memoID string) {
	maxRetries := 3
	backoff := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		// Fetch task from Memos
		task, err := h.memosRepo.GetTask(ctx, memoID)
		if err != nil {
			h.l.Warnf(ctx, "webhook: fetch memo failed (retry %d/%d): %v", i+1, maxRetries, err)
			time.Sleep(backoff)
			backoff *= 2
			continue
		}

		// Embed to Qdrant
		if err := h.vectorRepo.EmbedTask(ctx, task); err != nil {
			h.l.Warnf(ctx, "webhook: embed failed (retry %d/%d): %v", i+1, maxRetries, err)
			time.Sleep(backoff)
			backoff *= 2
			continue
		}

		// Success
		h.l.Infof(ctx, "webhook: successfully synced task %s to Qdrant", memoID)
		return
	}

	// All retries failed
	h.l.Errorf(ctx, "webhook: FAILED to sync task %s after %d retries. Data drift occurred!", memoID, maxRetries)
}
```

**File:** `internal/sync/new.go`

```go
package sync

import (
	"autonomous-task-management/internal/task/repository"
	pkgLog "autonomous-task-management/pkg/log"
)

type WebhookHandler struct {
	memosRepo  repository.MemosRepository
	vectorRepo repository.VectorRepository
	l          pkgLog.Logger
}

func NewWebhookHandler(memosRepo repository.MemosRepository, vectorRepo repository.VectorRepository, l pkgLog.Logger) *WebhookHandler {
	return &WebhookHandler{
		memosRepo:  memosRepo,
		vectorRepo: vectorRepo,
		l:          l,
	}
}
```

---
### Task 3A.4: Update Telegram Handler (Multi-mode Interface)

**Mục tiêu:** Tách biệt 3 modes: Create (default), Search (`/search`), Agent (`/ask`)

**File:** `internal/task/delivery/telegram/handler.go` (update)

```go
func (h *handler) processMessage(ctx context.Context, msg *pkgTelegram.Message) error {
	sc := model.Scope{UserID: fmt.Sprintf("telegram_%d", msg.From.ID)}

	// Handle commands
	switch {
	case msg.Text == "/start":
		return h.handleStart(ctx, msg.Chat.ID)

	case msg.Text == "/help":
		return h.handleHelp(ctx, msg.Chat.ID)

	case strings.HasPrefix(msg.Text, "/search "):
		// Fast semantic search (Phase 3 Basic)
		query := strings.TrimSpace(strings.TrimPrefix(msg.Text, "/search"))
		return h.handleSearch(ctx, sc, query, msg.Chat.ID)

	case strings.HasPrefix(msg.Text, "/ask "):
		// Intelligent agent mode (Phase 3 Advanced)
		query := strings.TrimSpace(strings.TrimPrefix(msg.Text, "/ask"))
		return h.handleAgentOrchestrator(ctx, sc, query, msg.Chat.ID)

	default:
		// Default: Create task
		return h.handleCreateTask(ctx, sc, msg)
	}
}

// handleSearch performs fast semantic search (existing functionality).
func (h *handler) handleSearch(ctx context.Context, sc model.Scope, query string, chatID int64) error {
	if query == "" {
		return h.bot.SendMessage(chatID, "❌ Vui lòng nhập từ khóa tìm kiếm.\n\nVí dụ: `/search meeting tomorrow`")
	}

	h.bot.SendMessage(chatID, "🔍 Đang tìm kiếm...")

	// Use existing search functionality
	searchInput := task.SearchInput{Query: query}
	result, err := h.taskUseCase.Search(ctx, sc, searchInput)
	if err != nil {
		h.l.Errorf(ctx, "Search failed: %v", err)
		return h.bot.SendMessage(chatID, "❌ Lỗi tìm kiếm. Vui lòng thử lại.")
	}

	if len(result.Tasks) == 0 {
		return h.bot.SendMessage(chatID, "🤷‍♂️ Không tìm thấy task nào phù hợp.")
	}

	// Format results
	var response strings.Builder
	response.WriteString(fmt.Sprintf("🎯 Tìm thấy %d task:\n\n", len(result.Tasks)))

	for i, taskResult := range result.Tasks {
		response.WriteString(fmt.Sprintf("**%d. [%s](%s)**\n", i+1, taskResult.Title, taskResult.MemoURL))
		response.WriteString(fmt.Sprintf("📅 %s | 🎯 %.0f%%\n", taskResult.CreatedAt.Format("02/01"), taskResult.Score*100))
		
		// Show preview (first 100 chars)
		preview := taskResult.Content
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		response.WriteString(fmt.Sprintf("💭 %s\n\n", preview))
	}

	return h.bot.SendMessageWithMode(chatID, response.String(), "Markdown")
}

// handleAgentOrchestrator uses intelligent agent with tools.
func (h *handler) handleAgentOrchestrator(ctx context.Context, sc model.Scope, query string, chatID int64) error {
	if query == "" {
		return h.bot.SendMessage(chatID, "❌ Vui lòng nhập câu hỏi.\n\nVí dụ: `/ask Tôi có meeting nào vào thứ 2 không?`")
	}

	h.bot.SendMessage(chatID, "🧠 Agent đang suy nghĩ...")

	// Call orchestrator (agent will decide which tools to use)
	answer, err := h.orchestrator.ProcessQuery(ctx, query)
	if err != nil {
		h.l.Errorf(ctx, "Agent failed: %v", err)
		return h.bot.SendMessage(chatID, "❌ Lỗi hệ thống Agent. Vui lòng thử lại.")
	}

	return h.bot.SendMessageWithMode(chatID, answer, "Markdown")
}

// handleStart shows welcome message with all modes.
func (h *handler) handleStart(ctx context.Context, chatID int64) error {
	message := `👋 **Chào mừng đến với Task Management Bot!**

🎯 **3 chế độ sử dụng:**

**1. Tạo Task (Mặc định)**
Gửi tin nhắn bình thường để tạo task mới.
*Ví dụ: "Meeting với team lúc 2pm ngày mai"*

**2. Tìm kiếm nhanh**
\`/search [từ khóa]\` - Tìm task theo từ khóa
*Ví dụ: \`/search meeting tomorrow\`*

**3. Trợ lý thông minh**
\`/ask [câu hỏi]\` - Agent tự động tìm kiếm và phân tích
*Ví dụ: \`/ask Tôi có meeting nào vào thứ 2 không?\`*

Gõ \`/help\` để xem hướng dẫn chi tiết.`

	return h.bot.SendMessageWithMode(chatID, message, "Markdown")
}

// handleHelp shows detailed usage instructions.
func (h *handler) handleHelp(ctx context.Context, chatID int64) error {
	message := `📖 **Hướng dẫn sử dụng**

**🆕 Tạo Task**
Gửi tin nhắn bình thường:
• "Họp team lúc 10am ngày mai"
• "Deadline dự án ABC vào 15/3"
• "Gọi điện cho khách hàng XYZ"

**🔍 Tìm kiếm nhanh**
\`/search [từ khóa]\`
• \`/search meeting\` - Tìm tất cả meeting
• \`/search deadline march\` - Tìm deadline tháng 3
• \`/search client call\` - Tìm cuộc gọi khách hàng

**🧠 Trợ lý thông minh**
\`/ask [câu hỏi]\`
• \`/ask Tôi có meeting nào tuần này?\`
• \`/ask Deadline nào gần nhất?\`
• \`/ask Tóm tắt công việc hôm nay\`

**💡 Mẹo:**
• Agent mode (/ask) thông minh hơn nhưng chậm hơn
• Search mode (/search) nhanh hơn cho truy vấn đơn giản
• Tạo task trực tiếp bằng tin nhắn thường`

	return h.bot.SendMessageWithMode(chatID, message, "Markdown")
}
```

**File:** `internal/task/delivery/telegram/new.go` (update)

```go
package telegram

import (
	"autonomous-task-management/internal/agent/orchestrator"
	"autonomous-task-management/internal/task"
	pkgLog "autonomous-task-management/pkg/log"
	pkgTelegram "autonomous-task-management/pkg/telegram"
)

type handler struct {
	bot           pkgTelegram.Bot
	taskUseCase   task.UseCase
	orchestrator  *orchestrator.Orchestrator // ✅ NEW
	l             pkgLog.Logger
}

func New(bot pkgTelegram.Bot, taskUseCase task.UseCase, orchestrator *orchestrator.Orchestrator, l pkgLog.Logger) *handler {
	return &handler{
		bot:          bot,
		taskUseCase:  taskUseCase,
		orchestrator: orchestrator, // ✅ NEW
		l:            l,
	}
}
```

---

### Task 3A.5: Calendar Conflict Detection Tool

**Mục tiêu:** Agent có thể check lịch Google Calendar để phát hiện xung đột

**File:** `internal/agent/tools/check_calendar.go` (NEW)

```go
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"autonomous-task-management/internal/agent"
	"autonomous-task-management/pkg/gcalendar"
	pkgLog "autonomous-task-management/pkg/log"
)

type CheckCalendarTool struct {
	calendar gcalendar.Client
	l        pkgLog.Logger
}

func NewCheckCalendarTool(calendar gcalendar.Client, l pkgLog.Logger) *CheckCalendarTool {
	return &CheckCalendarTool{
		calendar: calendar,
		l:        l,
	}
}

func (t *CheckCalendarTool) Name() string {
	return "check_calendar"
}

func (t *CheckCalendarTool) Description() string {
	return "Check Google Calendar for events in a specific time range. Useful for detecting scheduling conflicts."
}

func (t *CheckCalendarTool) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"start_date": map[string]interface{}{
				"type":        "string",
				"description": "Start date in YYYY-MM-DD format",
			},
			"end_date": map[string]interface{}{
				"type":        "string",
				"description": "End date in YYYY-MM-DD format",
			},
			"time_zone": map[string]interface{}{
				"type":        "string",
				"description": "Time zone (e.g., 'Asia/Ho_Chi_Minh')",
				"default":     "Asia/Ho_Chi_Minh",
			},
		},
		"required": []string{"start_date", "end_date"},
	}
}

type CheckCalendarInput struct {
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
	TimeZone  string `json:"time_zone"`
}

type CheckCalendarOutput struct {
	Events      []CalendarEvent `json:"events"`
	EventCount  int             `json:"event_count"`
	HasConflict bool            `json:"has_conflict"`
	Summary     string          `json:"summary"`
}

type CalendarEvent struct {
	Title     string    `json:"title"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Location  string    `json:"location,omitempty"`
}

func (t *CheckCalendarTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	// Parse input
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}

	var params CheckCalendarInput
	if err := json.Unmarshal(inputBytes, &params); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	// Set default timezone
	if params.TimeZone == "" {
		params.TimeZone = "Asia/Ho_Chi_Minh"
	}

	t.l.Infof(ctx, "check_calendar: checking %s to %s (%s)", params.StartDate, params.EndDate, params.TimeZone)

	// Parse dates
	startTime, err := time.Parse("2006-01-02", params.StartDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start_date format: %w", err)
	}

	endTime, err := time.Parse("2006-01-02", params.EndDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end_date format: %w", err)
	}

	// Add timezone and set time bounds
	loc, err := time.LoadLocation(params.TimeZone)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone: %w", err)
	}

	startTime = time.Date(startTime.Year(), startTime.Month(), startTime.Day(), 0, 0, 0, 0, loc)
	endTime = time.Date(endTime.Year(), endTime.Month(), endTime.Day(), 23, 59, 59, 0, loc)

	// Query Google Calendar
	events, err := t.calendar.GetEvents(ctx, gcalendar.GetEventsOptions{
		TimeMin: startTime,
		TimeMax: endTime,
	})
	if err != nil {
		t.l.Errorf(ctx, "check_calendar: failed to get events: %v", err)
		return CheckCalendarOutput{
			Events:      []CalendarEvent{},
			EventCount:  0,
			HasConflict: false,
			Summary:     fmt.Sprintf("❌ Không thể truy cập lịch: %v", err),
		}, nil
	}

	// Convert to output format
	var calendarEvents []CalendarEvent
	for _, event := range events {
		calendarEvents = append(calendarEvents, CalendarEvent{
			Title:     event.Summary,
			StartTime: event.Start,
			EndTime:   event.End,
			Location:  event.Location,
		})
	}

	// Generate summary
	var summary string
	if len(calendarEvents) == 0 {
		summary = fmt.Sprintf("📅 Không có sự kiện nào từ %s đến %s", params.StartDate, params.EndDate)
	} else {
		summary = fmt.Sprintf("📅 Tìm thấy %d sự kiện từ %s đến %s:\n", len(calendarEvents), params.StartDate, params.EndDate)
		for i, event := range calendarEvents {
			summary += fmt.Sprintf("%d. %s (%s - %s)\n", 
				i+1, 
				event.Title,
				event.StartTime.Format("02/01 15:04"),
				event.EndTime.Format("15:04"))
		}
	}

	return CheckCalendarOutput{
		Events:      calendarEvents,
		EventCount:  len(calendarEvents),
		HasConflict: len(calendarEvents) > 0,
		Summary:     summary,
	}, nil
}

// Verify interface compliance
var _ agent.Tool = (*CheckCalendarTool)(nil)
```

---

### Task 3A.6: Wiring in main.go

**Mục tiêu:** Kết nối tất cả components trong main.go

**File:** `cmd/api/main.go` (update)

```go
// Add imports
import (
	"autonomous-task-management/internal/agent/orchestrator"
	"autonomous-task-management/internal/sync"
	// ... existing imports
)

func main() {
	// ... existing setup ...

	// ✅ NEW: Initialize orchestrator
	orchestratorInstance := orchestrator.New(geminiClient, toolRegistry, logger)

	// ✅ UPDATE: Telegram handler with orchestrator
	telegramHandler := telegram.New(telegramBot, taskUseCase, orchestratorInstance, logger)

	// ✅ NEW: Webhook sync handler
	webhookHandler := sync.NewWebhookHandler(memosRepo, vectorRepo, logger)

	// ✅ NEW: Register webhook routes
	apiGroup := router.Group("/api/v1")
	{
		// Existing routes...
		apiGroup.POST("/webhook/memos", webhookHandler.HandleMemosWebhook)
	}

	// ... rest of main.go ...
}
```

**File:** `internal/agent/tools/registry.go` (update)

```go
func (r *Registry) registerDefaultTools(
	taskUseCase task.UseCase,
	calendar gcalendar.Client,
	l pkgLog.Logger,
) {
	// Existing tools
	r.Register(NewSearchTasksTool(taskUseCase, l))
	
	// ✅ NEW: Calendar tool
	r.Register(NewCheckCalendarTool(calendar, l))
}
```

---

## Configuration Updates

### Update config.yaml

```yaml
# Add webhook configuration
webhook:
  memos:
    enabled: true
    endpoint: "/api/v1/webhook/memos"
    secret: "${WEBHOOK_SECRET}" # Optional for authentication

# Add agent configuration
agent:
  max_steps: 5
  temperature: 0.3
  max_output_tokens: 1024

# Add RAG configuration
rag:
  max_tasks_in_context: 5
  max_chars_per_task: 800
  
# Add sync configuration
sync:
  retry:
    max_attempts: 3
    initial_backoff: "2s"
    max_backoff: "30s"
```

### Update .env.example

```bash
# Add webhook secret
WEBHOOK_SECRET=your-webhook-secret-here

# Agent configuration
AGENT_MAX_STEPS=5
AGENT_TEMPERATURE=0.3

# RAG configuration
RAG_MAX_TASKS=5
RAG_MAX_CHARS_PER_TASK=800
```

---

## Testing Strategy

### Unit Tests

**File:** `internal/agent/orchestrator/orchestrator_test.go`

```go
func TestOrchestrator_ProcessQuery_MaxSteps(t *testing.T) {
	// Test that orchestrator stops after MaxAgentSteps
	// Mock LLM to always return function calls
	// Verify it returns "exceeded max steps" message
}

func TestOrchestrator_ProcessQuery_Success(t *testing.T) {
	// Test successful query processing
	// Mock LLM to return final answer after 2 steps
	// Verify correct response
}
```

**File:** `internal/task/usecase/answer_query_test.go`

```go
func TestAnswerQuery_TokenTruncation(t *testing.T) {
	// Test that tasks are truncated to MaxCharsPerTask
	// Create tasks with >800 chars content
	// Verify truncation works correctly
}

func TestAnswerQuery_EmptyResults(t *testing.T) {
	// Test behavior when no relevant tasks found
	// Verify appropriate message returned
}
```

**File:** `internal/sync/handler_test.go`

```go
func TestWebhookHandler_RetryLogic(t *testing.T) {
	// Test exponential backoff retry
	// Mock failures for first 2 attempts, success on 3rd
	// Verify retry intervals: 2s, 4s, success
}

func TestWebhookHandler_MaxRetriesExceeded(t *testing.T) {
	// Test behavior when all retries fail
	// Verify error logging
}
```

### Integration Tests

**File:** `test/integration/agent_test.go`

```go
func TestAgent_EndToEnd(t *testing.T) {
	// Test complete agent flow:
	// 1. User sends "/ask" command
	// 2. Agent searches tasks
	// 3. Agent checks calendar
	// 4. Agent returns intelligent response
}

func TestWebhook_EndToEnd(t *testing.T) {
	// Test webhook sync:
	// 1. Send webhook payload
	// 2. Verify task embedded to Qdrant
	// 3. Verify search returns updated results
}
```

### Load Tests

```bash
# Test webhook performance
hey -n 1000 -c 10 -m POST -H "Content-Type: application/json" \
  -d '{"activityType":"memos.memo.created","memo":{"uid":"test123"}}' \
  http://localhost:8080/api/v1/webhook/memos

# Test agent performance
hey -n 100 -c 5 -m POST -H "Content-Type: application/json" \
  -d '{"message":{"text":"/ask Tôi có meeting nào hôm nay?"}}' \
  http://localhost:8080/api/v1/telegram/webhook
```

---

## Implementation Checklist

### Phase 3A.1: ReAct Agent Orchestrator
- [ ] Create `internal/agent/orchestrator/orchestrator.go`
- [ ] Create `internal/agent/orchestrator/new.go`
- [ ] Implement `ProcessQuery()` with max steps limit
- [ ] Add comprehensive logging for debugging
- [ ] Write unit tests for max steps and success cases
- [ ] Test with mock LLM responses

### Phase 3A.2: RAG with Context Truncation
- [ ] Create `internal/task/usecase/answer_query.go`
- [ ] Implement `truncateText()` helper function
- [ ] Update `internal/task/types.go` with QueryInput/Output
- [ ] Update `internal/task/interface.go` with AnswerQuery method
- [ ] Write tests for token truncation logic
- [ ] Verify context quality with truncated content

### Phase 3A.3: Webhook Sync with Exponential Backoff
- [ ] Create `internal/sync/handler.go`
- [ ] Create `internal/sync/new.go`
- [ ] Implement `syncWithRetry()` with exponential backoff
- [ ] Define `MemosWebhookPayload` struct
- [ ] Write tests for retry logic and max retries
- [ ] Test with network failures and timeouts

### Phase 3A.4: Update Telegram Handler
- [ ] Update `internal/task/delivery/telegram/handler.go`
- [ ] Update `internal/task/delivery/telegram/new.go`
- [ ] Implement `/ask` command handler
- [ ] Update `/start` and `/help` messages
- [ ] Test all three modes: create, search, ask
- [ ] Verify markdown formatting works correctly

### Phase 3A.5: Calendar Conflict Detection Tool
- [ ] Create `internal/agent/tools/check_calendar.go`
- [ ] Implement calendar event parsing
- [ ] Add timezone support
- [ ] Update tool registry to include calendar tool
- [ ] Write tests for date parsing and event formatting
- [ ] Test with real Google Calendar API

### Phase 3A.6: Wiring in main.go
- [ ] Update `cmd/api/main.go` with orchestrator
- [ ] Register webhook routes
- [ ] Update tool registry with calendar tool
- [ ] Update configuration files
- [ ] Test complete application startup
- [ ] Verify all dependencies injected correctly

### Configuration & Environment
- [ ] Update `config/config.yaml` with new sections
- [ ] Update `.env.example` with new variables
- [ ] Update `config/config.go` to parse new fields
- [ ] Test configuration loading
- [ ] Document environment variables

### Testing & Quality Assurance
- [ ] Write unit tests for all new components
- [ ] Write integration tests for end-to-end flows
- [ ] Run load tests on webhook endpoint
- [ ] Test error handling and edge cases
- [ ] Verify logging and monitoring
- [ ] Test with real Telegram bot and Google Calendar

---

## Troubleshooting Guide

### Agent Issues

**Problem:** Agent exceeds max steps
```
Agent exceeded max steps (5)
```
**Solution:** 
- Check if LLM is stuck in loop
- Verify tool responses are informative
- Consider increasing MaxAgentSteps if needed

**Problem:** Agent calls non-existent tool
```
Tool xyz not found
```
**Solution:**
- Verify tool is registered in registry
- Check tool name matches exactly
- Update LLM prompt to use correct tool names

### RAG Issues

**Problem:** Token overflow despite truncation
```
context too long: 50000 tokens
```
**Solution:**
- Reduce MaxTasksInContext (5 → 3)
- Reduce MaxCharsPerTask (800 → 500)
- Check if system prompt is too long

**Problem:** Poor answer quality after truncation
```
Answers are too generic
```
**Solution:**
- Improve truncation logic to preserve key information
- Extract title + summary instead of raw content
- Increase MaxCharsPerTask if token budget allows

### Webhook Issues

**Problem:** Webhook sync fails repeatedly
```
Sync failed after 3 retries
```
**Solution:**
- Check Memos API connectivity
- Verify Qdrant is running and accessible
- Check network stability
- Consider increasing retry attempts

**Problem:** Webhook payload parsing fails
```
Failed to parse payload
```
**Solution:**
- Verify Memos webhook format matches MemosWebhookPayload
- Check JSON structure in logs
- Update payload struct if Memos API changed

### Performance Issues

**Problem:** Agent responses are slow (>10s)
```
Agent taking too long to respond
```
**Solution:**
- Optimize tool execution time
- Reduce LLM temperature for faster responses
- Cache frequent queries
- Consider async processing for complex queries

**Problem:** High memory usage
```
Memory usage increasing over time
```
**Solution:**
- Check for goroutine leaks in webhook handler
- Verify context cancellation
- Monitor vector embeddings memory usage
- Add memory profiling

---

## Performance Considerations

### Agent Performance
- **Target:** <5s response time for simple queries
- **Optimization:** Cache tool results for repeated calls
- **Monitoring:** Track steps per query, tool execution time

### RAG Performance
- **Target:** <2s for search + LLM generation
- **Optimization:** Pre-compute embeddings, optimize Qdrant queries
- **Monitoring:** Track search latency, context size

### Webhook Performance
- **Target:** <100ms acknowledgment, background processing
- **Optimization:** Async processing, batch updates
- **Monitoring:** Track sync success rate, retry frequency

### Memory Usage
- **Target:** <512MB for agent components
- **Optimization:** Limit conversation history, cleanup old contexts
- **Monitoring:** Track goroutine count, memory allocation

---

## Deliverables Summary

### Core Components
1. **ReAct Agent Orchestrator** - Multi-step reasoning with tool calls
2. **RAG with Context Truncation** - Token-safe question answering
3. **Webhook Sync with Retry** - Auto-update Qdrant from Memos
4. **Multi-mode Telegram Interface** - Create/Search/Ask modes
5. **Calendar Conflict Detection** - Google Calendar integration

### Key Features
- **Intelligent Agent:** Can reason, search, and check calendar
- **Token Safety:** Prevents overflow with smart truncation
- **Data Consistency:** Auto-sync between Memos and Qdrant
- **User Choice:** Fast search vs intelligent analysis
- **Production Ready:** Comprehensive error handling and logging

### Success Metrics
- Agent completes 95% of queries within 5 steps
- RAG maintains <1200 tokens per query
- Webhook sync achieves 99% success rate
- User satisfaction with intelligent responses
- System stability under load

**Estimated Implementation Time:** 4-6 days (32-48 hours)

**Dependencies:** Phase 3 Basic must be 100% complete and tested

**Risk Level:** Medium (well-defined architecture, proven patterns)

---

## 🎯 Next Steps

1. **Verify Phase 3 Basic** - Ensure all basic functionality works
2. **Start with Orchestrator** - Core agent loop is foundation
3. **Add RAG Truncation** - Prevent token issues early
4. **Implement Webhook Sync** - Maintain data consistency
5. **Update Telegram Interface** - User-facing improvements
6. **Add Calendar Tool** - Enhanced intelligence
7. **Integration Testing** - End-to-end validation

**Ready to begin Phase 3 Advanced implementation!** 🚀