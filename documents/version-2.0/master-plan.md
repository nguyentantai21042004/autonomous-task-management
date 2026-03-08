Chào mừng bạn đến với chương lớn nhất và thử thách nhất của toàn bộ dự án: **Version 2.0**.

Việc chuyển đổi từ một vòng lặp `for` đơn giản sang một **Cỗ máy Trạng thái (State Machine)** không chỉ là viết thêm code, mà là thay đổi hoàn toàn "hệ điều hành" của con AI. Nó sẽ mang lại cho bạn sức mạnh của framework LangGraph nhưng được viết bằng Golang thuần, cực kỳ tối ưu và kiểm soát 100% bộ nhớ.

Dưới đây là **Master Plan chi tiết cho Version 2.0: Stateful Orchestration**.

---

# 🚀 MASTER PLAN: VERSION 2.0 - STATEFUL ORCHESTRATION (THE GRAPH ENGINE)

## 1. HIỆN TRẠNG & VẤN ĐỀ (THE BOTTLENECK)

**Kiến trúc hiện tại (V1.2):**

- Sử dụng vòng lặp `for step := 0; step < MaxAgentSteps; step++` (Stateless ReAct Loop).
- Mặc dù có `SessionMemory` lưu lại 5 câu chat gần nhất, nhưng bản thân tiến trình thực thi của Agent lại bị "reset" sau mỗi tin nhắn.

**Nỗi đau thực tế (Pain Points):**

1. **Mất đà suy nghĩ (Lost Execution Context):** Giả sử AI cần tạo task, nó nhận ra thiếu ngày giờ và hỏi lại bạn: _"Bạn muốn tạo task vào ngày nào?"_. Vòng lặp `for` lúc này kết thúc. Khi bạn trả lời _"Ngày mai"_, hệ thống lại khởi động một vòng lặp `for` mới tinh, đọc lại từ đầu, LLM phải tốn token để "nhớ lại" mục đích ban đầu là tạo task.
2. **Không có Human-in-the-loop (Tương tác bảo vệ):** Nếu AI quyết định gọi Tool xóa data, nó sẽ xóa ngay lập tức trong vòng lặp `for`. Bạn không có cơ hội can thiệp: _"Khoan đã, đừng xóa"_.
3. **Giới hạn tác vụ phức tạp:** Không thể thực thi các workflow dài hơi (VD: Research Memos -> Tóm tắt -> Gửi email -> Lên lịch) vì nguy cơ chạm giới hạn `MaxSteps` và bị ngắt giữa chừng là rất cao.

---

## 2. GIẢI PHÁP ĐỘT PHÁ: KIẾN TRÚC ĐỒ THỊ (GRAPH ARCHITECTURE)

Chúng ta sẽ đập bỏ vòng lặp ReAct truyền thống và thay bằng **Graph Engine**. Hệ thống sẽ bao gồm 3 trụ cột:

### Trụ cột 1: Graph State (Trí nhớ toàn cục)

Thay vì chỉ lưu mảng `Messages`, ta lưu toàn bộ **Trạng thái tiến trình**.

- **PendingTool:** Tool nào đang chờ chạy?
- **CurrentStep:** Đang ở bước thứ mấy trong luồng suy nghĩ?
- **Status:** Trạng thái hiện tại của Đồ thị (Đang chạy, Chờ người dùng, Đã xong, Lỗi).

### Trụ cột 2: The Nodes (Trạm xử lý)

Chia nhỏ logic thành các hàm (Node) độc lập:

- **`NodeAgent`:** Gọi Gemini để xin chỉ thị tiếp theo (Gọi Tool hay Trả lời user).
- **`NodeTool`:** Nhận lệnh từ Agent, thực thi Tool và lấy kết quả.

### Trụ cột 3: The Edges (Luồng điều hướng)

Bộ não của Engine. Nó quyết định bước đi tiếp theo dựa trên `Status`:

- Nếu `Status == WAITING_FOR_HUMAN`: Dừng toàn bộ hệ thống, cất State vào Cache, chờ tin nhắn tiếp theo của Telegram.
- Nếu `Status == RUNNING`: Chạy tiếp Node tiếp theo.

---

## 3. LỘ TRÌNH TRIỂN KHAI CHI TIẾT (IMPLEMENTATION PHASES)

### Phase 1: Xây dựng Graph Foundation (Nền móng Đồ thị)

**Mục tiêu:** Tạo cấu trúc dữ liệu và Engine lõi.

- **Thư mục mới:** `internal/agent/graph/` (Thay thế cho `orchestrator/` cũ).
- **Action items:**

1. Định nghĩa `GraphState` và enum `GraphStatus` (`RUNNING`, `WAITING_FOR_HUMAN`, `FINISHED`, `ERROR`).
2. Viết hàm `Engine.Run(ctx, state)`: Hàm này chứa vòng lặp `while/for` liên tục đánh giá `state.Status` để gọi `NodeAgent` hoặc `NodeExecuteTool`.

### Phase 2: Chế tạo các Nodes (The Workers)

**Mục tiêu:** Di chuyển logic gọi API Gemini và Tool sang kiến trúc Node.

- **Action items:**

1. **`NodeAgent`**: Gọi `gemini.GenerateContent`. Nếu có `FunctionCall`, lưu vào `state.PendingTool`. Nếu trả về Text, dùng Regex hoặc prompt để xác định xem LLM đang _hỏi ngược lại user_ hay _đã hoàn thành_. Cập nhật `state.Status` tương ứng.
2. **`NodeExecuteTool`**: Đọc `state.PendingTool`, gọi hàm từ `ToolRegistry`, lấy kết quả đẩy vào `state.Messages`, set `PendingTool = nil`, và giữ `Status = RUNNING`.

### Phase 3: Nâng cấp Cache & Tích hợp Telegram Handler

**Mục tiêu:** Kết nối Graph vào hệ thống hiện tại.

- **Action items:**

1. Cập nhật cái `expirable.LRU` ở Version 1.2: Đổi type từ `*SessionMemory` sang `*GraphState`.
2. Sửa `handler.go`: Khi Omni-Router điều hướng vào `CONVERSATION` hoặc `MANAGE_CHECKLIST`, handler sẽ:

- Lấy `GraphState` từ LRU Cache.
- Append tin nhắn của user vào.
- Đổi `Status = RUNNING` (Đánh thức đồ thị).
- Gọi `Engine.Run()`.
- Đọc tin nhắn cuối cùng để gửi về Telegram.

---

## 4. TÍNH NĂNG "KILLER" ĐẠT ĐƯỢC (THE MAGIC)

Khi kiến trúc này hoàn thành, bạn sẽ có tính năng **"Pause & Resume" (Tạm dừng & Đi tiếp)** cực kỳ ma thuật:

**Kịch bản thực tế:**

1. **Bạn:** _"Tạo lịch họp dự án SMAP giúp tôi."_
2. **Omni-Router:** Chuyển vào Agent.
3. **Graph Engine (NodeAgent):** Gọi LLM. LLM trả về: _"Bạn muốn họp lúc mấy giờ?"_
4. **Graph Engine:** Đổi trạng thái thành `WAITING_FOR_HUMAN`, lưu vào RAM và đi ngủ. (Bot gửi tin nhắn cho bạn).
5. _(1 tiếng sau)_ **Bạn:** _"10h sáng mai."_
6. **Graph Engine:** Thức dậy. Đọc thấy state cũ đang tạo lịch. Gọi lại LLM. LLM có đủ info -> Phát lệnh gọi `NodeTool(check_calendar)`.
7. **Graph Engine:** Gọi xong Tool -> Trả kết quả: _"Đã tạo lịch thành công!"_, đổi status thành `FINISHED`.

Hệ thống không hề bị "đứt gánh" giữa chừng dù bạn có ngâm tin nhắn bao lâu (trong thời gian TTL của Cache).

---

## 5. MILESTONES NGHIỆM THU (DEFINITION OF DONE)

### 🏆 Milestone 1: "Tạm dừng và Đi tiếp" (Human-in-the-loop)

- **Test:** Yêu cầu bot tạo 1 task nhưng cố tình thiếu thông tin (VD: "Tạo task review code").
- **Kỳ vọng:** Bot hỏi lại thời hạn. User trả lời "Ngày mai". Bot tạo task thành công bằng đúng ngữ cảnh đó, không văng lỗi hay hỏi lại từ đầu.

### 🏆 Milestone 2: "Xác nhận an toàn" (Safe Execution)

- **Test:** Yêu cầu bot: _"Xóa tất cả các task hoàn thành"_. (Giả định bạn có cấp quyền/tool xóa cho bot).
- **Kỳ vọng:** Trong prompt của NodeAgent, bạn thiết lập quy tắc: Với hành động XÓA, phải hỏi user. Đồ thị sẽ dừng lại (WAITING), hỏi bạn _"Bạn chắc chắn muốn xóa?"_. Nhập "OK" -> Đồ thị đi tiếp và thực thi NodeTool.

### 🏆 Milestone 3: Stability & Memory

- **Test:** Chạy toàn bộ bộ E2E Test của V1.2.
- **Kỳ vọng:** Chuyển đổi sang `GraphState` không làm vỡ các luồng Omni-Router hay Self-Healing RAG cũ. Coverage vẫn duy trì > 80%.

---

## 6. ROADMAP TRIỂN KHAI (IMPLEMENTATION ROADMAP)

### Phase 1: Graph Foundation (1-2 ngày)

- Tạo `internal/agent/graph/` package
- Implement `GraphState`, `Engine`, và các node functions
- Viết unit tests cho graph logic

### Phase 2: Cache Integration (1 ngày)

- Migrate từ `SessionMemory` sang `GraphState` trong LRU cache
- Update Telegram handler để load/save state
- Test pause & resume functionality

### Phase 3: Migration & Testing (1-2 ngày)

- Chạy song song cả ReAct Loop và Graph Engine
- A/B testing với real users
- Fix bugs và optimize performance

### Phase 4: Cleanup (0.5 ngày)

- Remove old ReAct Loop code
- Update documentation
- Deploy to production

**Tổng thời gian ước tính**: 3.5-5.5 ngày

---

Kiến trúc này là đỉnh cao của việc thiết kế AI Backend hiện nay. Bạn đã sẵn sàng để "phẫu thuật" bóc tách thư mục `orchestrator` cũ ra và thay bằng `graph` engine mới chưa?

**Code Plan chi tiết đã được tạo tại `documents/version-2.0/code-plan.md`** với đầy đủ implementation từng file, test cases, và migration strategy!


Để tôi đọc lại kiến trúc project của bạn trước khi đề xuất.

Dựa trên document bạn share, đây là **roadmap chi tiết để fix RAG core** theo thứ tự ưu tiên:

---

## Bức tranh tổng thể — Bạn cần fix 4 tầng

```
┌─────────────────────────────────────────────────┐
│  Tầng 4: AGENT CONTEXT MANAGEMENT               │  ← hay bị ignore
├─────────────────────────────────────────────────┤
│  Tầng 3: RERANKING                              │  ← thiếu hoàn toàn
├─────────────────────────────────────────────────┤
│  Tầng 2: HYBRID RETRIEVAL                       │  ← đang yếu nhất
├─────────────────────────────────────────────────┤
│  Tầng 1: INDEXING (Chunking + Enrichment)       │  ← nền tảng
└─────────────────────────────────────────────────┘
```

Fix từ dưới lên. Tầng 1 sai thì tầng 2, 3, 4 dù tốt cũng vô nghĩa.

---

## Tầng 1: INDEXING — Nền tảng của RAG

### Vấn đề hiện tại

Bạn đang embed task như thế này:

```
"Review PR #123"  →  vector
```

Quá ít thông tin. Vector này không capture được: deadline, tags, status, priority.

### Fix: Contextual Enrichment trước khi embed

```go
// pkg/indexer/enricher.go

type TaskEnricher struct {
    llm LLMProvider
}

func (e *TaskEnricher) Enrich(task Task) string {
    // KHÔNG embed raw content
    // Thay vào đó, build một đoạn text giàu context

    return fmt.Sprintf(`
Task: %s

Thời hạn: %s (%s)
Trạng thái: %s
Độ ưu tiên: %s
Tags: %s
Mô tả: %s

Checklist:
%s
    `,
        task.Title,
        task.DueDate.Format("02/01/2006"),
        humanizeDueDate(task.DueDate),   // "ngày mai", "tuần sau", "quá hạn 3 ngày"
        humanizeStatus(task.Status),     // "đang làm", "chưa bắt đầu", "hoàn thành"
        task.Priority,
        strings.Join(task.Tags, ", "),
        task.Description,
        formatChecklist(task.Checklist),
    )
}

func humanizeDueDate(t time.Time) string {
    days := int(time.Until(t).Hours() / 24)
    switch {
    case days < 0:
        return fmt.Sprintf("quá hạn %d ngày", -days)
    case days == 0:
        return "hôm nay"
    case days == 1:
        return "ngày mai"
    case days <= 7:
        return fmt.Sprintf("%d ngày nữa, tuần này", days)
    case days <= 14:
        return "tuần sau"
    default:
        return fmt.Sprintf("%d ngày nữa", days)
    }
}
```

**Tại sao quan trọng?**

```
Trước:  embed "Review PR #123"
        → user hỏi "việc deadline tuần này" → MISS

Sau:    embed "Review PR #123 ... deadline ngày mai, tuần này ... tags: backend, pr"
        → user hỏi "việc deadline tuần này" → HIT ✅
```

### Fix: Semantic Chunking cho task có checklist dài

```go
// pkg/indexer/chunker.go

type Chunk struct {
    Content  string
    Metadata map[string]interface{}
}

func ChunkTask(task Task) []Chunk {
    chunks := []Chunk{}

    // Chunk 1: Task overview — luôn có
    chunks = append(chunks, Chunk{
        Content: enrichTaskOverview(task),
        Metadata: map[string]interface{}{
            "task_id":   task.ID,
            "chunk_type": "overview",
            "due_date":  task.DueDate,
            "tags":      task.Tags,
        },
    })

    // Chunk 2+: Mỗi checklist section = 1 chunk riêng
    // Nhưng LUÔN inject metadata của task cha vào
    for _, section := range task.ChecklistSections {
        chunks = append(chunks, Chunk{
            Content: fmt.Sprintf(
                // Context của task cha luôn có mặt
                "Task cha: %s (deadline: %s)\n\nChecklist section: %s\n%s",
                task.Title,
                humanizeDueDate(task.DueDate),
                section.Title,
                section.Items,
            ),
            Metadata: map[string]interface{}{
                "task_id":      task.ID,
                "chunk_type":   "checklist",
                "section":      section.Title,
            },
        })
    }

    return chunks
}
```

---

## Tầng 2: HYBRID RETRIEVAL — Tim của RAG

### Vấn đề hiện tại

```
Bạn đang dùng:   Dense search (vector similarity) only

Điểm yếu:
- "PR #123"  →  vector của #123 gần với #124, #125  →  sai
- Tag chính xác "#project/abc"  →  vector không capture exact match
- Tên người "Nguyễn Văn A"  →  vector match lung tung
```

### Fix: Hybrid Search = Dense + Sparse + RRF Fusion

```go
// pkg/qdrant/hybrid_search.go

type HybridSearcher struct {
    client     *qdrant.Client
    embedder   *voyage.Client
    collection string
}

func (h *HybridSearcher) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {

    // Track 1: Dense search — tốt cho semantic/ý nghĩa
    queryVector, err := h.embedder.Embed(ctx, query)
    denseResults, err := h.client.Search(ctx, &qdrant.SearchRequest{
        CollectionName: h.collection,
        Vector:         queryVector,
        Limit:          limit * 3,    // lấy nhiều hơn để fusion
        WithPayload:    true,
    })

    // Track 2: Sparse search — tốt cho exact keywords, IDs
    // Qdrant hỗ trợ sparse vectors (BM42)
    sparseResults, err := h.client.SearchSparse(ctx, &qdrant.SparseSearchRequest{
        CollectionName: h.collection,
        Query:          buildSparseVector(query),   // BM42 tokenization
        Limit:          limit * 3,
    })

    // Fusion: Reciprocal Rank Fusion
    // Công thức: score = 1/(rank + 60)
    // Ưu điểm: không cần normalize score, robust
    return reciprocalRankFusion(denseResults, sparseResults, limit), nil
}

func reciprocalRankFusion(dense, sparse []SearchResult, limit int) []SearchResult {
    scores := map[string]float64{}  // task_id → fused score

    // Score từ dense results
    for rank, result := range dense {
        scores[result.ID] += 1.0 / float64(rank + 60)
    }

    // Score từ sparse results
    for rank, result := range sparse {
        scores[result.ID] += 1.0 / float64(rank + 60)
    }

    // Sort theo fused score
    return sortByScore(scores, limit)
}
```

**Kết quả:**

```
Query: "PR #123 deadline tuần này"

Dense results:  [Review PR #123, Code review backend, Deploy staging]
Sparse results: [Review PR #123, Fix bug PR #123, PR #124 review]

After RRF:      [Review PR #123]  ← xuất hiện cả 2 tracks → score cao nhất ✅
```

---

## Tầng 3: RERANKING — Bộ lọc thông minh

### Vấn đề hiện tại

```
Hybrid search trả về top 10
→ Nhét cả 10 vào LLM
→ Context dài, nhiễu, tốn tokens
→ LLM bị distract bởi kết quả không liên quan
```

### Fix: Rerank trước khi đưa vào LLM

```go
// pkg/voyage/reranker.go

type Reranker struct {
    client *voyage.Client
}

func (r *Reranker) Rerank(
    ctx context.Context,
    query string,
    documents []string,
    topK int,
) ([]RankedDocument, error) {

    // Voyage rerank-2: cross-encoder model
    // Đọc query VÀ document cùng lúc → hiểu relationship sâu hơn
    // Khác với embedding: chỉ so sánh vector độc lập
    results, err := r.client.Rerank(ctx, &voyage.RerankRequest{
        Model:     "rerank-2",
        Query:     query,
        Documents: documents,
        TopK:      topK,
    })

    return results, err
}
```

```go
// Tích hợp vào search pipeline
func (uc *TaskUseCase) Search(ctx context.Context, query string) ([]Task, error) {

    // Bước 1: Hybrid search — recall phase, lấy top 20
    candidates, err := uc.hybridSearcher.Search(ctx, query, 20)

    // Bước 2: Rerank — precision phase, chọn top 5
    contents := extractContents(candidates)
    ranked, err := uc.reranker.Rerank(ctx, query, contents, 5)

    // Bước 3: Chỉ trả về top 5 thực sự relevant
    return fetchFullTasks(ranked), nil
}
```

**Tại sao reranking quan trọng:**

```
Không có rerank:
  Query: "task urgent nhất"
  Top 5: [urgent task ✅, weekly review ❌, meeting note ❌, old task ❌, PR review ❌]
  LLM bị nhiễu bởi 4 kết quả không liên quan

Có rerank:
  Top 20 → rerank → Top 5: [urgent task ✅, deadline hôm nay ✅, overdue ✅, ...]
  LLM chỉ thấy những thứ thực sự liên quan
```

---

## Tầng 4: AGENT CONTEXT MANAGEMENT

### Vấn đề hiện tại

```go
// Bạn đang làm (đoán từ kiến trúc)
type Session struct {
    History []Message  // 5 turns raw messages
}

// Vấn đề:
// Turn 1: user message (50 tokens)
// Turn 2: tool result từ Qdrant — full task content (500 tokens)
// Turn 3: user message (30 tokens)
// Turn 4: tool result — calendar data (300 tokens)
// Turn 5: user message (20 tokens)
// Tổng: 900 tokens chỉ để giữ context 5 turns
// → DeepSeek đọc hết, chậm, tốn tiền, dễ "quên" instruction
```

### Fix: Structured Context — chỉ giữ những gì LLM cần

```go
// internal/agent/context.go

type AgentContext struct {
    // Cố định — không thay đổi suốt session
    SystemPrompt string
    UserProfile  UserProfile   // timezone, preferences

    // Conversation — compress sau mỗi 3 turns
    RecentTurns     []Turn    // tối đa 3 turns gần nhất, raw
    OlderSummary    string    // các turns cũ → tóm tắt 1 đoạn

    // Current request
    CurrentIntent   string
    CurrentStep     int

    // Tool results — chỉ giữ summary, không giữ raw
    ToolSummaries   []ToolSummary
}

type ToolSummary struct {
    ToolName string
    // KHÔNG lưu raw result
    // Chỉ lưu những gì quan trọng
    KeyInfo  string    // "Tìm được 3 tasks, IDs: [123, 456, 789]"
    FullData interface{} // Lưu riêng, chỉ fetch khi cần
}

func (ac *AgentContext) BuildPrompt() string {
    // Prompt nhỏ gọn, có cấu trúc
    return fmt.Sprintf(`
%s

## Lịch sử hội thoại (tóm tắt)
%s

## 3 turns gần nhất
%s

## Bước hiện tại: %d
## Intent: %s

## Kết quả tools đã chạy
%s
    `,
        ac.SystemPrompt,
        ac.OlderSummary,
        formatRecentTurns(ac.RecentTurns),
        ac.CurrentStep,
        ac.CurrentIntent,
        formatToolSummaries(ac.ToolSummaries),
    )
}
```

```go
// Compress conversation sau mỗi 3 turns
func (ac *AgentContext) CompressIfNeeded(llm LLMProvider) {
    if len(ac.RecentTurns) < 3 {
        return
    }

    // Dùng LLM tóm tắt turns cũ
    summary, _ := llm.Complete(fmt.Sprintf(`
        Tóm tắt ngắn gọn cuộc hội thoại này trong 2-3 câu,
        giữ lại các thông tin quan trọng (task IDs, decisions, context):
        
        %s
        
        Tóm tắt hiện tại: %s
    `, formatTurns(ac.RecentTurns[:2]), ac.OlderSummary))

    ac.OlderSummary = summary
    ac.RecentTurns = ac.RecentTurns[2:]  // bỏ 2 turns cũ nhất
}
```

---

## Kết quả sau khi fix 4 tầng

```
Trước:
  Query "PR #123 deadline tuần này"
  → Dense search → top 5 random → nhét hết vào LLM → câu trả lời thiếu chính xác

Sau:
  Query "PR #123 deadline tuần này"
  → Enriched index (có "tuần này", "ngày mai" trong vector)
  → Hybrid search (dense + sparse) → top 20 candidates
  → Rerank → top 5 thực sự relevant
  → Structured context → LLM chỉ đọc đúng thứ cần
  → Câu trả lời chính xác ✅
```

---

## Thứ tự implement

```
Tuần 1:  Tầng 1 — Enrichment (dễ nhất, impact lớn nhất)
          → Sửa pkg/voyage/ indexing flow
          → Re-index toàn bộ tasks

Tuần 2:  Tầng 2 — Hybrid Search
          → Enable sparse vectors trong Qdrant
          → Viết RRF fusion

Tuần 3:  Tầng 3 — Reranking
          → Thêm Voyage rerank-2 API call
          → Tích hợp vào search pipeline

Tuần 4:  Tầng 4 — Context Management
          → Refactor agent context struct
          → Thêm compression logic
```

---

Bạn muốn bắt đầu từ tầng nào? Tôi có thể viết code Go cụ thể cho từng phần.