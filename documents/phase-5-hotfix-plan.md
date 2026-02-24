# Phase 5 Hotfix Plan - Production Bug Fixes

**Ngày tạo:** 24/02/2026  
**Mức độ ưu tiên:** 🔥 CRITICAL  
**Thời gian ước tính:** 2-3 giờ

---

## 📋 Tổng quan

Sau khi phân tích log thực tế (`real-chat.log` và `system.log`), phát hiện 4 lỗi nghiêm trọng cần fix ngay:

1. ❌ **Conversational Fallback bị tê liệt** - Logic check error không chính xác
2. ❌ **LLM Temporal Blindness** - Agent không hiểu thời gian tương đối
3. ❌ **Markdown Parsing Crash** - Telegram API từ chối tin nhắn
4. ❌ **Data Drift (Qdrant vs Memos)** - Vector rác gây kết quả sai

---

## 🎯 Mục tiêu

- [x] User có thể chat tự nhiên không cần lệnh `/ask`
- [x] Agent tự động hiểu "tuần này", "ngày mai" mà không hỏi ngược
- [x] Bot không bao giờ crash do lỗi Markdown
- [x] Search luôn trả về kết quả chính xác (tự động xóa vector rác)

---

## 🔧 HOTFIX 1: Conversational Fallback Logic

### Vấn đề
```log
# system.log 10:03:35.516
ERROR telegram handler: CreateBulk failed: no tasks parsed from input

# real-chat.log 17:03:28
User: "trong tuần này"
Bot: "Không thể xử lý yêu cầu: no tasks parsed from input"
```

**Root cause:** `errors.Is(err, task.ErrNoTasksParsed)` trả về `false` vì lỗi bị wrapped ở đâu đó.

### Files cần sửa

#### 1. `internal/task/delivery/telegram/handler.go`

**Vị trí:** Hàm `handleCreateTask()` line ~125-135

**Thay đổi:**
```go
// ❌ BEFORE (chỉ check exact match)
if errors.Is(err, task.ErrNoTasksParsed) {
    h.l.Infof(ctx, "No tasks parsed, falling back to conversational agent for text: %s", msg.Text)
    return h.handleAgentOrchestrator(ctx, sc, msg.Text, msg.Chat.ID)
}

// ✅ AFTER (check cả string contains)
if errors.Is(err, task.ErrNoTasksParsed) || strings.Contains(err.Error(), "no tasks parsed") {
    h.l.Infof(ctx, "No tasks parsed, falling back to conversational agent for text: %s", msg.Text)
    return h.handleAgentOrchestrator(ctx, sc, msg.Text, msg.Chat.ID)
}
```

#### 2. `internal/task/usecase/create_bulk.go`

**Kiểm tra:** Đảm bảo khi LLM trả về `[]`, phải return đúng `task.ErrNoTasksParsed` (không wrap thêm)

**Vị trí:** Hàm `parseInputWithLLM()` hoặc `CreateBulk()`

**Cần verify:**
```go
// ✅ CORRECT
if len(tasks) == 0 {
    return task.ErrNoTasksParsed
}

// ❌ WRONG (sẽ làm errors.Is fail)
if len(tasks) == 0 {
    return fmt.Errorf("failed to parse: %w", task.ErrNoTasksParsed)
}
```

### Test cases

```bash
# Test 1: Conversational question
Input: "Bạn có thể làm gì?"
Expected: Agent trả lời chức năng (không báo lỗi)

# Test 2: Ambiguous input
Input: "Giúp tôi với"
Expected: Agent hỏi lại hoặc giải thích

# Test 3: Time query without /ask
Input: "trong tuần này"
Expected: Fallback sang Agent (không báo lỗi)
```

---

## 🔧 HOTFIX 2: LLM Temporal Blindness

### Vấn đề
```log
# real-chat.log 17:03:20
User: /ask kiểm tra lịch trong tuần này
Bot: Bạn muốn kiểm tra lịch từ ngày nào đến ngày nào vậy?
```

**Root cause:** LLM ignore SystemInstruction, không tự tính toán ngày từ "tuần này".

### Files cần sửa

#### 1. `internal/agent/orchestrator/orchestrator.go`

**Vị trí:** Hàm `ProcessQuery()` line ~70-120

**Thay đổi:** Inject temporal context vào CUỐI user query (không dựa vào SystemInstruction)

```go
func (o *Orchestrator) ProcessQuery(ctx context.Context, userID string, query string) (string, error) {
    loc, err := time.LoadLocation(o.timezone)
    if err != nil {
        loc = time.UTC
    }
    currentTime := time.Now().In(loc)
    
    // ✅ NEW: Calculate week boundaries
    weekday := int(currentTime.Weekday())
    if weekday == 0 { // Sunday
        weekday = 7
    }
    weekStart := currentTime.AddDate(0, 0, -(weekday - 1)) // Monday
    weekEnd := weekStart.AddDate(0, 0, 6)                  // Sunday
    
    // ✅ NEW: Hard inject vào cuối user query
    timeContext := fmt.Sprintf(
        "\n\n[SYSTEM CONTEXT - Thông tin thời gian hiện tại:"+
        "\n- Hôm nay: %s (%s)"+
        "\n- Tuần này: từ %s đến %s"+
        "\n- Ngày mai: %s"+
        "\n\nQUY TẮC QUAN TRỌNG:"+
        "\n1. Nếu user hỏi về 'tuần này', hãy TỰ ĐỘNG gọi tool với start_date='%s' và end_date='%s'"+
        "\n2. Nếu user hỏi về 'ngày mai', dùng date='%s'"+
        "\n3. KHÔNG BAO GIỜ hỏi ngược lại user về ngày tháng cụ thể"+
        "\n4. Format ngày LUÔN LUÔN là YYYY-MM-DD]",
        currentTime.Format("2006-01-02"),
        currentTime.Weekday().String(),
        weekStart.Format("2006-01-02"),
        weekEnd.Format("2006-01-02"),
        currentTime.AddDate(0, 0, 1).Format("2006-01-02"),
        weekStart.Format("2006-01-02"),
        weekEnd.Format("2006-01-02"),
        currentTime.AddDate(0, 0, 1).Format("2006-01-02"),
    )
    
    // ✅ Nối vào query
    enhancedQuery := query + timeContext
    
    session := o.getSession(userID)
    
    // Build current user message với enhanced query
    userMessage := gemini.Content{Role: "user", Parts: []gemini.Part{{Text: enhancedQuery}}}
    
    // ... rest of existing code (không thay đổi)
}
```

#### 2. `internal/agent/tools/check_calendar.go`

**Kiểm tra:** Đảm bảo tool schema yêu cầu `start_date` và `end_date` là required

**Verify:**
```go
// Tool definition phải có:
{
    "name": "check_calendar",
    "parameters": {
        "type": "object",
        "properties": {
            "start_date": {"type": "string", "description": "Format: YYYY-MM-DD"},
            "end_date": {"type": "string", "description": "Format: YYYY-MM-DD"}
        },
        "required": ["start_date", "end_date"]
    }
}
```

### Test cases

```bash
# Test 1: "tuần này"
Input: /ask lịch trình tuần này
Expected: Agent gọi check_calendar(start_date="2026-02-24", end_date="2026-03-02")

# Test 2: "ngày mai"
Input: /ask tôi có meeting nào ngày mai?
Expected: Agent gọi check_calendar với ngày 25/02/2026

# Test 3: "tháng này"
Input: /ask deadline tháng này
Expected: Agent tự tính từ 01/02 đến 28/02
```

---

## 🔧 HOTFIX 3: Markdown Parsing Crash

### Vấn đề
```log
# system.log 10:02:12.878
ERROR telegram sendMessage API error 400: 
can't parse entities: Can't find end of the entity starting at byte offset 72
```

**Root cause:** LLM sinh ra Markdown không hợp lệ (unclosed `*`, `[`, `_`), Telegram từ chối.

### Files cần sửa

#### 1. `pkg/telegram/bot.go`

**Thêm hàm mới:** `sanitizeMarkdownV2()`

```go
package telegram

import (
    "regexp"
    "strings"
)

// sanitizeMarkdownV2 escapes special characters for Telegram MarkdownV2
// Reference: https://core.telegram.org/bots/api#markdownv2-style
func sanitizeMarkdownV2(text string) string {
    // Special characters that need escaping in MarkdownV2:
    // _ * [ ] ( ) ~ ` > # + - = | { } . !
    specialChars := []string{
        "_", "*", "[", "]", "(", ")", "~", "`", 
        ">", "#", "+", "-", "=", "|", "{", "}", ".", "!",
    }
    
    result := text
    for _, char := range specialChars {
        result = strings.ReplaceAll(result, char, "\\"+char)
    }
    
    return result
}

// removeInvalidMarkdown removes unclosed markdown tags
func removeInvalidMarkdown(text string) string {
    // Remove unclosed bold
    boldCount := strings.Count(text, "**")
    if boldCount%2 != 0 {
        text = strings.ReplaceAll(text, "**", "")
    }
    
    // Remove unclosed italic
    italicCount := strings.Count(text, "*")
    if italicCount%2 != 0 {
        text = strings.ReplaceAll(text, "*", "")
    }
    
    // Remove unclosed links [text](url)
    openBracket := strings.Count(text, "[")
    closeBracket := strings.Count(text, "]")
    if openBracket != closeBracket {
        text = regexp.MustCompile(`\[([^\]]*)\]\(([^\)]*)\)`).ReplaceAllString(text, "$1")
    }
    
    return text
}
```

#### 2. `pkg/telegram/bot.go` - Sửa hàm `SendMessageWithMode()`

**Vị trí:** Hàm `SendMessageWithMode()`

**Thay đổi:**
```go
func (b *Bot) SendMessageWithMode(chatID int64, text string, parseMode string) error {
    // ✅ NEW: Sanitize trước khi gửi
    if parseMode == "Markdown" || parseMode == "MarkdownV2" {
        text = removeInvalidMarkdown(text)
    }
    
    // Existing code
    payload := map[string]interface{}{
        "chat_id":    chatID,
        "text":       text,
        "parse_mode": parseMode,
    }
    
    // ... rest of code
}
```

#### 3. Alternative: Chuyển sang HTML mode (Safer)

**Option B:** Thay vì fix Markdown, chuyển toàn bộ sang HTML (ít lỗi hơn)

```go
// File: internal/task/delivery/telegram/handler.go
// Tìm tất cả chỗ gọi SendMessageWithMode(..., "Markdown")
// Thay bằng SendMessageWithMode(..., "HTML")

// Và convert markdown sang HTML:
func markdownToHTML(text string) string {
    // **bold** -> <b>bold</b>
    text = regexp.MustCompile(`\*\*([^*]+)\*\*`).ReplaceAllString(text, "<b>$1</b>")
    
    // *italic* -> <i>italic</i>
    text = regexp.MustCompile(`\*([^*]+)\*`).ReplaceAllString(text, "<i>$1</i>")
    
    // [text](url) -> <a href="url">text</a>
    text = regexp.MustCompile(`\[([^\]]+)\]\(([^\)]+)\)`).ReplaceAllString(text, `<a href="$2">$1</a>`)
    
    return text
}
```

### Test cases

```bash
# Test 1: Unclosed bold
LLM output: "Bạn có **3 meetings"
Expected: Bot gửi thành công (tự động fix)

# Test 2: Unclosed link
LLM output: "Xem [Memo"
Expected: Bot gửi "Xem Memo" (remove markdown)

# Test 3: Mixed invalid markdown
LLM output: "Task *abc với **deadline"
Expected: Bot gửi thành công (sanitize)
```

---

## 🔧 HOTFIX 4: Data Drift (Qdrant vs Memos)

### Vấn đề
```log
# system.log 10:01:28
INFO qdrant/task.go:139 qdrant repository: found 2 results
WARN usecase/search.go:56 failed to fetch task memos/g4tMughM3bDifMLaqqNWpj from Memos: 404
WARN usecase/search.go:56 failed to fetch task memos/BM2GDVTACCrEbnfxzJN75r from Memos: 404
INFO usecase/search.go:68 Search: found 0 results
```

**Root cause:** Task bị xóa ở Memos nhưng vector vẫn còn trong Qdrant (zombie vectors).

### Files cần sửa

#### 1. `internal/task/usecase/search.go`

**Vị trí:** Hàm `Search()` line ~40-60

**Thay đổi:** Thêm self-healing logic

```go
func (uc *implUseCase) Search(ctx context.Context, sc model.Scope, input task.SearchInput) (task.SearchOutput, error) {
    // ... existing code until fetching from Memos ...
    
    // Fetch full task details from Memos
    results := make([]task.SearchResultItem, 0, len(searchResults))
    zombieVectors := make([]string, 0) // ✅ NEW: Track zombie vectors
    
    for _, sr := range searchResults {
        // Fetch from Memos
        memoTask, err := uc.repo.GetTask(ctx, sr.MemoID)
        if err != nil {
            // ✅ NEW: Self-healing - xóa vector rác
            if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "Not Found") {
                uc.l.Warnf(ctx, "Search: Task %s deleted in Memos. Self-healing: removing from Qdrant", sr.MemoID)
                zombieVectors = append(zombieVectors, sr.MemoID)
                
                // Trigger async cleanup (không block search)
                go func(memoID string) {
                    cleanupCtx := context.Background()
                    if err := uc.vectorRepo.DeleteTask(cleanupCtx, memoID); err != nil {
                        uc.l.Errorf(cleanupCtx, "Failed to cleanup zombie vector %s: %v", memoID, err)
                    } else {
                        uc.l.Infof(cleanupCtx, "Successfully cleaned up zombie vector %s", memoID)
                    }
                }(sr.MemoID)
                
                continue
            }
            
            uc.l.Warnf(ctx, "Search: failed to fetch task %s from Memos: %v", sr.MemoID, err)
            continue
        }
        
        results = append(results, task.SearchResultItem{
            MemoID:  memoTask.ID,
            MemoURL: memoTask.MemoURL,
            Content: memoTask.Content,
            Score:   sr.Score,
        })
    }
    
    // ✅ NEW: Log self-healing stats
    if len(zombieVectors) > 0 {
        uc.l.Infof(ctx, "Search: Self-healing cleaned up %d zombie vectors: %v", len(zombieVectors), zombieVectors)
    }
    
    uc.l.Infof(ctx, "Search: found %d results (filtered from %d raw results)", len(results), len(searchResults))
    
    return task.SearchOutput{
        Results: results,
        Count:   len(results),
    }, nil
}
```

#### 2. `internal/task/repository/qdrant/task.go`

**Kiểm tra:** Đảm bảo có hàm `DeleteTask()`

**Verify:**
```go
// Must have this method
func (r *qdrantRepository) DeleteTask(ctx context.Context, memoID string) error {
    // Delete by payload filter (memoID)
    // Implementation should exist from Phase 3/4
}
```

#### 3. `internal/sync/handler.go` (Webhook sync)

**Kiểm tra:** Đảm bảo webhook `deleted` event xóa cả Qdrant

**Verify:**
```go
func (h *handler) HandleMemosWebhook(ctx context.Context, event MemosEvent) error {
    switch event.Type {
    case "deleted":
        // ✅ Must delete from Qdrant
        if err := h.vectorRepo.DeleteTask(ctx, event.MemoID); err != nil {
            h.l.Errorf(ctx, "Failed to delete from Qdrant: %v", err)
        }
    }
}
```

### Test cases

```bash
# Test 1: Search với zombie vectors
Setup: Xóa 2 tasks ở Memos nhưng giữ vectors trong Qdrant
Input: /search meeting
Expected: 
- Bot trả về kết quả hợp lệ (không có 404)
- Log ghi "Self-healing cleaned up 2 zombie vectors"

# Test 2: Verify cleanup
Setup: Sau test 1
Action: Search lại cùng query
Expected: Không còn warning 404 (vectors đã bị xóa)
```

---

## 📝 Implementation Checklist

### Phase 1: Hotfix Critical Bugs (1 giờ)

- [ ] **HOTFIX 1:** Conversational Fallback
  - [ ] Sửa `handler.go` - thêm `strings.Contains()` check
  - [ ] Verify `create_bulk.go` - đảm bảo return đúng error
  - [ ] Test: "Bạn làm được gì?" → Agent trả lời
  - [ ] Test: "trong tuần này" → Fallback sang Agent

- [ ] **HOTFIX 3:** Markdown Crash (Ưu tiên cao vì crash production)
  - [ ] Thêm `sanitizeMarkdownV2()` vào `pkg/telegram/bot.go`
  - [ ] Thêm `removeInvalidMarkdown()` vào `pkg/telegram/bot.go`
  - [ ] Sửa `SendMessageWithMode()` - gọi sanitize trước khi gửi
  - [ ] Test: Gửi tin nhắn có markdown lỗi → Không crash

### Phase 2: Improve Intelligence (1 giờ)

- [ ] **HOTFIX 2:** Temporal Blindness
  - [ ] Sửa `orchestrator.go` - inject time context vào user query
  - [ ] Tính toán week boundaries (Monday-Sunday)
  - [ ] Thêm examples vào prompt (few-shot)
  - [ ] Test: "/ask lịch tuần này" → Agent tự tính ngày
  - [ ] Test: "/ask meeting ngày mai" → Agent dùng đúng ngày

- [ ] **HOTFIX 4:** Data Drift Self-Healing
  - [ ] Sửa `search.go` - thêm zombie vector detection
  - [ ] Thêm async cleanup goroutine
  - [ ] Verify `DeleteTask()` method exists
  - [ ] Test: Search với zombie vectors → Tự động cleanup

### Phase 3: Testing & Verification (30 phút)

- [ ] **Manual Testing**
  - [ ] Test toàn bộ 12 test cases ở trên
  - [ ] Verify logs không còn ERROR
  - [ ] Check Telegram bot response time (<3s)

- [ ] **Regression Testing**
  - [ ] Test các lệnh cũ vẫn hoạt động:
    - [ ] `/search meeting`
    - [ ] `/progress abc123`
    - [ ] `/check abc123 item`
    - [ ] `/complete abc123`

- [ ] **Load Testing** (Optional)
  - [ ] Gửi 10 tin nhắn liên tiếp
  - [ ] Verify session memory hoạt động
  - [ ] Check memory không leak

### Phase 4: Documentation (30 phút)

- [ ] Update `README.md` - ghi chú về các fix
- [ ] Update `documents/phase-5-verification-plan.md` - đánh dấu hoàn thành
- [ ] Tạo `CHANGELOG.md` - ghi lại các thay đổi
- [ ] Commit với message: `hotfix: Phase 5 production bugs (conversational fallback, temporal context, markdown crash, data drift)`

---

## 🚀 Deployment Plan

### Pre-deployment

```bash
# 1. Backup current state
docker-compose exec memos /bin/sh -c "memos backup"

# 2. Run tests
go test ./internal/task/delivery/telegram/... -v
go test ./internal/agent/orchestrator/... -v
go test ./pkg/telegram/... -v

# 3. Build
make build
```

### Deployment

```bash
# 1. Stop services
make down

# 2. Pull latest code
git pull origin main

# 3. Restart
make up

# 4. Watch logs
make logs
```

### Post-deployment Verification

```bash
# 1. Health check
curl http://localhost:8080/health

# 2. Test bot
# Gửi tin nhắn Telegram: "Bạn làm được gì?"
# Expected: Agent trả lời (không lỗi)

# 3. Monitor logs
tail -f system.log | grep ERROR
# Expected: Không có ERROR mới
```

---

## 🎯 Success Criteria

### Functional Requirements

- [x] User có thể chat tự nhiên mà không cần `/ask`
- [x] Agent hiểu "tuần này", "ngày mai" mà không hỏi ngược
- [x] Bot không bao giờ crash do Markdown lỗi
- [x] Search luôn trả về kết quả chính xác (tự động cleanup)

### Performance Requirements

- [x] Response time <3s cho conversational queries
- [x] Self-healing cleanup <100ms (async, không block)
- [x] Session memory không leak (cleanup mỗi 5 phút)

### Quality Requirements

- [x] Zero ERROR logs trong 1 giờ production
- [x] Zero crash do Telegram API 400
- [x] Zero 404 warnings từ zombie vectors (sau lần đầu cleanup)

---

## 📊 Monitoring & Alerts

### Metrics to Track

```bash
# 1. Error rate
grep "ERROR" system.log | wc -l
# Target: 0 errors/hour

# 2. Fallback rate
grep "falling back to conversational agent" system.log | wc -l
# Target: >0 (nghĩa là fallback hoạt động)

# 3. Zombie vector cleanup
grep "Self-healing cleaned up" system.log | wc -l
# Target: Giảm dần về 0 (sau khi cleanup hết)

# 4. Telegram API errors
grep "telegram sendMessage API error 400" system.log | wc -l
# Target: 0 errors
```

### Alert Rules

```yaml
# Nếu dùng monitoring tool (Prometheus, Grafana)
alerts:
  - name: TelegramAPIError
    condition: telegram_api_errors > 0
    severity: critical
    
  - name: HighErrorRate
    condition: error_rate > 5/hour
    severity: warning
    
  - name: ZombieVectorSpike
    condition: zombie_vectors > 10
    severity: info
```

---

## 🔍 Rollback Plan

Nếu có vấn đề sau khi deploy:

```bash
# 1. Revert code
git revert HEAD
git push origin main

# 2. Redeploy
make down
make up

# 3. Restore backup (nếu cần)
docker-compose exec memos /bin/sh -c "memos restore /backup/latest.db"
```

---

## 📚 References

- [Telegram Bot API - MarkdownV2](https://core.telegram.org/bots/api#markdownv2-style)
- [Gemini API - System Instructions](https://ai.google.dev/docs/system_instructions)
- [Go Error Handling Best Practices](https://go.dev/blog/error-handling-and-go)
- [Qdrant Delete Operations](https://qdrant.tech/documentation/concepts/points/#delete-points)

---

## 💡 Lessons Learned

1. **SystemInstruction không đáng tin cậy** - Luôn inject critical context vào user message
2. **Error wrapping phá vỡ errors.Is()** - Cần check cả string contains
3. **Telegram MarkdownV2 cực kỳ khắt khe** - Nên dùng HTML hoặc sanitize kỹ
4. **Vector DB cần self-healing** - Không thể tin tưởng 100% vào webhook sync

---

**Người thực hiện:** [Your Name]  
**Reviewer:** [Reviewer Name]  
**Ngày hoàn thành dự kiến:** 24/02/2026 EOD
