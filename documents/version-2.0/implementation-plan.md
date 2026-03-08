# V2.0 IMPLEMENTATION PLAN — STATEFUL ORCHESTRATION + RAG UPGRADE

Mục tiêu: Nâng cấp hệ thống từ stateless ReAct loop đơn giản lên Graph Engine
tương đương LangGraph, đồng thời nâng cấp RAG pipeline từ dense-only lên
hybrid search + reranking.

---

## TONG QUAN HIEN TRANG (V1.2)

### Van de 1 — Stateless ReAct Loop

File: `internal/agent/usecase/process_query.go:38`

```go
for step := 0; step < MaxAgentSteps; step++ {
    // Vong lap ket thuc → execution context bien mat
    // LLM hoi "ban muon ngay nao?" → vong for ket thuc
    // User reply "Ngay mai" → for moi → LLM khong nho dang lam gi
}
```

**Hau qua:** Agent bi "chet nao" sau moi tin nhan. Khong the thuc hien workflow
phuc tap hon 5 buoc.

### Van de 2 — SessionMemory qua don gian

File: `internal/agent/types.go:9-14`

```go
type SessionMemory struct {
    Messages    []llmprovider.Message  // chi luu raw messages
    LastUpdated time.Time
}
```

Khong luu: step hien tai, tool dang pending, trang thai workflow, intent.

### Van de 3 — RAG chi dung Dense Search

File: `internal/task/usecase/answer_query.go:29-34`

```go
searchResults, err := uc.vectorRepo.SearchTasks(ctx, repository.SearchTasksOptions{
    Query: input.Query,
    Limit: MaxTasksInContext,  // 5 results, nhet thang vao LLM
})
```

Khong co: Hybrid search, Reranking, Contextual Enrichment.
Ket qua: "PR #123" → vector match voi #124, #125. Query "deadline tuan nay"
→ miss vi vector khong co chuoi "tuan nay".

### Van de 4 — Cache thu cong de bi loi

File: `internal/agent/usecase/new.go:17`

```go
sessionCache map[string]*agent.SessionMemory  // plain map + mutex thu cong
```

Nen dung: `expirable.LRU` da co trong go.mod (`github.com/hashicorp/golang-lru/v2`)

---

## PHASE 1 — GRAPH ENGINE FOUNDATION

**Tao thu muc moi:** `internal/agent/graph/`

### 1.1 — `internal/agent/graph/state.go`

```go
package graph

import (
    "time"
    "autonomous-task-management/pkg/llmprovider"
)

type GraphStatus string

const (
    StatusRunning         GraphStatus = "RUNNING"
    StatusWaitingForHuman GraphStatus = "WAITING_FOR_HUMAN"
    StatusFinished        GraphStatus = "FINISHED"
    StatusError           GraphStatus = "ERROR"
)

// GraphState thay the SessionMemory — luu TOAN BO trang thai tien trinh.
// Day la khac biet lon nhat so voi V1.2: luu ca execution context,
// cho phep pause/resume giua cac tin nhan.
type GraphState struct {
    UserID  string
    Status  GraphStatus

    // Conversation history
    Messages []llmprovider.Message

    // Execution context (KHONG co trong V1.2)
    PendingTool   *llmprovider.FunctionCall // tool dang cho chay
    CurrentStep   int
    CurrentIntent string

    // Context compression (Tầng 4 RAG)
    OlderSummary string               // cac turns cu duoc tom tat
    RecentTurns  []llmprovider.Message // chi giu 6 turns gan nhat raw

    // Metadata
    LastUpdated time.Time
    TTL         time.Duration
}

func NewGraphState(userID string) *GraphState {
    return &GraphState{
        UserID:      userID,
        Status:      StatusFinished,
        Messages:    []llmprovider.Message{},
        LastUpdated: time.Now(),
        TTL:         30 * time.Minute, // tang tu 10m len 30m
    }
}

func (s *GraphState) IsExpired() bool {
    return time.Since(s.LastUpdated) > s.TTL
}

func (s *GraphState) AppendMessage(msg llmprovider.Message) {
    s.Messages = append(s.Messages, msg)
    s.RecentTurns = append(s.RecentTurns, msg)
    // Giu toi da 6 turns gan nhat
    if len(s.RecentTurns) > 6 {
        s.RecentTurns = s.RecentTurns[len(s.RecentTurns)-6:]
    }
}
```

### 1.2 — `internal/agent/graph/errors.go`

```go
package graph

import "errors"

var (
    ErrEmptyResponse = errors.New("empty LLM response")
    ErrNoPendingTool = errors.New("no pending tool in state")
    ErrMaxSteps      = errors.New("exceeded max graph steps")
)
```

### 1.3 — `internal/agent/graph/node_agent.go`

```go
package graph

import (
    "context"
    "strings"
    "autonomous-task-management/pkg/llmprovider"
)

// NodeAgent: goi LLM, phan tich ket qua, cap nhat Status.
// Tuong duong: Reason step trong ReAct cu, nhung co the PAUSE.
func NodeAgent(
    ctx context.Context,
    state *GraphState,
    llm llmprovider.IManager,
    tools []llmprovider.Tool,
    systemPrompt string,
) error {
    req := &llmprovider.Request{
        SystemInstruction: &llmprovider.Message{
            Parts: []llmprovider.Part{{Text: systemPrompt}},
        },
        Messages: state.Messages,
        Tools:    tools,
    }

    resp, err := llm.GenerateContent(ctx, req)
    if err != nil {
        state.Status = StatusError
        return err
    }

    if len(resp.Content.Parts) == 0 {
        state.Status = StatusError
        return ErrEmptyResponse
    }

    part := resp.Content.Parts[0]

    // Append LLM response vao history
    state.AppendMessage(resp.Content)
    state.CurrentStep++

    if part.FunctionCall != nil {
        state.PendingTool = part.FunctionCall

        // Destructive operations → hoi xac nhan truoc
        if isDangerousOperation(part.FunctionCall.Name) {
            state.Status = StatusWaitingForHuman
        } else {
            state.Status = StatusRunning // safe tool → chay ngay
        }
        return nil
    }

    // LLM tra ve text → phan tich: hoi user hay da xong?
    if isAskingUser(part.Text) {
        state.Status = StatusWaitingForHuman
    } else {
        state.Status = StatusFinished
    }
    return nil
}

// isDangerousOperation: cac tool can confirm truoc khi chay
func isDangerousOperation(toolName string) bool {
    dangerous := map[string]bool{
        "delete_task":      true,
        "delete_all_tasks": true,
        "complete_all":     true,
    }
    return dangerous[toolName]
}

// isAskingUser: kiem tra LLM co dang hoi nguoc lai user khong
// Dung keyword detection don gian, co the mo rong them
func isAskingUser(text string) bool {
    questionIndicators := []string{
        "?", "ban muon", "vui long cho biet", "ban co the",
        "ngay nao", "may gio", "khi nao", "o dau",
    }
    lower := strings.ToLower(text)
    for _, indicator := range questionIndicators {
        if strings.Contains(lower, indicator) {
            return true
        }
    }
    return false
}
```

### 1.4 — `internal/agent/graph/node_tool.go`

```go
package graph

import (
    "context"
    "autonomous-task-management/internal/agent"
    "autonomous-task-management/pkg/llmprovider"
)

// NodeExecuteTool: doc PendingTool, thuc thi, luu ket qua vao state.
// Tuong duong: Act + Observe step trong ReAct cu.
func NodeExecuteTool(
    ctx context.Context,
    state *GraphState,
    registry *agent.ToolRegistry,
) error {
    if state.PendingTool == nil {
        state.Status = StatusError
        return ErrNoPendingTool
    }

    toolName := state.PendingTool.Name
    tool, ok := registry.Get(toolName)

    var toolResult interface{}
    if !ok {
        toolResult = map[string]string{"error": "tool not found: " + toolName}
    } else {
        result, err := tool.Execute(ctx, state.PendingTool.Args)
        if err != nil {
            toolResult = map[string]string{"error": err.Error()}
        } else {
            toolResult = result
        }
    }

    // Append tool result vao messages
    state.AppendMessage(llmprovider.Message{
        Role: "function",
        Parts: []llmprovider.Part{{
            FunctionResponse: &llmprovider.FunctionResponse{
                Name:     toolName,
                Response: toolResult,
            },
        }},
    })

    // Reset pending tool, tiep tuc reasoning
    state.PendingTool = nil
    state.Status = StatusRunning
    return nil
}
```

### 1.5 — `internal/agent/graph/engine.go`

```go
package graph

import (
    "context"
    "fmt"
    "autonomous-task-management/internal/agent"
    "autonomous-task-management/pkg/llmprovider"
    pkgLog "autonomous-task-management/pkg/log"
)

// MaxGraphSteps tang tu 5 (V1.2) len 10 vi pause/resume giam token waste
const MaxGraphSteps = 10

type Engine struct {
    llm          llmprovider.IManager
    registry     *agent.ToolRegistry
    l            pkgLog.Logger
    systemPrompt string
    tools        []llmprovider.Tool
}

func NewEngine(
    llm llmprovider.IManager,
    registry *agent.ToolRegistry,
    l pkgLog.Logger,
    systemPrompt string,
) *Engine {
    return &Engine{
        llm:          llm,
        registry:     registry,
        l:            l,
        systemPrompt: systemPrompt,
        tools:        registry.ToFunctionDefinitions(),
    }
}

// Run thuc thi do thi tu state hien tai.
// Dung khi: FINISHED, WAITING_FOR_HUMAN, ERROR, hoac MaxSteps.
func (e *Engine) Run(ctx context.Context, state *GraphState) error {
    for state.CurrentStep < MaxGraphSteps {
        e.l.Infof(ctx, "graph.engine: step=%d status=%s", state.CurrentStep, state.Status)

        switch state.Status {
        case StatusRunning:
            if state.PendingTool != nil {
                // Co tool pending → chay tool truoc
                if err := NodeExecuteTool(ctx, state, e.registry); err != nil {
                    return fmt.Errorf("NodeExecuteTool: %w", err)
                }
            } else {
                // Khong co tool → reason tiep
                if err := NodeAgent(ctx, state, e.llm, e.tools, e.systemPrompt); err != nil {
                    return fmt.Errorf("NodeAgent: %w", err)
                }
            }

        case StatusWaitingForHuman:
            e.l.Infof(ctx, "graph.engine: pausing — waiting for human input")
            return nil // Caller se luu state vao cache

        case StatusFinished, StatusError:
            return nil
        }
    }

    e.l.Warnf(ctx, "graph.engine: exceeded MaxGraphSteps (%d)", MaxGraphSteps)
    state.Status = StatusFinished
    return nil
}

// GetLastResponse tra ve tin nhan cuoi cung cua assistant
func (e *Engine) GetLastResponse(state *GraphState) string {
    for i := len(state.Messages) - 1; i >= 0; i-- {
        msg := state.Messages[i]
        if msg.Role == "assistant" && len(msg.Parts) > 0 && msg.Parts[0].Text != "" {
            return msg.Parts[0].Text
        }
    }
    return ""
}
```

---

## PHASE 2 — KIEM TRA CACHE & UPDATE USECASE

### 2.1 — Update `internal/agent/usecase/new.go`

Thay `map[string]*SessionMemory + sync.RWMutex` bang `expirable.LRU`:

```go
package usecase

import (
    "time"

    "github.com/hashicorp/golang-lru/v2/expirable"

    "autonomous-task-management/internal/agent"
    "autonomous-task-management/internal/agent/graph"
    pkgLog "autonomous-task-management/pkg/log"
    "autonomous-task-management/pkg/llmprovider"
)

type implUseCase struct {
    llm        llmprovider.IManager
    registry   *agent.ToolRegistry
    l          pkgLog.Logger
    timezone   string
    engine     *graph.Engine
    stateCache *expirable.LRU[string, *graph.GraphState]
}

func New(llm llmprovider.IManager, registry *agent.ToolRegistry, l pkgLog.Logger, tz string) agent.UseCase {
    if tz == "" {
        tz = "Asia/Ho_Chi_Minh"
    }
    cache := expirable.NewLRU[string, *graph.GraphState](1000, nil, 30*time.Minute)
    engine := graph.NewEngine(llm, registry, l, SystemPromptAgent)

    return &implUseCase{
        llm:        llm,
        registry:   registry,
        l:          l,
        timezone:   tz,
        engine:     engine,
        stateCache: cache,
    }
}
```

### 2.2 — Rewrite `internal/agent/usecase/process_query.go`

```go
package usecase

import (
    "context"
    "strings"
    "time"

    "autonomous-task-management/internal/agent/graph"
    "autonomous-task-management/internal/model"
    "autonomous-task-management/pkg/llmprovider"
)

func (uc *implUseCase) ProcessQuery(ctx context.Context, sc model.Scope, query string) (string, error) {
    // Inject time context
    timeContext := buildTimeContext(uc.timezone)
    enhancedQuery := query + timeContext

    // Load hoac tao moi GraphState
    state, ok := uc.stateCache.Get(sc.UserID)
    if !ok || state.IsExpired() {
        state = graph.NewGraphState(sc.UserID)
    }

    // Append user message
    state.AppendMessage(llmprovider.Message{
        Role:  "user",
        Parts: []llmprovider.Part{{Text: enhancedQuery}},
    })

    // Neu graph dang WAITING_FOR_HUMAN → user vua reply → xu ly tiep
    if state.Status == graph.StatusWaitingForHuman {
        if state.PendingTool != nil {
            // Dangerous op dang cho confirm
            if isUserConfirmed(query) {
                state.Status = graph.StatusRunning
            } else {
                state.Status = graph.StatusFinished
                state.PendingTool = nil
                uc.stateCache.Add(sc.UserID, state)
                return "Da huy thao tac.", nil
            }
        } else {
            // LLM da hoi user, gio co answer → tiep tuc reason
            state.Status = graph.StatusRunning
        }
    } else {
        state.Status = graph.StatusRunning
    }

    state.CurrentStep = 0 // reset step counter cho moi turn

    // Chay engine
    if err := uc.engine.Run(ctx, state); err != nil {
        return "", err
    }

    // Gioi han history tranh context bloat
    if len(state.Messages) > MaxSessionHistory {
        state.Messages = state.Messages[len(state.Messages)-MaxSessionHistory:]
    }
    state.LastUpdated = time.Now()

    // Luu state lai (ke ca khi WAITING — de resume sau)
    uc.stateCache.Add(sc.UserID, state)

    return uc.engine.GetLastResponse(state), nil
}

// isUserConfirmed: kiem tra user co dong y voi dangerous operation khong
func isUserConfirmed(text string) bool {
    lower := strings.ToLower(strings.TrimSpace(text))
    confirmWords := []string{"ok", "yes", "dong y", "xac nhan", "co", "duoc"}
    for _, word := range confirmWords {
        if lower == word || strings.HasPrefix(lower, word) {
            return true
        }
    }
    return false
}
```

### 2.3 — Update `internal/agent/usecase/helpers.go`

Xoa cac method lien quan den sessionCache cu:

- `cleanupExpiredSessions()` — khong can, LRU tu quan ly TTL
- `getSession()` — thay bang stateCache.Get()
- `cacheMutex` — khong can, LRU thread-safe

Giu lai:

- `ClearSession()` — doi sang `stateCache.Remove(userID)`
- `GetSessionMessages()` — doi sang `stateCache.Get()` + lay Messages

---

## PHASE 3 — RAG UPGRADE (4 TANG)

### Tang 1: Contextual Enrichment — `pkg/indexer/enricher.go`

```go
package indexer

import (
    "fmt"
    "strings"
    "time"
)

// EnrichTaskForEmbedding tao enriched text truoc khi embed.
// Thay vi embed "Review PR #123" (qua it thong tin),
// embed day du context de vector capture duoc deadline, tags, trang thai.
func EnrichTaskForEmbedding(title, content string, tags []string, dueDate time.Time, timezone string) string {
    loc, err := time.LoadLocation(timezone)
    if err != nil {
        loc = time.UTC
    }
    now := time.Now().In(loc)

    parts := []string{
        fmt.Sprintf("Task: %s", title),
    }

    if !dueDate.IsZero() {
        parts = append(parts,
            fmt.Sprintf("Deadline: %s (%s)", dueDate.Format("02/01/2006"), humanizeDueDate(dueDate, now)),
        )
    }

    if len(tags) > 0 {
        parts = append(parts, fmt.Sprintf("Tags: %s", strings.Join(tags, ", ")))
    }

    if content != "" {
        parts = append(parts, fmt.Sprintf("Noi dung: %s", content))
    }

    return strings.Join(parts, "\n")
}

func humanizeDueDate(due, now time.Time) string {
    days := int(due.Sub(now).Hours() / 24)
    switch {
    case days < 0:
        return fmt.Sprintf("qua han %d ngay", -days)
    case days == 0:
        return "hom nay"
    case days == 1:
        return "ngay mai"
    case days <= 7:
        return fmt.Sprintf("%d ngay nua, tuan nay", days)
    case days <= 14:
        return "tuan sau"
    default:
        return fmt.Sprintf("%d ngay nua", days)
    }
}
```

**Noi tich hop:** `internal/task/usecase/create_bulk.go` khi goi VoyageAI embed —
thay `voyage.Embed(task.Content)` bang `voyage.Embed(indexer.EnrichTaskForEmbedding(...))`

### Tang 2: Hybrid Search — `pkg/qdrant/hybrid_search.go`

```go
package qdrant

import "context"

type HybridSearcher struct {
    client     *Client
    collection string
}

// SearchHybrid: Dense (semantic) + Sparse (BM42 keyword) + RRF Fusion
func (h *HybridSearcher) SearchHybrid(ctx context.Context, queryVector []float32, queryText string, limit int) ([]SearchResult, error) {
    // Track 1: Dense search — tot cho semantic/y nghia
    denseResults, err := h.client.Search(ctx, &SearchRequest{
        CollectionName: h.collection,
        Vector:         queryVector,
        Limit:          limit * 3,
        WithPayload:    true,
    })
    if err != nil {
        return nil, err
    }

    // Track 2: Sparse search (BM42) — tot cho exact keywords, IDs
    // Can enable sparse vectors trong Qdrant collection config
    sparseResults, err := h.client.SearchSparse(ctx, queryText, limit*3)
    if err != nil {
        // Fallback: chi dung dense neu sparse fail
        return denseResults[:min(limit, len(denseResults))], nil
    }

    // RRF Fusion: score = 1/(rank + 60)
    // Uu diem: khong can normalize score, robust voi outliers
    return reciprocalRankFusion(denseResults, sparseResults, limit), nil
}

func reciprocalRankFusion(dense, sparse []SearchResult, limit int) []SearchResult {
    scores := map[string]float64{}

    for rank, r := range dense {
        scores[r.ID] += 1.0 / float64(rank+60)
    }
    for rank, r := range sparse {
        scores[r.ID] += 1.0 / float64(rank+60)
    }

    return sortByFusedScore(scores, dense, sparse, limit)
}
```

### Tang 3: Reranking — `pkg/voyage/reranker.go`

```go
package voyage

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
)

type Reranker struct {
    apiKey string
    model  string
}

func NewReranker(apiKey string) *Reranker {
    return &Reranker{apiKey: apiKey, model: "rerank-2"}
}

type rerankRequest struct {
    Model     string   `json:"model"`
    Query     string   `json:"query"`
    Documents []string `json:"documents"`
    TopK      int      `json:"top_k"`
}

type RerankResult struct {
    Index          int     `json:"index"`
    RelevanceScore float64 `json:"relevance_score"`
}

// Rerank: cross-encoder model doc query + document cung luc
// → hieu relationship sau hon embedding biet lap
func (r *Reranker) Rerank(ctx context.Context, query string, documents []string, topK int) ([]RerankResult, error) {
    body, _ := json.Marshal(rerankRequest{
        Model:     r.model,
        Query:     query,
        Documents: documents,
        TopK:      topK,
    })

    req, _ := http.NewRequestWithContext(ctx, "POST",
        "https://api.voyageai.com/v1/rerank",
        bytes.NewBuffer(body),
    )
    req.Header.Set("Authorization", "Bearer "+r.apiKey)
    req.Header.Set("Content-Type", "application/json")

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("rerank API: %w", err)
    }
    defer resp.Body.Close()

    var result struct {
        Data []RerankResult `json:"data"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("rerank decode: %w", err)
    }
    return result.Data, nil
}
```

### Tang 4: Context Compression

Them method vao `GraphState` trong `internal/agent/graph/state.go`:

```go
// CompressIfNeeded: tom tat cac turns cu de giam token cost
// Goi sau moi 3 turns de tranh context bloat
func (s *GraphState) CompressIfNeeded(ctx context.Context, llm llmprovider.IManager) {
    if len(s.RecentTurns) < 6 {
        return // Chua du de compress
    }

    // Tom tat 3 turns dau, giu 3 turns cuoi nguyen
    toSummarize := s.RecentTurns[:3]
    toKeep := s.RecentTurns[3:]

    var textToSummarize strings.Builder
    for _, msg := range toSummarize {
        if len(msg.Parts) > 0 {
            textToSummarize.WriteString(msg.Role + ": " + msg.Parts[0].Text + "\n")
        }
    }

    summaryPrompt := fmt.Sprintf(
        "Tom tat ngan gon (2-3 cau) cuoc hoi thoai, giu thong tin quan trong (task IDs, quyet dinh, context):\n\n%s\n\nTom tat hien tai: %s",
        textToSummarize.String(),
        s.OlderSummary,
    )

    resp, err := llm.GenerateContent(ctx, &llmprovider.Request{
        Messages: []llmprovider.Message{
            {Role: "user", Parts: []llmprovider.Part{{Text: summaryPrompt}}},
        },
        MaxTokens: 200,
    })
    if err == nil && len(resp.Content.Parts) > 0 {
        s.OlderSummary = resp.Content.Parts[0].Text
    }

    s.RecentTurns = toKeep
}
```

### Tang 1+2+3 tich hop vao answer_query

Update `internal/task/usecase/answer_query.go`:

```go
// Thay the searchResults, err := uc.vectorRepo.SearchTasks(...) bang:

// Buoc 1: Tao enriched query vector
queryVector, err := uc.embeddingClient.Embed(ctx, input.Query)

// Buoc 2: Hybrid search — recall phase, lay top 20
candidates, err := uc.hybridSearcher.SearchHybrid(ctx, queryVector, input.Query, 20)

// Buoc 3: Lay content cua tung candidate
contents := make([]string, len(candidates))
for i, c := range candidates {
    task, _ := uc.repo.GetTask(ctx, c.MemoID)
    contents[i] = task.Content
}

// Buoc 4: Rerank — precision phase, chon top 5
ranked, err := uc.reranker.Rerank(ctx, input.Query, contents, MaxTasksInContext)

// Buoc 5: Chi dung top 5 thuc su relevant
finalResults := selectByRankedIndices(candidates, ranked)
```

---

## CHECKLIST IMPLEMENTATION CHI TIET

### PHASE 1 — Graph Engine

- [ ] **1.1** Tao `internal/agent/graph/state.go`
  - [ ] Dinh nghia `GraphStatus` enum (4 trang thai)
  - [ ] Dinh nghia `GraphState` struct (thay the `SessionMemory`)
  - [ ] Ham `NewGraphState(userID string) *GraphState`
  - [ ] Method `IsExpired() bool`
  - [ ] Method `AppendMessage(msg)`

- [ ] **1.2** Tao `internal/agent/graph/errors.go`
  - [ ] `ErrEmptyResponse`
  - [ ] `ErrNoPendingTool`
  - [ ] `ErrMaxSteps`

- [ ] **1.3** Tao `internal/agent/graph/node_agent.go`
  - [ ] Ham `NodeAgent(ctx, state, llm, tools, systemPrompt) error`
  - [ ] Logic: FunctionCall → set PendingTool → StatusRunning / StatusWaitingForHuman
  - [ ] Logic: Text response → isAskingUser() → StatusWaitingForHuman / StatusFinished
  - [ ] Ham `isDangerousOperation(toolName) bool`
  - [ ] Ham `isAskingUser(text) bool`

- [ ] **1.4** Tao `internal/agent/graph/node_tool.go`
  - [ ] Ham `NodeExecuteTool(ctx, state, registry) error`
  - [ ] Doc `state.PendingTool`, chay tool, append result
  - [ ] Reset `PendingTool = nil`, set `StatusRunning`

- [ ] **1.5** Tao `internal/agent/graph/engine.go`
  - [ ] Struct `Engine` voi LLM, registry, logger, tools
  - [ ] Ham `NewEngine(llm, registry, l, systemPrompt) *Engine`
  - [ ] Method `Run(ctx, state) error` — main loop
  - [ ] Method `GetLastResponse(state) string`

### PHASE 2 — Ket noi vao Usecase

- [ ] **2.1** Update `internal/agent/usecase/new.go`
  - [ ] Import `expirable.LRU` tu `github.com/hashicorp/golang-lru/v2/expirable`
  - [ ] Thay `sessionCache map + cacheMutex` bang `stateCache *expirable.LRU`
  - [ ] Import va khoi tao `graph.Engine`
  - [ ] Cap nhat ham `New()` khoi tao cache + engine

- [ ] **2.2** Rewrite `internal/agent/usecase/process_query.go`
  - [ ] Load state tu `stateCache.Get()`
  - [ ] Xu ly khi WAITING_FOR_HUMAN (resume / confirm / cancel)
  - [ ] Goi `engine.Run(ctx, state)`
  - [ ] Luu state bang `stateCache.Add()`
  - [ ] Ham `isUserConfirmed(text) bool`

- [ ] **2.3** Update `internal/agent/usecase/helpers.go`
  - [ ] Xoa `cleanupExpiredSessions()` (LRU tu xu ly)
  - [ ] Xoa `getSession()` (thay bang stateCache.Get)
  - [ ] Cap nhat `ClearSession()` → `stateCache.Remove(userID)`
  - [ ] Cap nhat `GetSessionMessages()` → lay tu GraphState.Messages

- [ ] **2.4** Xoa field `cacheMutex sync.RWMutex` va `stopCleanup chan struct{}`
  trong `internal/agent/usecase/new.go`

### PHASE 3 — RAG Upgrade

- [ ] **3.1** Tao `pkg/indexer/enricher.go`
  - [ ] Ham `EnrichTaskForEmbedding(title, content, tags, dueDate, timezone) string`
  - [ ] Ham `humanizeDueDate(due, now time.Time) string`

- [ ] **3.2** Tich hop Enricher vao create_bulk
  - [ ] Update `internal/task/usecase/create_bulk.go`
  - [ ] Thay `voyage.Embed(task.Content)` bang `voyage.Embed(indexer.Enrich(...))`

- [ ] **3.3** Tao `pkg/voyage/reranker.go`
  - [ ] Struct `Reranker` voi apiKey, model
  - [ ] Ham `NewReranker(apiKey) *Reranker`
  - [ ] Method `Rerank(ctx, query, documents, topK) ([]RerankResult, error)`
  - [ ] Struct `RerankResult` {Index, RelevanceScore}

- [ ] **3.4** Tao `pkg/qdrant/hybrid_search.go`
  - [ ] Struct `HybridSearcher`
  - [ ] Method `SearchHybrid(ctx, queryVector, queryText, limit) ([]SearchResult, error)`
  - [ ] Ham `reciprocalRankFusion(dense, sparse, limit) []SearchResult`
  - [ ] Fallback: neu sparse fail → tra ve dense only

- [ ] **3.5** Update `internal/task/usecase/answer_query.go`
  - [ ] Them `hybridSearcher` va `reranker` vao struct TaskUseCase
  - [ ] Thay dense search bang: Hybrid (top 20) → Rerank → top 5
  - [ ] Update `internal/task/usecase/new.go` inject cac dependency moi

---

## CHECKLIST TESTING

### Unit Tests — Graph Engine

- [ ] **T1.1** Test `GraphState`

  ```
  File: internal/agent/graph/state_test.go
  - NewGraphState: status=FINISHED, messages empty
  - IsExpired: false khi moi tao, true khi qua TTL
  - AppendMessage: them vao Messages va RecentTurns
  - AppendMessage: RecentTurns khong vuot qua 6
  ```

- [ ] **T1.2** Test `NodeAgent` (dung mock LLM)

  ```
  File: internal/agent/graph/node_agent_test.go
  - LLM tra ve FunctionCall → state.PendingTool set, Status=RUNNING
  - LLM tra ve FunctionCall nguy hiem → Status=WAITING_FOR_HUMAN
  - LLM tra ve text cau hoi → Status=WAITING_FOR_HUMAN
  - LLM tra ve text ket luan → Status=FINISHED
  - LLM tra ve empty → Status=ERROR, ErrEmptyResponse
  - LLM tra ve loi → Status=ERROR, error propagated
  ```

- [ ] **T1.3** Test `NodeExecuteTool` (dung mock registry)

  ```
  File: internal/agent/graph/node_tool_test.go
  - PendingTool nil → ErrNoPendingTool, Status=ERROR
  - Tool khong ton tai → append error result, Status=RUNNING
  - Tool thanh cong → append result, PendingTool=nil, Status=RUNNING
  - Tool loi → append error string, Status=RUNNING (khong panic)
  ```

- [ ] **T1.4** Test `Engine.Run`

  ```
  File: internal/agent/graph/engine_test.go
  - Status=FINISHED ban dau → khong goi NodeAgent
  - Status=RUNNING, LLM tra final text → 1 NodeAgent call, Status=FINISHED
  - Status=RUNNING, LLM goi tool → NodeAgent + NodeTool, Status=FINISHED
  - Status=RUNNING, LLM hoi user → NodeAgent, Status=WAITING_FOR_HUMAN, dung lai
  - MaxGraphSteps exceeded → Status=FINISHED, khong panic
  - Pause & Resume: WAITING → add user reply → RUNNING → FINISHED
  ```

- [ ] **T1.5** Test `Engine.GetLastResponse`

  ```
  - Messages rong → tra ve ""
  - Chi co function message → bo qua, tra ve ""
  - Co assistant text message → tra ve text do
  - Nhieu assistant messages → tra ve message cuoi
  ```

### Unit Tests — Usecase

- [ ] **T2.1** Test `ProcessQuery` voi Graph Engine

  ```
  File: internal/agent/usecase/process_query_test.go
  - Tin nhan moi: tao GraphState moi, chay engine, luu cache
  - Session con song: load tu cache, append message, chay engine
  - Session het han: tao GraphState moi (bo qua cache cu)
  - WAITING_FOR_HUMAN + user reply khong confirm → cancel, "Da huy"
  - WAITING_FOR_HUMAN + user reply confirm → tiep tuc chay tool
  - WAITING_FOR_HUMAN + LLM da hoi → resume reasoning
  - ClearSession: xoa khoi stateCache
  - GetSessionMessages: lay Messages tu state trong cache
  ```

### Unit Tests — RAG

- [ ] **T3.1** Test `EnrichTaskForEmbedding`

  ```
  File: pkg/indexer/enricher_test.go
  - Task voi deadline ngay mai → output chua "ngay mai"
  - Task voi deadline tuan nay → output chua "tuan nay"
  - Task qua han → output chua "qua han X ngay"
  - Task khong co deadline → khong chua dong Deadline
  - Task co tags → output chua tat ca tags
  - Task khong co tags → khong chua dong Tags
  ```

- [ ] **T3.2** Test `Reranker`

  ```
  File: pkg/voyage/reranker_test.go
  - Happy path: tra ve RerankResult sorted by score
  - API error → propagate error
  - Empty documents → xu ly dung
  - topK > len(documents) → tra ve tat ca
  (Dung mock HTTP server, khong goi API that)
  ```

- [ ] **T3.3** Test `reciprocalRankFusion`

  ```
  File: pkg/qdrant/hybrid_search_test.go
  - Result xuat hien ca 2 track → score cao nhat
  - Result chi o 1 track → score thap hon
  - Dense va sparse giong nhau → deduplicate dung
  - limit nho hon total results → tra ve dung so luong
  ```

### Integration Tests

- [ ] **T4.1** Test Pause & Resume (end-to-end)

  ```
  Scenario:
  1. User: "Tao lich hop du an SMAP"
  2. Engine goi NodeAgent → LLM hoi "May gio?"
  3. State = WAITING_FOR_HUMAN, luu vao cache
  4. User: "10h sang mai"
  5. Load state tu cache, status WAITING → RUNNING
  6. Engine tiep tuc, LLM co du info → goi tool create_calendar
  7. State = FINISHED, tra ve "Da tao lich thanh cong"
  ```

- [ ] **T4.2** Test Dangerous Operation Confirmation

  ```
  Scenario:
  1. User: "Xoa tat ca task da hoan thanh"
  2. Agent nhan ra delete_all_tasks → WAITING_FOR_HUMAN
  3. Bot hoi "Ban chac chan muon xoa?"
  4a. User: "ok" → RUNNING → tool thuc thi → xoa
  4b. User: "thoi" → cancel → "Da huy thao tac"
  ```

- [ ] **T4.3** Test RAG Pipeline

  ```
  - Tao 10 tasks voi cac deadline khac nhau
  - Query "deadline tuan nay" → chi tra ve tasks trong tuan
  - Query "PR #123" → tra ve dung PR 123, khong phai 124, 125
  - Query mot cau mo → reranker chon dung top 5
  ```

### Regression Tests

- [ ] **T5.1** Chay toan bo existing tests — khong duoc vo

  ```bash
  go test ./internal/... ./pkg/... -count=1 -timeout=60s
  ```

- [ ] **T5.2** Coverage check

  ```bash
  go test ./internal/... ./pkg/... -coverprofile=coverage.out
  go tool cover -func=coverage.out | grep total
  # Phai dat >= 80%
  ```

- [ ] **T5.3** Kiem tra khong co race condition

  ```bash
  go test ./internal/agent/... -race -count=3
  ```

---

## THU TU IMPLEMENT KHUYEN NGHI

```
Tuan 1: Phase 1 (Graph Engine)
  Ngay 1-2: state.go + errors.go + node_agent.go + node_tool.go
  Ngay 3:   engine.go
  Ngay 4-5: Unit tests T1.1 → T1.5

Tuan 2: Phase 2 (Ket noi vao He thong)
  Ngay 1-2: Update new.go + process_query.go + helpers.go
  Ngay 3-4: Unit tests T2.1
  Ngay 5:   Integration tests T4.1 + T4.2

Tuan 3: Phase 3 RAG Tang 1+2 (Enrichment + Hybrid)
  Ngay 1-2: pkg/indexer/enricher.go + tich hop vao create_bulk
  Ngay 3-4: pkg/qdrant/hybrid_search.go
  Ngay 5:   Tests T3.1 + T3.3

Tuan 4: Phase 3 RAG Tang 3+4 (Reranking + Compression)
  Ngay 1-2: pkg/voyage/reranker.go + tich hop vao answer_query
  Ngay 3:   Context compression trong GraphState
  Ngay 4-5: Tests T3.2 + T4.3 + Regression T5.1-T5.3
```

---

## FILE CHANGES SUMMARY

| File | Hanh dong | Ghi chu |
|------|-----------|---------|
| `internal/agent/graph/state.go` | TAO MOI | GraphState, GraphStatus |
| `internal/agent/graph/errors.go` | TAO MOI | Sentinel errors |
| `internal/agent/graph/node_agent.go` | TAO MOI | NodeAgent function |
| `internal/agent/graph/node_tool.go` | TAO MOI | NodeExecuteTool function |
| `internal/agent/graph/engine.go` | TAO MOI | Engine.Run() |
| `internal/agent/usecase/new.go` | SUA | LRU cache + Engine |
| `internal/agent/usecase/process_query.go` | REWRITE | Delegate to Engine |
| `internal/agent/usecase/helpers.go` | SUA | Bo sessionCache cu |
| `internal/agent/types.go` | GIU NGUYEN | SessionMemory con dung cho backward compat |
| `pkg/indexer/enricher.go` | TAO MOI | Contextual enrichment |
| `pkg/voyage/reranker.go` | TAO MOI | Rerank API |
| `pkg/qdrant/hybrid_search.go` | TAO MOI | Hybrid search + RRF |
| `internal/task/usecase/answer_query.go` | SUA | Dung hybrid + rerank |
| `internal/task/usecase/create_bulk.go` | SUA | Dung enriched embedding |

---

*Plan nay duoc tao dua tren phan tich truc tiep codebase V1.2 va master-plan V2.0.*
*Moi thay doi duoc thiet ke de backward-compatible: `agent.UseCase` interface giu nguyen,*
*Telegram handler khong can sua trong Phase 1 va 2.*
