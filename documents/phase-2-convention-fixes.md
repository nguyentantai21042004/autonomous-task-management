# Phase 2 Convention Compliance Review

## ✅ Verified Against Conventions

Phase 2 plan đã được review theo:
- `documents/convention/convention.md`
- `documents/convention/convention_delivery.md`
- `documents/convention/convention_repository.md`
- `documents/convention/convention_usecase.md`

---

## 🔧 Required Fixes

### 1. Telegram Delivery Structure

**Current (WRONG):**
```
internal/task/delivery/telegram/
├── handler.go
├── process_request.go
├── presenters.go
├── errors.go
└── new.go
```

**Fixed (CORRECT):**
```
internal/task/delivery/telegram/
├── new.go              # Handler interface + factory
├── handler.go          # HandleWebhook + processMessage
├── presenters.go       # Message DTOs (Update, Message)
└── errors.go           # Error mapping
```

**Reason:** Telegram is message-based (like Kafka/RabbitMQ), not HTTP. No need for `process_request.go` pattern.

---

### 2. UseCase Interface - Add models.Scope

**Current (WRONG):**
```go
type UseCase interface {
    CreateBulk(ctx context.Context, input CreateBulkInput) (CreateBulkOutput, error)
}
```

**Fixed (CORRECT):**
```go
type UseCase interface {
    CreateBulk(ctx context.Context, sc models.Scope, input CreateBulkInput) (CreateBulkOutput, error)
}
```

**Reason:** Convention mandates `context.Context` and `models.Scope` as first two parameters.

---

### 3. Repository Methods - Extract Scope from Context

**Current (WRONG):**
```go
func (r *implRepository) CreateTask(ctx context.Context, opt CreateTaskOptions) (model.Task, error)
```

**Fixed (CORRECT):**
```go
func (r *implRepository) CreateTask(ctx context.Context, opt CreateTaskOptions) (model.Task, error) {
    // Extract scope from context if needed
    // sc := scope.GetScopeFromContext(ctx)
    // Use sc for filtering/logging
}
```

**Reason:** Scope should be in context, not passed as parameter. Repository extracts it when needed.

---

### 4. UseCase File Structure

**Current (WRONG):**
```
usecase/
├── new.go
├── create_bulk.go
├── parse_input.go      ← Helper, should be in helpers.go
├── helpers.go
└── types.go
```

**Fixed (CORRECT):**
```
usecase/
├── new.go              # Factory only
├── create_bulk.go      # CreateBulk() method
├── helpers.go          # ALL helpers (parseInputWithLLM, buildTaskMarkdown, etc.)
└── types.go            # Private types (taskWithDate)
```

**Reason:** Convention: One file per public method, ALL helpers in `helpers.go`.

---

### 5. Telegram Handler - Simplified Pattern

**handler.go:**
```go
package telegram

import (
    "context"
    "fmt"
    
    "github.com/gin-gonic/gin"
    "github.com/yourusername/autonomous-task-management/internal/model"
    "github.com/yourusername/autonomous-task-management/internal/task"
    pkgLog "github.com/yourusername/autonomous-task-management/pkg/log"
    pkgResponse "github.com/yourusername/autonomous-task-management/pkg/response"
    pkgTelegram "github.com/yourusername/autonomous-task-management/pkg/telegram"
)

type handler struct {
    l   pkgLog.Logger
    uc  task.UseCase
    bot *pkgTelegram.Bot
}

func (h *handler) HandleWebhook(c *gin.Context) {
    ctx := c.Request.Context()
    
    var update pkgTelegram.Update
    if err := c.ShouldBindJSON(&update); err != nil {
        h.l.Errorf(ctx, "Failed to parse update: %v", err)
        pkgResponse.Error(c, err, nil)
        return
    }
    
    // Ignore non-message updates
    if update.Message == nil {
        pkgResponse.OK(c, map[string]string{"status": "ignored"})
        return
    }
    
    // Process message
    if err := h.processMessage(ctx, update.Message); err != nil {
        h.l.Errorf(ctx, "Failed to process message: %v", err)
        pkgResponse.OK(c, map[string]string{"status": "error"})
        return
    }
    
    pkgResponse.OK(c, map[string]string{"status": "ok"})
}

func (h *handler) processMessage(ctx context.Context, msg *pkgTelegram.Message) error {
    if msg.Text == "" {
        return nil
    }
    
    // Handle /start command
    if msg.Text == "/start" {
        return h.bot.SendMessage(msg.Chat.ID, "Welcome to Autonomous Task Management!")
    }
    
    // Build scope from Telegram user
    sc := model.Scope{
        UserID: fmt.Sprintf("telegram_%d", msg.From.ID),
    }
    
    // Handle bulk task creation
    input := task.CreateBulkInput{
        RawText:        msg.Text,
        TelegramChatID: msg.Chat.ID,
    }
    
    output, err := h.uc.CreateBulk(ctx, sc, input)
    if err != nil {
        h.l.Errorf(ctx, "CreateBulk failed: %v", err)
        h.bot.SendMessage(msg.Chat.ID, "Sorry, failed to process your request.")
        return err
    }
    
    // Send success message
    response := fmt.Sprintf("✅ Created %d tasks successfully!", output.TaskCount)
    return h.bot.SendMessage(msg.Chat.ID, response)
}
```

---

### 6. CreateBulkInput - Remove UserID

**Current (WRONG):**
```go
type CreateBulkInput struct {
    UserID         int64  // ← Should be in Scope
    RawText        string
    TelegramChatID int64
}
```

**Fixed (CORRECT):**
```go
type CreateBulkInput struct {
    RawText        string
    TelegramChatID int64
}
```

**Reason:** UserID should be in `models.Scope`, not in Input.

---

### 7. UseCase Implementation - Use Scope

**create_bulk.go:**
```go
func (uc *implUseCase) CreateBulk(ctx context.Context, sc models.Scope, input task.CreateBulkInput) (task.CreateBulkOutput, error) {
    uc.l.Infof(ctx, "CreateBulk: Processing input from user %s", sc.UserID)
    
    // ... rest of implementation
}
```

---

## 📋 Updated Checklist

### Convention Compliance
- [ ] Telegram delivery follows message-based pattern (not HTTP)
- [ ] UseCase methods have `(ctx context.Context, sc models.Scope, input Input)` signature
- [ ] Repository extracts scope from context when needed
- [ ] UseCase structure: one file per method + helpers.go
- [ ] No UserID in Input structs (use Scope)
- [ ] All types in `types.go` (module root) or `usecase/types.go` (private)
- [ ] Factory in `new.go` contains ONLY struct + New() + setters

### Phase 2 Implementation
- [ ] `pkg/telegram` - Bot client
- [ ] `pkg/gemini` - LLM client
- [ ] `pkg/datemath` - Date parser
- [ ] `pkg/gcalendar` - Calendar client
- [ ] `internal/task/interface.go` - UseCase interface (with Scope)
- [ ] `internal/task/types.go` - Input/Output structs
- [ ] `internal/task/errors.go` - Domain errors
- [ ] `internal/task/repository/interface.go` - Repository interfaces
- [ ] `internal/task/repository/option.go` - Options structs
- [ ] `internal/task/repository/memos/` - Memos implementation
- [ ] `internal/task/usecase/new.go` - Factory
- [ ] `internal/task/usecase/create_bulk.go` - Main logic
- [ ] `internal/task/usecase/helpers.go` - ALL helpers
- [ ] `internal/task/usecase/types.go` - Private types
- [ ] `internal/task/delivery/telegram/` - Telegram handler
- [ ] `cmd/api/main.go` - Wiring

---

## 🎯 Key Takeaways

1. **Telegram ≠ HTTP**: Don't use HTTP delivery pattern for message-based systems
2. **Scope is mandatory**: Always pass `models.Scope` to UseCase methods
3. **One file per method**: Public methods get their own file, helpers go to `helpers.go`
4. **Types centralization**: Public types in module root, private in `usecase/types.go`
5. **Factory purity**: `new.go` contains ONLY struct + factory + setters

---

## ✅ Compliance Status

After applying these fixes, Phase 2 plan will be **100% compliant** with workspace conventions.

**Main plan file:** `documents/phase-2-implementation-plan.md` (already contains all code, just needs these structural fixes applied)

---

## 🚨 CRITICAL RUNTIME ISSUES (From Expert Review)

### Issue 1: Telegram Webhook Timeout Risk ⚠️ CRITICAL

**Vấn đề:** Trong `handler.go`, hàm `processMessage()` được gọi đồng bộ (synchronous) trước khi trả về HTTP 200 OK cho Telegram. Luồng xử lý bao gồm:
- Gọi Gemini API (2-5 giây)
- Vòng lặp tạo nhiều Memos (1+ giây)
- Gọi Google Calendar API (2+ giây)

Tổng thời gian có thể vượt quá timeout của Telegram webhook (vài giây), dẫn đến:
- Telegram tưởng bot chết và retry gửi lại message
- Tạo task trùng lặp (Duplicate Tasks)
- User experience kém

**Giải pháp:** Đẩy `processMessage()` vào Goroutine chạy ngầm (Background Job), trả về 200 OK ngay lập tức.

**Code fix trong `handler.go`:**

```go
func (h *handler) HandleWebhook(c *gin.Context) {
    ctx := c.Request.Context()
    
    var update pkgTelegram.Update
    if err := c.ShouldBindJSON(&update); err != nil {
        h.l.Errorf(ctx, "Failed to parse update: %v", err)
        pkgResponse.Error(c, err, nil)
        return
    }
    
    // Ignore non-message updates
    if update.Message == nil {
        pkgResponse.OK(c, map[string]string{"status": "ignored"})
        return
    }
    
    // ✅ FIX: Process message in background goroutine
    go func(msg *pkgTelegram.Message) {
        // Tạo context mới detached khỏi HTTP Request
        bgCtx := context.Background()
        
        if err := h.processMessage(bgCtx, msg); err != nil {
            h.l.Errorf(bgCtx, "Background process failed: %v", err)
            // Optionally: Send error message to user
            h.bot.SendMessage(msg.Chat.ID, "❌ Failed to process your request. Please try again.")
        }
    }(update.Message)
    
    // ✅ Trả về ngay lập tức
    pkgResponse.OK(c, map[string]string{"status": "accepted"})
}
```

**Lưu ý quan trọng:**
- Phải tạo `context.Background()` mới vì HTTP request context sẽ bị cancel sau khi response
- Cần có error handling trong goroutine để thông báo user nếu thất bại
- Consider thêm queue system (Redis/RabbitMQ) trong Phase 3 cho production

---

### Issue 2: Gemini JSON Parse Error ⚠️ HIGH

**Vấn đề:** Trong `usecase/create_bulk.go`, hàm `parseInputWithLLM()` lấy response từ Gemini và đập thẳng vào `json.Unmarshal()`:

```go
responseText := resp.Candidates[0].Content.Parts[0].Text
var tasks []gemini.ParsedTask
if err := json.Unmarshal([]byte(responseText), &tasks); err != nil {
    return nil, fmt.Errorf("failed to parse LLM response: %w", err)
}
```

**Vấn đề:** LLM models (bao gồm Gemini) thường bọc JSON trong markdown code blocks:

```
```json
[{"title": "Task 1", ...}]
```
```

Hoặc thêm text giải thích trước/sau JSON. Điều này khiến `json.Unmarshal()` fail.

**Giải pháp:** Thêm bước sanitize (làm sạch) response trước khi parse.

**Code fix trong `usecase/helpers.go`:**

```go
import (
    "regexp"
    "strings"
)

func (uc *implUseCase) parseInputWithLLM(ctx context.Context, rawText string) ([]gemini.ParsedTask, error) {
    prompt := gemini.BuildTaskParsingPrompt(rawText)

    req := gemini.GenerateRequest{
        Contents: []gemini.Content{
            {
                Parts: []gemini.Part{
                    {Text: prompt},
                },
            },
        },
    }

    resp, err := uc.llm.GenerateContent(ctx, req)
    if err != nil {
        return nil, err
    }

    if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
        return nil, fmt.Errorf("empty response from LLM")
    }

    responseText := resp.Candidates[0].Content.Parts[0].Text
    
    // ✅ FIX: Sanitize response before parsing
    cleanedJSON := sanitizeJSONResponse(responseText)

    var tasks []gemini.ParsedTask
    if err := json.Unmarshal([]byte(cleanedJSON), &tasks); err != nil {
        uc.l.Errorf(ctx, "Failed to parse LLM response. Raw: %s, Cleaned: %s", responseText, cleanedJSON)
        return nil, fmt.Errorf("failed to parse LLM response: %w", err)
    }

    return tasks, nil
}

// sanitizeJSONResponse removes markdown code blocks and extra text
func sanitizeJSONResponse(text string) string {
    // Remove markdown code blocks: ```json ... ``` or ``` ... ```
    re := regexp.MustCompile("(?s)```(?:json)?\\s*(.+?)\\s*```")
    matches := re.FindStringSubmatch(text)
    if len(matches) > 1 {
        return strings.TrimSpace(matches[1])
    }
    
    // If no code blocks, try to extract JSON array/object
    // Find first [ or { and last ] or }
    start := strings.IndexAny(text, "[{")
    if start == -1 {
        return text
    }
    
    end := strings.LastIndexAny(text, "]}")
    if end == -1 || end < start {
        return text
    }
    
    return strings.TrimSpace(text[start : end+1])
}
```

**Testing:**

```go
// Test cases
testCases := []string{
    // Case 1: Clean JSON
    `[{"title": "Task 1"}]`,
    
    // Case 2: Markdown wrapped
    "```json\n[{\"title\": \"Task 1\"}]\n```",
    
    // Case 3: With explanation
    "Here are the tasks:\n```json\n[{\"title\": \"Task 1\"}]\n```\nHope this helps!",
    
    // Case 4: No markdown
    "Sure! [{'title': 'Task 1'}]",
}
```

---

### Issue 3: Timezone Conflict in Calendar API ⚠️ MEDIUM

**Vấn đề:** Trong `pkg/datemath/parser.go`, hàm `startOfDay()` trả về `time.Time` đã gắn timezone (VD: Asia/Ho_Chi_Minh). Nhưng trong `pkg/gcalendar/client.go`, khi tạo event:

```go
Start: &calendar.EventDateTime{
    DateTime: req.StartTime.Format("2006-01-02T15:04:05Z07:00"),
    TimeZone: req.Timezone,
}
```

**Vấn đề:** Format string `Z07:00` sẽ format theo timezone của `time.Time` object, nhưng lại truyền thêm `TimeZone` field riêng. Điều này có thể gây conflict hoặc sai lệch giờ.

**Giải pháp:** Sử dụng `time.RFC3339` format (chuẩn ISO 8601) mà Google Calendar API yêu thích.

**Code fix trong `pkg/gcalendar/client.go`:**

```go
func (c *Client) CreateEvent(ctx context.Context, req CreateEventRequest) (*Event, error) {
    event := &calendar.Event{
        Summary:     req.Summary,
        Description: req.Description,
        Start: &calendar.EventDateTime{
            // ✅ FIX: Use RFC3339 format (includes timezone info)
            DateTime: req.StartTime.Format(time.RFC3339),
            TimeZone: req.Timezone,
        },
        End: &calendar.EventDateTime{
            // ✅ FIX: Use RFC3339 format
            DateTime: req.EndTime.Format(time.RFC3339),
            TimeZone: req.Timezone,
        },
    }

    createdEvent, err := c.service.Events.Insert(req.CalendarID, event).Context(ctx).Do()
    if err != nil {
        return nil, fmt.Errorf("failed to create event: %w", err)
    }

    return &Event{
        ID:          createdEvent.Id,
        Summary:     createdEvent.Summary,
        Description: createdEvent.Description,
        HtmlLink:    createdEvent.HtmlLink,
    }, nil
}
```

**Hoặc đơn giản hơn:** Chỉ dùng DateTime với RFC3339, bỏ TimeZone field (vì RFC3339 đã chứa timezone):

```go
Start: &calendar.EventDateTime{
    DateTime: req.StartTime.Format(time.RFC3339),
    // TimeZone field is optional when DateTime includes timezone
}
```

**Testing:**

```go
// Verify timezone handling
loc, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
testTime := time.Date(2024, 3, 15, 14, 30, 0, 0, loc)

// Should output: 2024-03-15T14:30:00+07:00
fmt.Println(testTime.Format(time.RFC3339))
```

---

## 📋 Updated Implementation Checklist

### Critical Fixes (Must implement before testing)

- [ ] **Telegram Handler:** Implement background goroutine for `processMessage()`
- [ ] **LLM Parser:** Add `sanitizeJSONResponse()` helper function
- [ ] **Calendar Client:** Use `time.RFC3339` format for DateTime

### Additional Improvements (Recommended)

- [ ] Add retry logic for Gemini API calls (exponential backoff)
- [ ] Add timeout for background goroutine (context.WithTimeout)
- [ ] Add metrics/monitoring for goroutine execution time
- [ ] Add structured logging for LLM raw responses (debugging)
- [ ] Add validation for parsed tasks (check required fields)

### Testing Priorities

1. **Test Telegram webhook timeout:** Send complex input, verify 200 OK returned immediately
2. **Test LLM response parsing:** Mock various response formats (with/without markdown)
3. **Test timezone handling:** Create events, verify correct time in Google Calendar
4. **Test error scenarios:** Network failures, API rate limits, invalid inputs

---

## 🎯 Implementation Priority

**Phase 2A (Critical - Must have):**
1. ✅ Telegram background processing (Issue 1)
2. ✅ JSON sanitization (Issue 2)
3. ✅ RFC3339 timezone format (Issue 3)

**Phase 2B (Important - Should have):**
4. Retry logic for API calls
5. Structured error handling
6. Comprehensive logging

**Phase 2C (Nice to have):**
7. Metrics and monitoring
8. Performance optimization
9. Advanced error recovery

---

## 💡 Expert Recommendations Summary

1. **Telegram Webhook:** NEVER block HTTP response với long-running operations. Always use background jobs.

2. **LLM Integration:** ALWAYS sanitize LLM responses. Never trust raw output format.

3. **Timezone Handling:** Use standard formats (RFC3339) để tránh ambiguity và bugs.

4. **Error Handling:** Implement graceful degradation - nếu Calendar API fail, vẫn tạo được Memos.

5. **Logging:** Log raw LLM responses và intermediate states để debug dễ dàng.

6. **Testing:** Test với real-world scenarios: slow networks, malformed inputs, API failures.

---

## 🔗 Related Documentation

- [Telegram Bot Best Practices](https://core.telegram.org/bots/webhooks)
- [Go Concurrency Patterns](https://go.dev/blog/pipelines)
- [RFC3339 DateTime Format](https://datatracker.ietf.org/doc/html/rfc3339)
- [Google Calendar API DateTime](https://developers.google.com/calendar/api/v3/reference/events#resource)
