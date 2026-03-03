# 🚀 MASTER PLAN: VERSION 1.2 - TESTING & OPTIMIZATION

## TỔNG QUAN

Version 1.2 tập trung vào **Testing Infrastructure** và **Performance Optimization** cho hệ thống ATM đã hoạt động ổn định từ Version 1.1.

**Điểm khác biệt:** Đây KHÔNG phải là version xây dựng tính năng mới, mà là version **củng cố nền móng** thông qua test coverage và tối ưu hóa hiệu năng.

---

## 1. HIỆN TRẠNG HỆ THỐNG (CURRENT STATUS)

### ✅ Đã Hoàn Thành (V1.0 - V1.1)

**Kiến trúc Agent:**
- ✅ ReAct Loop với max 5 steps (chống infinite loop)
- ✅ Tool Registry với function calling
- ✅ Multi-provider LLM Manager (DeepSeek → Gemini → Qwen)
- ✅ Semantic Router phân loại 4 intents
- ✅ Time context injection tự động

**Memory Management:**
- ✅ Session memory với TTL-based cleanup
- ✅ Background goroutine dọn rác mỗi 5 phút
- ✅ Sliding window (10 messages = 5 turns)
- ✅ Mutex-protected concurrent access
- ✅ TTL: 10 phút (configurable)

**Infrastructure:**
- ✅ Docker Compose với 4 services (memos, qdrant, backend, ngrok)
- ✅ Health check endpoints
- ✅ Test endpoints (`/test/message`, `/test/reset`)
- ✅ Swagger documentation

### ⚠️ Gaps & Rủi Ro

**Rủi ro 1 - Test Coverage Thấp:**
- Chỉ có 15 test files, chủ yếu là unit tests
- KHÔNG có E2E tests cho Telegram webhook flow
- KHÔNG có tests cho `ProcessQuery()` agent logic
- KHÔNG có tests cho router classification
- Coverage hiện tại: CHƯA ĐO (unknown)

**Rủi ro 2 - Performance Chưa Tối Ưu:**
- Session cleanup dùng manual map scan (có thể chậm với >1000 users)
- Có thể tối ưu bằng `expirable.LRU` (đã có trong dependencies)
- Chưa có metrics/monitoring cho memory usage

**Rủi ro 3 - Thiếu Integration Tests:**
- Các component test riêng lẻ nhưng chưa test tích hợp
- Chưa test full flow: Telegram → Router → Agent → Tools → Response

---

## 2. MỤC TIÊU VERSION 1.2

### 🎯 Objective 1: Test Coverage ≥ 80%

**Ưu tiên cao:**
1. E2E tests cho Telegram webhook handler
2. Unit tests cho Agent `ProcessQuery()` logic
3. Unit tests cho Router classification
4. Integration tests cho full message flow

**Ưu tiên trung bình:**
5. Tests cho session memory cleanup
6. Tests cho tool execution
7. Tests cho time context injection

### 🎯 Objective 2: Performance Optimization (Optional)

**Đánh giá:**
- Session cleanup hiện tại hoạt động tốt với <1000 users
- Có thể migrate sang `expirable.LRU` nếu cần scale lớn hơn

**Quyết định:** Giữ nguyên implementation hiện tại, chỉ refactor nếu có performance issues thực tế.

### 🎯 Objective 3: Monitoring & Metrics

**Thêm:**
- Coverage report automation
- Session memory metrics logging
- Performance benchmarks

---

## 3. CHIẾN LƯỢC THỰC HIỆN

### Phase 1: E2E Testing Infrastructure (Ưu tiên cao nhất)

**Mục tiêu:** Test full flow từ Telegram webhook đến response

**Scope:**
- Test router classification với các message types
- Test agent ReAct loop với mock tools
- Test session memory persistence
- Test error handling

### Phase 2: Unit Test Coverage

**Mục tiêu:** Đạt ≥80% coverage cho core components

**Scope:**
- `internal/agent/usecase/`: ProcessQuery, session management
- `internal/router/usecase/`: Classify logic
- `internal/checklist/usecase/`: Checkbox operations
- `internal/task/usecase/`: Task operations

### Phase 3: Integration Tests

**Mục tiêu:** Test tích hợp giữa các components

**Scope:**
- Router → Agent integration
- Agent → Tools integration
- Full message flow with real dependencies

### Phase 4: Performance Optimization (Nếu cần)

**Điều kiện trigger:**
- Session cleanup time > 100ms
- Memory usage > 500MB với <1000 users
- Có performance complaints từ users

**Solution:** Migrate sang `expirable.LRU`

---

## 4. IMPLEMENTATION DETAILS

### 4.1. Hiện Trạng Session Memory (ĐÃ IMPLEMENT)

**Location:** `internal/agent/usecase/`

**Current Implementation:**
```go
type implUseCase struct {
    sessionCache map[string]*agent.SessionMemory
    cacheMutex   sync.RWMutex
    cacheTTL     time.Duration // 10 minutes
}
```

**Features:**
- ✅ Background cleanup goroutine (mỗi 5 phút)
- ✅ TTL-based expiration (10 phút)
- ✅ Mutex-protected concurrent access
- ✅ Sliding window (10 messages max)

**Không cần thay đổi** - đã hoạt động tốt!

### 4.2. Hiện Trạng Sliding Window (ĐÃ IMPLEMENT)

**Location:** `internal/agent/usecase/process_query.go`

**Current Implementation:**
```go
// Limit history to last N messages
if len(session.Messages) > MaxSessionHistory {
    session.Messages = session.Messages[len(session.Messages)-MaxSessionHistory:]
}
```

**Configuration:**
- `MaxSessionHistory = 10` (5 turns)
- Applied after each agent response

**Không cần thay đổi** - đã hoạt động đúng!

### 4.3. Test Infrastructure Cần Xây Dựng (TODO)

**Thiếu:**
1. E2E tests cho webhook flow
2. Unit tests cho agent logic
3. Integration tests
4. Coverage reporting

**Sẽ implement trong Code Plan**

---

## 5. MILESTONES NGHIỆM THU

### Milestone 1: E2E Test Suite ✅
**Tiêu chí:**
- ✅ Test router với 4 intents (CREATE_TASK, SEARCH_TASK, MANAGE_CHECKLIST, CONVERSATION)
- ✅ Test agent với mock tools
- ✅ Test session persistence
- ✅ Test error scenarios

**Cách verify:**
```bash
go test -v ./internal/task/delivery/telegram/... -run TestWebhook
```

### Milestone 2: Coverage ≥ 80% ✅
**Tiêu chí:**
- ✅ Core packages có coverage ≥ 80%
- ✅ Critical paths được test đầy đủ

**Cách verify:**
```bash
go test -coverprofile=coverage.out ./internal/... ./pkg/...
go tool cover -func=coverage.out | grep total
```

### Milestone 3: CI/CD Integration ✅
**Tiêu chí:**
- ✅ Tests chạy tự động trên mỗi commit
- ✅ Coverage report được generate
- ✅ Fail build nếu coverage < 80%

---

## 6. NEXT STEPS

1. **Đọc Code Plan chi tiết** → Xem implementation cụ thể
2. **Bắt đầu với E2E tests** → Ưu tiên cao nhất
3. **Tăng dần coverage** → Từng package một
4. **Monitor performance** → Quyết định có cần optimize không

**Lưu ý quan trọng:** Session memory và sliding window ĐÃ HOẠT ĐỘNG TỐT. Version 1.2 tập trung vào TESTING, không phải refactor memory management!



