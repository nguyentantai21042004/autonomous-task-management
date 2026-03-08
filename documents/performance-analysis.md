# Performance & Quality Analysis Report

> Cập nhật: Tháng 3/2026

## Tổng quan

Báo cáo phân tích hiệu suất và chất lượng phản hồi của bot. Phần lớn các bottleneck nghiêm trọng **đã được fix**. Các mục còn lại là cải tiến tiềm năng cho tương lai.

---

## 1. Các vấn đề đã FIX

### 1.1 ✅ N+1 Query Pattern → Parallel Fetch

**File:** `internal/task/usecase/answer_query.go`, `internal/task/usecase/search.go`

**Trước:** Mỗi task fetch tuần tự từ Memos API (5 task × ~80ms = ~400ms).

**Sau:** Dùng `sync.WaitGroup` để fetch song song. Tất cả tasks fetch đồng thời.

```go
var wg sync.WaitGroup
for i, sr := range searchResults {
    wg.Add(1)
    go func(idx int, memoID string) {
        defer wg.Done()
        t, err := uc.repo.GetTask(ctx, memoID)
        fetched[idx] = fetchResult{task: t, err: err}
    }(i, sr.MemoID)
}
wg.Wait()
```

**Tiết kiệm: ~300ms mỗi request.** Zombie vector self-healing vẫn hoạt động.

### 1.2 ✅ HTTP Timeout cho Qdrant & Voyage

**File:** `pkg/qdrant/client.go`, `pkg/voyage/client.go`

**Trước:** `&http.Client{}` — không timeout, có thể treo vô hạn.

**Sau:**
- Qdrant: `&http.Client{Timeout: 10 * time.Second}`
- Voyage: `&http.Client{Timeout: 15 * time.Second}`

### 1.3 ✅ Giảm LLM Timeout & Retry

**File:** `config/config.go`

| Config | Trước | Sau |
|--------|-------|-----|
| `max_total_timeout` | 60s | **20s** |
| `retry_attempts` | 3 | **2** |

Worst case giảm từ 60s xuống 20s.

### 1.4 ✅ Loại bỏ Double Reranking

**File:** `internal/task/usecase/helpers.go`

**Trước:** Pipeline `RRF₁ (qdrant repo) → BM25+RRF₂ (usecase)` — hai lần fusion làm méo score.

**Sau:** BM25+RRF₂ đã loại bỏ. Usecase chỉ limit candidates + optional Voyage cross-encoder rerank. RRF fusion chỉ xảy ra 1 lần ở repo layer.

### 1.5 ✅ System Prompt Chi Tiết

**File:** `internal/agent/usecase/constant.go`

**Trước:** ~5 dòng, chỉ nói "thân thiện".

**Sau:** Prompt đầy đủ bao gồm:
- Persona chi tiết (xưng hô, giọng văn, từ ngữ tự nhiên: "nhé", "nha", "ạ", "nè")
- Quy tắc phản hồi (ngắn gọn 2-4 câu, bullet points, không bịa)
- Ví dụ tone mẫu
- Hướng dẫn xử lý khi không tìm thấy info

### 1.6 ✅ Time Context → System Prompt

**File:** `internal/agent/usecase/process_query.go`, `internal/agent/graph/node_agent.go`, `internal/agent/graph/state.go`

**Trước:** `enhancedQuery := query + timeContext` — append ~200 token vào mỗi user message, lặp trong history.

**Sau:** Time context lưu vào `state.TimeContext`, inject vào system prompt 1 lần/request. User message history sạch, không lặp.

### 1.7 ✅ Agent Temperature = 0.7

**File:** `internal/agent/graph/node_agent.go`

**Trước:** Không set Temperature → phụ thuộc default của provider (không nhất quán).

**Sau:** `Temperature: 0.7` — rõ ràng, phản hồi tự nhiên hơn. Router giữ 0.1, task parsing giữ 0.2.

### 1.8 ✅ `isUserConfirmed()` Hỗ Trợ Tiếng Việt Có Dấu

**File:** `internal/agent/usecase/process_query.go`

**Trước:** Chỉ match ASCII: `"dong y"`, `"xac nhan"`, `"co"`.

**Sau:** Thêm Unicode: `"đồng ý"`, `"xác nhận"`, `"có"`, `"được"`, `"chắc chắn"`, `"ừ"`, `"ờ"`, `"uh huh"`, `"rồi"`.

---

## 2. Hiệu suất sau khi fix

| Flow | Trước | Sau |
|------|-------|-----|
| Typical RAG query | ~3-5s | **~1-2s** |
| Worst case timeout | 60s | **20s** |
| Rule-based router | ~5ms | ~5ms (không đổi) |
| Ambiguous message | 4-8s (2 LLM) | 4-8s (chưa fix, xem §3.1) |

---

## 3. Vấn đề còn lại (chưa fix)

### 3.1 Double LLM Call — Router + Agent

Flow cho message mơ hồ (rule-based score < 80):
1. **Router Classify** → LLM call #1 (~1-3s)
2. **Agent ProcessQuery** → LLM call #2 (~1-3s)

**Đề xuất:** Giảm threshold rule-based từ 80 → 70, hoặc merge router vào agent. **Tiết kiệm: 1-3s.**

### 3.2 Router Prompt Gửi Sai Role

**File:** `internal/router/usecase/classify.go`

Router gửi prompt phân loại như `user` message thay vì `SystemInstruction`. Một số provider cache system instruction hiệu quả hơn.

### 3.3 `isAskingUser()` Detection Quá Đơn Giản

**File:** `internal/agent/graph/node_agent.go`

Hàm chỉ check `?` + vài keyword. Dễ false positive/negative. Có thể gây bot treo ở `WAITING_FOR_HUMAN`.

**Đề xuất:** Chuyển thành tool `request_human_input` để LLM tự quyết định.

### 3.4 Query Embedding Không Cache

`SearchTasks()` gọi `Voyage.Embed()` cho mỗi query — HTTP call ra ngoài.

**Đề xuất:** LRU cache embed results (key = query hash, TTL 5 phút).

### 3.5 Context Compression Thô Sơ

**File:** `internal/agent/graph/state.go`

`CompressIfNeeded()` cắt mỗi message còn 120 ký tự rồi nối bằng `|`. Mất ngữ cảnh.

**Đề xuất:** Dùng LLM rẻ/nhanh để tóm tắt conversation history.

---

## 4. Test Coverage Summary

| Package | Tests trước | Tests thêm | Tổng |
|---------|------------|------------|------|
| automation/usecase | 0 | 18 | 18 |
| webhook/usecase | 0 | 19 | 19 |
| task/usecase | 0 | 26 | 26 |
| router/usecase | ~16 | 11 | ~27 |
| agent/graph | ~11 | 0 | ~11 |
| **Tổng** | **~27** | **74** | **~101** |

Tất cả 101 tests **PASS** (0 failures).
