# Phase 5: Verification, Optimization & Testing

Based on user feedback, the AI Agent currently struggles with basic conversational inputs and relative timeframes during `/ask` queries. Furthermore, the system requires comprehensive testing to ensure stability and correctness.

This phase focuses on improving the AI Agent's intelligence (Context Injection), building a robust E2E test suite for the Telegram handler, and increasing unit test coverage across the backend to >80%.

## 1. Agent Intelligence & UX Improvements

### 1.1 Context Injection (`internal/agent/orchestrator/orchestrator.go`)

- **Issue:** The Agent (`/ask`) currently has no concept of the current date or timezone because no `SystemInstruction` is provided in the `gemini.GenerateRequest`.
- **Solution:** Inject a dynamic `SystemInstruction` into the Orchestrator's Gemini requests.
  - Provide the Agent's persona ("You are an intelligent task management assistant").
  - Provide the **current date, time, and timezone** so the LLM can resolve relative terms like "today", "tomorrow", or "this week".

### 1.2 Conversational Fallback (`internal/task/delivery/telegram/handler.go`)

- **Issue:** Any text without a slash command falls into `handleCreateTask`. If the user asks a conversational question (e.g., "What can you do?"), the LLM returns 0 tasks, resulting in an unfriendly error message.
- **Solution:**
  - Improve the error handling in `handleCreateTask`. If the LLM returns 0 tasks, instead of failing, automatically fallback to `handleAsk` (the Orchestrator) to answer the user conversationally.
  - Modify the fast semantic search `/search` prompt and `AnswerQuery` prompt to also include timezone context.

## 2. Comprehensive E2E Testing

### 2.1 Telegram Webhook E2E Tests

- **Objective:** Simulate real-world Telegram messages to test the entire request lifecycle (Routing -> Parsing -> LLM -> Response).
- **Target:** `internal/task/delivery/telegram/handler_test.go`
- **Test Cases to Build:**
  1. **Task Creation:** Normal task creation (e.g., "Họp 9h sáng mai").
  2. **Conversational Fallback:** Non-task conversational input (e.g., "Bạn làm được gì?").
  3. **Relative Time Queries:** `/ask lịch trình tuần này` (mocking the LLM to verify dates are requested correctly).
  4. **Checklist manipulation:** `/check 123` and `/uncheck 123`.
  5. **Search queries:** `/search meeting`.

### 2.2 Logging & Debugging Enhancements

- Enhance structured logging (Zap) around LLM inputs and outputs.
- Log the exact `SystemInstruction` and Tool Calls the Agent executes to make E2E test debugging straightforward.

## 3. Unit Testing & Coverage (>80%)

### 3.1 Target Areas

We will systematically write unit tests using Go's standard library and testify (mocking) to achieve >80% coverage on core packages:

- `internal/agent/orchestrator/`: Mock the Gemini LLM and Tool Registry to test step-by-step reasoning limits.
- `internal/agent/tools/`: Test individual tool parameter parsing and execution.
- `internal/task/usecase/`: Test `parseInputWithLLM`, `resolveDueDates`, and `AnswerQuery`.
- `pkg/datemath/`: Ensure relative date parsing is mathematically perfect across edge cases (e.g., leap years, week boundaries).
- `internal/webhook/`: Test GitHub/GitLab signature validation (already mostly done, but verify coverage).

### 3.2 Execution Strategy

1. Run `go test -coverprofile=coverage.out ./...` to establish the baseline.
2. Iteratively write tests for the lowest-coverage packages.
3. Use `go tool cover -html=coverage.out` to visually identify missing branches for final 100% logic coverage in core packages.

---

## Next Steps for Execution

1. Begin by implementing the **Context Injection** and **Conversational Fallback** to fix the immediate UX bugs.
2. Build the **E2E Test Suite** for the Telegram Handler.
3. Iterate over the remaining packages to hit the **>80% Coverage** goal.

---

## 🚨 Phân tích Chuyên sâu: Bệnh "Mù thời gian" và "Máy móc" của AI

### Vấn đề 1: LLM Temporal Blindness (Mù thời gian)

**Triệu chứng (từ ảnh image_441c12.jpg):**

```
User: "Báo cáo lịch trình tuần này"
Agent: "Vui lòng cho biết ngày bắt đầu và kết thúc..."
```

**Nguyên nhân gốc rễ:**

- LLM không có khái niệm về thời gian thực tế
- Không biết "hôm nay" là ngày nào, "tuần này" là từ ngày X đến ngày Y
- `gemini.GenerateRequest` trong `orchestrator.go` không có `SystemInstruction` chứa context thời gian

**Tác động:**

- UX cực kỳ tệ: User phải tự tính toán và nhập ngày thủ công
- Agent mất đi khả năng hiểu ngữ cảnh thời gian tương đối
- Không thể trả lời các câu hỏi như "lịch trình ngày mai", "deadline tuần sau"

**Giải pháp: Context Injection (Mục 1.1)**

Inject thông tin thời gian vào `SystemInstruction` của mỗi request:

```go
// File: internal/agent/orchestrator/orchestrator.go
// Trong hàm ProcessQuery

import "time"

func (o *Orchestrator) ProcessQuery(ctx context.Context, query string) (string, error) {
    // ✅ CRITICAL FIX: Inject temporal context
    currentTime := time.Now().In(o.timezone) // Load từ config
    dateContext := fmt.Sprintf(
        "Hôm nay là %s, ngày %s. Timezone: %s.",
        currentTime.Weekday().String(),
        currentTime.Format("02/01/2006 15:04:05"),
        currentTime.Location().String(),
    )

    systemPrompt := `Bạn là một trợ lý quản lý công việc cá nhân cực kỳ thông minh.
Nhiệm vụ của bạn là tư vấn, giải đáp lịch trình và hỗ trợ người dùng.

LUÔN LUÔN ghi nhớ thông tin thời gian sau để nội suy các mốc thời gian tương đối:
` + dateContext + `

Khi người dùng hỏi về "tuần này", "ngày mai", "tháng sau", hãy tự động tính toán dựa trên thông tin trên.
Không bao giờ hỏi ngược lại người dùng về ngày tháng cụ thể.`

    req := gemini.GenerateRequest{
        SystemInstruction: &gemini.Content{
            Parts: []gemini.Part{{Text: systemPrompt}},
        },
        Contents: []gemini.Content{
            {Role: "user", Parts: []gemini.Part{{Text: query}}},
        },
        Tools: o.registry.ToFunctionDefinitions(),
    }

    // ... rest of logic
}
```

**Lợi ích:**

- Agent tự động hiểu "tuần này" = từ ngày X đến ngày Y
- Không cần user nhập ngày thủ công
- Trải nghiệm tự nhiên như chat với người thật

---

### Vấn đề 2: Strict Routing (Bệnh "Máy móc")

**Triệu chứng (từ ảnh image_441c12.jpg):**

```
User: "Bạn có thể làm được những gì?"
System: "no tasks parsed from input" (Lỗi!)
```

**Nguyên nhân gốc rễ:**

- Mọi tin nhắn không có `/` đều bị ép vào luồng `handleCreateTask`
- LLM parse không ra task nào → Trả về lỗi thay vì trả lời conversational
- Thiếu fallback mechanism cho các câu hỏi thông thường

**Tác động:**

- User không thể hỏi bot về chức năng
- Mọi câu hỏi đều bị coi là "tạo task" → Trải nghiệm cứng nhắc
- Bot mất đi tính "thông minh" và "linh hoạt"

**Giải pháp: Conversational Fallback (Mục 1.2)**

Sửa logic trong `handleCreateTask` để tự động chuyển sang Agent mode khi LLM không parse được task:

```go
// File: internal/task/delivery/telegram/handler.go
// Trong hàm handleCreateTask

func (h *handler) handleCreateTask(ctx context.Context, sc model.Scope, msg *pkgTelegram.Message) error {
    // Notify user
    if err := h.bot.SendMessage(msg.Chat.ID, "⏳ Đang xử lý..."); err != nil {
        h.l.Warnf(ctx, "telegram handler: failed to send ack message: %v", err)
    }

    input := task.CreateBulkInput{
        RawText:        msg.Text,
        TelegramChatID: msg.Chat.ID,
    }

    output, err := h.uc.CreateBulk(ctx, sc, input)

    // ✅ CRITICAL FIX: Conversational Fallback
    if err != nil {
        // Check if error is "no tasks parsed"
        if errors.Is(err, task.ErrNoTasksParsed) {
            h.l.Infof(ctx, "No tasks parsed, falling back to conversational agent for text: %s", msg.Text)

            // Kích hoạt luồng Agent (/ask) tự động
            return h.handleAgentOrchestrator(ctx, sc, msg.Text, msg.Chat.ID)
        }

        // Nếu là lỗi thực sự khác (như API sập, DB sập)
        h.l.Errorf(ctx, "CreateBulk failed: %v", err)
        return h.bot.SendMessage(msg.Chat.ID, "Có lỗi khi xử lý hệ thống. Vui lòng thử lại.")
    }

    // ... rest of success logic
}
```

**Lợi ích:**

- User có thể chat tự nhiên: "Bạn làm được gì?", "Giúp tôi với"
- Bot tự động phân biệt: Tạo task vs Trò chuyện
- Trải nghiệm mượt mà, không cần nhớ lệnh `/ask`

---

### Vấn đề 3: Session Memory (Mất trí nhớ ngắn hạn)

**Triệu chứng (từ ảnh image_441c12.jpg):**

```
User: /ask Báo cáo lịch trình...
Agent: Vui lòng cho tôi biết ngày bắt đầu...
User: tự lấy ngày hôm nay và đoán đi (không có /ask)
System: "no tasks parsed" (Lỗi!)
User: /ask tự lấy ngày...
Agent: Tuần này bạn không có lịch... (Hallucination!)
```

**Nguyên nhân gốc rễ:**

- Orchestrator ở Phase 3 được thiết kế là **Stateless** (Không trạng thái)
- Mỗi lần gọi `/ask` tạo một `gemini.GenerateRequest` mới tinh
- Không hề nhớ 5 giây trước user vừa hỏi về "lịch trình"
- Bước 4 trả lời "ảo giác" vì không có context từ câu hỏi trước

**Tác động:**

- Agent không thể duy trì hội thoại nhiều lượt
- User phải lặp lại context mỗi lần hỏi
- Trải nghiệm rời rạc, không liền mạch

**Giải pháp: Session Memory Cache (Mục 1.3 - MỚI)**

Triển khai cache lịch sử chat với TTL 10 phút:

```go
// File: internal/agent/orchestrator/types.go
package orchestrator

import "time"

type SessionMemory struct {
    UserID      string
    Messages    []gemini.Content // Lịch sử hội thoại
    LastUpdated time.Time
}

// File: internal/agent/orchestrator/orchestrator.go
import (
    "sync"
    "time"
)

type Orchestrator struct {
    // ... existing fields
    sessionCache map[string]*SessionMemory
    cacheMutex   sync.RWMutex
    cacheTTL     time.Duration
}

func New(...) *Orchestrator {
    return &Orchestrator{
        // ... existing fields
        sessionCache: make(map[string]*SessionMemory),
        cacheTTL:     10 * time.Minute,
    }
}

// getSession retrieves or creates session for user
func (o *Orchestrator) getSession(userID string) *SessionMemory {
    o.cacheMutex.Lock()
    defer o.cacheMutex.Unlock()

    session, exists := o.sessionCache[userID]
    if !exists || time.Since(session.LastUpdated) > o.cacheTTL {
        session = &SessionMemory{
            UserID:      userID,
            Messages:    []gemini.Content{},
            LastUpdated: time.Now(),
        }
        o.sessionCache[userID] = session
    }

    return session
}

// ProcessQuery với session memory
func (o *Orchestrator) ProcessQuery(ctx context.Context, userID string, query string) (string, error) {
    // ✅ NEW: Load session history
    session := o.getSession(userID)

    // Build request với lịch sử
    req := gemini.GenerateRequest{
        SystemInstruction: &gemini.Content{
            Parts: []gemini.Part{{Text: o.buildSystemPrompt()}},
        },
        Contents: append(session.Messages, gemini.Content{
            Role:  "user",
            Parts: []gemini.Part{{Text: query}},
        }),
        Tools: o.registry.ToFunctionDefinitions(),
    }

    // ... existing ReAct loop logic

    // ✅ NEW: Save to session after getting final answer
    session.Messages = append(session.Messages,
        gemini.Content{Role: "user", Parts: []gemini.Part{{Text: query}}},
        gemini.Content{Role: "model", Parts: []gemini.Part{{Text: finalAnswer}}},
    )

    // Limit history to last 5 turns (10 messages)
    if len(session.Messages) > 10 {
        session.Messages = session.Messages[len(session.Messages)-10:]
    }

    session.LastUpdated = time.Now()

    return finalAnswer, nil
}
```

**Lợi ích:**

- Agent nhớ 3-5 cặp câu hỏi/trả lời gần nhất
- Hội thoại liền mạch, không cần lặp lại context
- Auto-cleanup sau 10 phút không hoạt động (tránh memory leak)

---

## 📋 Implementation Checklist

### Phase 5.1: Context Injection & Conversational Fallback

**Files to modify:**

- [ ] `internal/agent/orchestrator/orchestrator.go`
  - [ ] Add `timezone` field to Orchestrator struct
  - [ ] Implement `buildSystemPrompt()` với temporal context
  - [ ] Update `ProcessQuery()` để inject SystemInstruction
- [ ] `internal/task/delivery/telegram/handler.go`
  - [ ] Update `handleCreateTask()` với conversational fallback
  - [ ] Add `errors.Is(err, task.ErrNoTasksParsed)` check
- [ ] `internal/task/errors.go`
  - [ ] Define `ErrNoTasksParsed = errors.New("no tasks parsed from input")`
- [ ] `config/config.yaml`
  - [ ] Add `timezone: "Asia/Ho_Chi_Minh"` to app config

**Testing:**

- [ ] Test: `/ask lịch trình tuần này` → Agent tự tính ngày (không hỏi ngược)
- [ ] Test: "Bạn làm được gì?" → Agent trả lời (không báo lỗi)
- [ ] Test: "Tìm hiểu cách tích hợp VNPay" → Tạo task (không trigger search)

---

### Phase 5.2: Session Memory

**Files to create/modify:**

- [ ] `internal/agent/orchestrator/types.go`
  - [ ] Define `SessionMemory` struct
- [ ] `internal/agent/orchestrator/orchestrator.go`
  - [ ] Add `sessionCache map[string]*SessionMemory`
  - [ ] Implement `getSession(userID string) *SessionMemory`
  - [ ] Update `ProcessQuery()` để load/save session
  - [ ] Add background goroutine để cleanup expired sessions
- [ ] `internal/task/delivery/telegram/handler.go`
  - [ ] Update `handleAgentOrchestrator()` để pass `userID` vào Orchestrator

**Testing:**

- [ ] Test: Multi-turn conversation
  ```
  User: /ask Tôi có meeting nào tuần này?
  Agent: Bạn có 3 meetings...
  User: Cái nào quan trọng nhất? (không cần /ask)
  Agent: Meeting với CEO vào thứ 2... (nhớ context)
  ```
- [ ] Test: Session expiry sau 10 phút
- [ ] Test: Memory limit (chỉ giữ 5 turns gần nhất)

---

### Phase 5.3: E2E Testing

**Files to create:**

- [ ] `internal/task/delivery/telegram/handler_test.go`
  - [ ] Test: Normal task creation
  - [ ] Test: Conversational fallback
  - [ ] Test: Relative time queries với mocked LLM
  - [ ] Test: Checklist manipulation
  - [ ] Test: Search queries
  - [ ] Test: Multi-turn conversation với session memory

**Mock Strategy:**

- Mock `task.UseCase` để control LLM output
- Mock `orchestrator.Orchestrator` để verify tool calls
- Use `httptest.NewRecorder()` để capture Telegram responses

---

### Phase 5.4: Unit Testing (>80% Coverage)

**Target packages:**

- [ ] `internal/agent/orchestrator/` (>80%)
  - [ ] Test: Max steps limit (5 steps)
  - [ ] Test: Tool execution success/failure
  - [ ] Test: Session memory load/save
- [ ] `internal/agent/tools/` (>90%)
  - [ ] Test: Each tool với valid/invalid inputs
  - [ ] Test: Error handling
- [ ] `pkg/datemath/` (100%)
  - [ ] Test: All relative date formats
  - [ ] Test: Edge cases (leap year, week boundaries)
  - [ ] Test: Timezone handling
- [ ] `internal/task/usecase/` (>80%)
  - [ ] Test: `parseInputWithLLM` với mocked Gemini
  - [ ] Test: `resolveDueDates` với various inputs
  - [ ] Test: `AnswerQuery` với RAG context

**Execution:**

```bash
# Baseline coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep total

# Target: >80% total coverage
# Focus on packages with <50% first
```

---

## 🎯 Success Criteria

### UX Improvements

- [x] Agent hiểu "tuần này", "ngày mai" mà không cần hỏi ngược
- [x] User có thể chat tự nhiên mà không cần nhớ lệnh
- [x] Agent nhớ context trong hội thoại nhiều lượt

### Testing

- [x] E2E test coverage cho tất cả Telegram commands
- [x] Unit test coverage >80% cho core packages
- [x] Zero regression bugs từ Phase 1-4

### Performance

- [x] Session cache không gây memory leak
- [x] Response time <3s cho conversational queries
- [x] LLM API cost không tăng >20% (do session history)

---

## 💡 Pro Tips

### Tip 1: Optimize Session History Size

Chỉ lưu 5 turns gần nhất (10 messages) để:

- Giảm token cost cho LLM
- Tránh context window overflow
- Maintain conversation coherence

**Lý do chọn số 10 (số chẵn):**

- Đảm bảo không bao giờ cắt lẻ một cặp câu hỏi-trả lời
- Giữ context truyền vào LLM luôn chuẩn xác
- Mỗi turn = 1 user message + 1 model response = 2 messages

### Tip 2: Graceful Degradation

Nếu session cache fail (Redis down), fallback về stateless mode:

```go
session, err := o.getSession(userID)
if err != nil {
    // Fallback: Process without history
    session = &SessionMemory{Messages: []gemini.Content{}}
}
```

### Tip 3: User Feedback Loop

Thêm command `/reset` để user clear session history khi Agent bị "confused":

```go
case strings.HasPrefix(msg.Text, "/reset"):
    o.clearSession(userID)
    return h.bot.SendMessage(chatID, "✅ Đã xóa lịch sử hội thoại. Bắt đầu lại từ đầu!")
```

### Tip 4: Active Eviction (Dọn dẹp bộ nhớ chủ động) ⚠️ CRITICAL

**Vấn đề với Lazy Eviction:**

```go
// ❌ WRONG: Lazy eviction - memory leak!
func (o *Orchestrator) getSession(userID string) *SessionMemory {
    session, exists := o.sessionCache[userID]
    if !exists || time.Since(session.LastUpdated) > o.cacheTTL {
        // Chỉ ghi đè khi user chat lại
        // Nếu user không chat nữa → memory leak!
        session = &SessionMemory{...}
        o.sessionCache[userID] = session
    }
    return session
}
```

**Tác động:**

- User chat xong và không bao giờ chat lại → session kẹt mãi trong map
- Sau 1 tháng: hàng nghìn sessions zombie → OOM crash
- Map không tự dọn dẹp → cần active cleanup

**Giải pháp: Background Goroutine với time.Ticker**

```go
// File: internal/agent/orchestrator/orchestrator.go

func New(...) *Orchestrator {
    o := &Orchestrator{
        // ... existing fields
        sessionCache: make(map[string]*SessionMemory),
        cacheTTL:     10 * time.Minute,
    }

    // ✅ CRITICAL: Start background cleanup goroutine
    go o.cleanupExpiredSessions()

    return o
}

// cleanupExpiredSessions runs every 5 minutes to remove expired sessions
func (o *Orchestrator) cleanupExpiredSessions() {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        o.cacheMutex.Lock()

        now := time.Now()
        expiredKeys := make([]string, 0)

        // Find expired sessions
        for userID, session := range o.sessionCache {
            if now.Sub(session.LastUpdated) > o.cacheTTL {
                expiredKeys = append(expiredKeys, userID)
            }
        }

        // Delete expired sessions
        for _, userID := range expiredKeys {
            delete(o.sessionCache, userID)
        }

        o.cacheMutex.Unlock()

        if len(expiredKeys) > 0 {
            o.l.Infof(context.Background(),
                "Cleaned up %d expired sessions", len(expiredKeys))
        }
    }
}
```

**Lợi ích:**

- Tự động dọn dẹp sessions không hoạt động
- Chạy mỗi 5 phút → không ảnh hưởng performance
- Tránh memory leak trong production

### Tip 5: Xử lý Conversational Fallback Context ⚠️ IMPORTANT

**Vấn đề:**

```go
// Khi user gửi tin nhắn thường (không có /ask)
// handleCreateTask → fallback → handleAgentOrchestrator

// ❌ WRONG: Có thể bị mất context
return h.handleAgentOrchestrator(ctx, sc, msg.Text, msg.Chat.ID)
```

**Lưu ý khi implement:**

1. **Input không có prefix để trim:**

```go
// Với /ask: query = strings.TrimPrefix(msg.Text, "/ask ")
// Với fallback: query = msg.Text (toàn bộ, không trim)

// Orchestrator phải hiểu cả 2 cases:
// - "/ask Lịch trình tuần này" → query = "Lịch trình tuần này"
// - "Bạn làm được gì?" → query = "Bạn làm được gì?" (full text)
```

2. **SystemPrompt phải linh hoạt:**

```go
systemPrompt := `Bạn là trợ lý thông minh.

Nếu user hỏi về chức năng của bạn, hãy giải thích:
- Tạo task tự động
- Tìm kiếm semantic
- Quản lý checklist
- Đồng bộ với Google Calendar

Nếu user hỏi về lịch trình/task, hãy dùng tools để trả lời.`
```

3. **Test cases quan trọng:**

```go
// Test 1: Conversational question
Input: "Bạn có thể làm gì?"
Expected: Agent giải thích chức năng (không gọi tools)

// Test 2: Task query
Input: "Tìm task về meeting"
Expected: Agent gọi search_tasks tool

// Test 3: Ambiguous input
Input: "Giúp tôi với"
Expected: Agent hỏi lại "Bạn cần giúp gì?"
```

---

## 🚀 Next Steps for Execution

1. **Week 1:** Implement Context Injection + Conversational Fallback
   - Sửa `orchestrator.go` và `handler.go`
   - Manual testing với các câu hỏi từ ảnh image_441c12.jpg
2. **Week 2:** Implement Session Memory
   - Thêm cache logic vào Orchestrator
   - Test multi-turn conversations
3. **Week 3:** Build E2E Test Suite
   - Viết tests cho tất cả Telegram commands
   - Mock LLM responses
4. **Week 4:** Increase Unit Test Coverage
   - Focus vào packages <50% coverage
   - Iterate until >80% total coverage

---

## 📚 References

- [Gemini API - System Instructions](https://ai.google.dev/docs/system_instructions)
- [Go Testing Best Practices](https://go.dev/doc/tutorial/add-a-test)
- [Testify Mock Library](https://github.com/stretchr/testify)
- [Context-Aware Chatbots](https://arxiv.org/abs/2304.13007)
