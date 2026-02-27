## PHASE 3: RAG & AGENT TOOLS - SEMANTIC MEMORY & INTELLIGENCE

### ✅ Phase 2 Verification

**Phase 2 Deliverables Completed:**

- ✅ Telegram Bot webhook với background processing
- ✅ Gemini LLM integration với JSON sanitization
- ✅ Date Math parser với timezone support
- ✅ Memos repository với batch operations
- ✅ Google Calendar integration với RFC3339 format
- ✅ End-to-end task creation pipeline
- ✅ Convention compliance (Scope, helpers.go, error handling)
- ✅ All 3 critical fixes applied

**Ready for Phase 3:**

- Qdrant infrastructure running (Phase 1)
- Task creation pipeline stable (Phase 2)
- Foundation for semantic search

---

---

## 🚨 CRITICAL FIXES (Must Read Before Implementation)

### Fix 1: Qdrant ID Constraint ⚠️ HARD BLOCKER

**Problem:** Qdrant KHÔNG chấp nhận ID là arbitrary string. ID phải là:

- UUID chuẩn (e.g., `550e8400-e29b-41d4-a716-446655440000`)
- HOẶC unsigned integer (uint64)

**Impact:** Memos UID (Base58/short string) sẽ gây lỗi HTTP 400 Bad Request khi upsert.

**Solution:** Hash Memos ID thành UUID deterministic (UUID v5)

```go
import "github.com/google/uuid"

// memoIDToUUID converts Memos ID to UUID for Qdrant
func memoIDToUUID(memoID string) string {
 namespace := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
 return uuid.NewSHA1(namespace, []byte(memoID)).String()
}

// Usage in EmbedTask:
qdrantID := memoIDToUUID(task.ID)  // ✅ UUID string
point := pkgQdrant.Point{
 ID:     qdrantID,
 Payload: map[string]interface{}{
  "memo_id": task.ID,  // ✅ Store original ID in payload
 },
}
```

**Why UUID v5:**

- Deterministic: Same memoID → Same UUID
- Namespace-based: Collision-resistant
- Standard compliant

---

### Fix 2: Intent Detection Logic Flaw ⚠️ CRITICAL

**Problem:** Regex-based intent detection với `strings.HasPrefix("tìm")` gây false positive.

**Example:**

```
User: "Tìm hiểu cách tích hợp VNPay vào Golang, thứ 5 làm, P1"
System: Detects "Tìm" → Triggers search → Returns "Không tìm thấy"
Result: Task creation LOST!
```

**Solution:** Use explicit `/search` command instead of regex

```go
// ❌ BAD: Regex intent detection
if isSearchIntent(msg.Text) {  // "Tìm hiểu..." triggers this!
 return h.handleSearch(ctx, sc, msg)
}

// ✅ GOOD: Explicit command
if strings.HasPrefix(msg.Text, "/search ") {
 return h.handleSearch(ctx, sc, msg)
}
// Default: create task (safer)
return h.handleCreateTask(ctx, sc, msg)
```

**User Experience:**

- Create task: Just send text (default, safe)
- Search: `/search task SMAP đang block`

**Alternative (Phase 3 Advanced):** Use LLM as intent classifier

```go
// Ask LLM: "Is this a search query or task creation?"
intent := classifyIntent(ctx, msg.Text)
if intent == "search" {
 return h.handleSearch(ctx, sc, msg)
}
```

---

### Fix 3: Embedding Text Optimization ⚠️ IMPORTANT

**Problem:** Embedding full `task.Content` dilutes semantic density.

**Example:**

```markdown
# Fix login bug

**Description:**
User reported error when logging in with special characters...
[500 lines of stack trace]
[100 lines of debug logs]

#bug #priority/p0 #domain/backend
```

**Impact:**

- Embedding focuses on noise (logs) instead of signal (title, tags)
- Search accuracy drops significantly
- Embedding API cost increases

**Solution:** Extract only semantic-rich parts

```go
func buildEmbeddingText(task model.Task) string {
 var parts []string

 // 1. Extract title (first non-empty line)
 title := extractTitle(task.Content)
 parts = append(parts, title)

 // 2. Extract tags
 tags := extractTags(task.Content)
 parts = append(parts, strings.Join(tags, " "))

 // 3. Extract first 2-3 sentences (summary)
 summary := extractSummary(task.Content, 2)
 parts = append(parts, summary)

 result := strings.Join(parts, "\n")
 
 // Limit to 1000 chars
 if len(result) > 1000 {
  result = result[:1000]
 }

 return result
}
```

**Benefits:**

- Higher search accuracy (focus on semantic content)
- Lower embedding cost (shorter text)
- Faster embedding generation

---

### Implementation Checklist

**Before coding:**

- [ ] Add `github.com/google/uuid` to go.mod
- [ ] Understand UUID v5 deterministic hashing
- [ ] Review Qdrant ID constraints documentation

**During implementation:**

- [ ] Implement `memoIDToUUID()` helper
- [ ] Update `Point.ID` type to `interface{}`
- [ ] Store original `memo_id` in payload
- [ ] Change intent detection to `/search` command
- [ ] Implement `buildEmbeddingText()` optimization
- [ ] Update `/help` command documentation
- [ ] ⚠️ Add safe type assertion for `memo_id` in SearchTasks
- [ ] ⚠️ Add markdown code block stripping in buildEmbeddingText

**Testing:**

- [ ] Test: Memos ID → UUID conversion
- [ ] Test: Same memoID → Same UUID (deterministic)
- [ ] Test: "Tìm hiểu X" creates task (not search)
- [ ] Test: `/search X` triggers search
- [ ] Test: Embedding text < 1000 chars
- [ ] Test: Search accuracy with optimized embeddings
- [ ] Test: Type assertion handles missing payload gracefully
- [ ] Test: Code blocks stripped from embedding text

---

### Mục tiêu Phase 3

Nâng cấp hệ thống từ "reactive" (chỉ tạo task) sang "intelligent" (hiểu context, tìm kiếm semantic, gợi ý):

1. **Qdrant Integration:** Embed tasks vào vector database
2. **Semantic Search:** Tìm kiếm tasks theo ngữ nghĩa, không chỉ keyword
3. **RAG (Retrieval Augmented Generation):** Enhance LLM với historical context
4. **Agent Tools:** LLM có khả năng gọi tools (check calendar, search tasks)
5. **Auto-Sync:** Webhook từ Memos → auto-update Qdrant

**Không implement trong Phase 3:**

- ❌ Webhook automation từ Git (Phase 4)
- ❌ Regex checklist parser (Phase 4)
- ❌ Advanced agent orchestration (Phase 4)

---

### Kiến trúc Phase 3

```
┌─────────────┐
│   Telegram  │
│     Bot     │
└──────┬──────┘
       │ User Query: "Tìm task SMAP đang block"
       ▼
┌─────────────────────────────────────────────────┐
│         Golang Backend (Agent)                   │
│  ┌──────────────────────────────────────────┐  │
│  │  Agent Orchestrator                       │  │
│  │  - Decide: Create task or Search?        │  │
│  │  - Function Calling với LLM              │  │
│  └──────────┬───────────────────────────────┘  │
│             │                                    │
│             ▼                                    │
│  ┌──────────────────────────────────────────┐  │
│  │  Tool: search_tasks                       │  │
│  │  1. Generate embedding từ query          │  │
│  │  2. Query Qdrant (vector similarity)     │  │
│  │  3. Retrieve top-K Memos IDs             │  │
│  └──────────┬───────────────────────────────┘  │
│             │                                    │
│             ▼                                    │
│  ┌──────────────────────────────────────────┐  │
│  │  Memos Repository                         │  │
│  │  - Fetch full content by IDs             │  │
│  └──────────┬───────────────────────────────┘  │
│             │                                    │
│             ▼                                    │
│  ┌──────────────────────────────────────────┐  │
│  │  LLM (RAG)                                │  │
│  │  - Context: Retrieved tasks              │  │
│  │  - Answer user query                     │  │
│  └──────────────────────────────────────────┘  │
└─────────────────────────────────────────────────┘
       │
       ▼
┌─────────────┐
│   Qdrant    │
│  (Vectors)  │
└─────────────┘
```

**Write Flow (với Embedding):**

```
Create Task → Memos → Generate Embedding → Store in Qdrant
```

**Read Flow (Semantic Search):**

```
User Query → Generate Embedding → Qdrant Search → Fetch Memos → LLM Answer
```

---

### Cấu trúc Module Phase 3

```
pkg/
├── qdrant/                     # Qdrant client wrapper
│   ├── client.go               # HTTP client for Qdrant API
│   ├── collection.go           # Collection management
│   ├── point.go                # Point (vector) operations
│   ├── search.go               # Semantic search
│   ├── types.go                # Request/Response structs
│   └── interface.go            # Client interface
│
├── embedding/                  # Embedding service
│   ├── client.go               # Gemini embedding API
│   ├── types.go                # Embedding request/response
│   └── interface.go            # Embedder interface
│
internal/
├── task/
│   ├── usecase/
│   │   ├── create_bulk.go      # Updated: embed after create
│   │   ├── search.go           # NEW: Semantic search
│   │   ├── helpers.go          # Updated: add embedding helpers
│   │   └── types.go            # Updated: SearchInput/Output
│   │
│   ├── repository/
│   │   ├── interface.go        # Updated: add Search method
│   │   └── qdrant/             # NEW: Qdrant repository
│   │       ├── new.go
│   │       ├── task.go         # Embed, search, delete vectors
│   │       └── types.go
│   │
│   └── delivery/
│       └── telegram/
│           └── handler.go      # Updated: detect search intent
│
├── agent/                      # NEW: Agent orchestration
│   ├── interface.go            # Agent interface
│   ├── types.go                # Tool definitions
│   ├── orchestrator/
│   │   ├── new.go
│   │   ├── orchestrator.go     # Main agent loop
│   │   ├── tools.go            # Tool registry
│   │   └── function_calling.go # LLM function calling
│   │
│   └── tools/                  # Agent tools
│       ├── search_tasks.go     # Tool: semantic search
│       ├── check_calendar.go   # Tool: calendar conflicts
│       └── get_task_detail.go  # Tool: fetch task by ID
│
└── sync/                       # NEW: Webhook sync service
    ├── interface.go
    ├── handler.go              # Memos webhook handler
    └── processor.go            # Process memo updates
```

---

## Task Breakdown

### Task 3.1: Qdrant Client Implementation

**Mục tiêu:** Wrapper cho Qdrant HTTP API

**File:** `pkg/qdrant/client.go`

```go
package qdrant

import (
 "bytes"
 "context"
 "encoding/json"
 "fmt"
 "net/http"
)

// Client is the Qdrant HTTP API client.
type Client struct {
 baseURL    string
 httpClient *http.Client
}

// NewClient creates a new Qdrant client.
func NewClient(baseURL string) *Client {
 return &Client{
  baseURL:    baseURL,
  httpClient: &http.Client{},
 }
}

// CreateCollection creates a new collection with the given configuration.
func (c *Client) CreateCollection(ctx context.Context, req CreateCollectionRequest) error {
 url := fmt.Sprintf("%s/collections/%s", c.baseURL, req.Name)

 body, err := json.Marshal(req)
 if err != nil {
  return fmt.Errorf("failed to marshal request: %w", err)
 }

 httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewBuffer(body))
 if err != nil {
  return fmt.Errorf("failed to create request: %w", err)
 }
 httpReq.Header.Set("Content-Type", "application/json")

 resp, err := c.httpClient.Do(httpReq)
 if err != nil {
  return fmt.Errorf("failed to call qdrant API: %w", err)
 }
 defer resp.Body.Close()

 if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
  return fmt.Errorf("qdrant API error: %d", resp.StatusCode)
 }

 return nil
}

// UpsertPoints inserts or updates points (vectors) in a collection.
func (c *Client) UpsertPoints(ctx context.Context, collectionName string, req UpsertPointsRequest) error {
 url := fmt.Sprintf("%s/collections/%s/points", c.baseURL, collectionName)

 body, err := json.Marshal(req)
 if err != nil {
  return fmt.Errorf("failed to marshal request: %w", err)
 }

 httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewBuffer(body))
 if err != nil {
  return fmt.Errorf("failed to create request: %w", err)
 }
 httpReq.Header.Set("Content-Type", "application/json")

 resp, err := c.httpClient.Do(httpReq)
 if err != nil {
  return fmt.Errorf("failed to call qdrant API: %w", err)
 }
 defer resp.Body.Close()

 if resp.StatusCode != http.StatusOK {
  return fmt.Errorf("qdrant API error: %d", resp.StatusCode)
 }

 return nil
}

// SearchPoints performs semantic search in a collection.
func (c *Client) SearchPoints(ctx context.Context, collectionName string, req SearchRequest) (*SearchResponse, error) {
 url := fmt.Sprintf("%s/collections/%s/points/search", c.baseURL, collectionName)

 body, err := json.Marshal(req)
 if err != nil {
  return nil, fmt.Errorf("failed to marshal request: %w", err)
 }

 httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
 if err != nil {
  return nil, fmt.Errorf("failed to create request: %w", err)
 }
 httpReq.Header.Set("Content-Type", "application/json")

 resp, err := c.httpClient.Do(httpReq)
 if err != nil {
  return nil, fmt.Errorf("failed to call qdrant API: %w", err)
 }
 defer resp.Body.Close()

 if resp.StatusCode != http.StatusOK {
  return nil, fmt.Errorf("qdrant API error: %d", resp.StatusCode)
 }

 var result SearchResponse
 if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
  return nil, fmt.Errorf("failed to decode response: %w", err)
 }

 return &result, nil
}

// DeletePoints deletes points by IDs.
func (c *Client) DeletePoints(ctx context.Context, collectionName string, ids []string) error {
 url := fmt.Sprintf("%s/collections/%s/points/delete", c.baseURL, collectionName)

 req := DeletePointsRequest{
  Points: ids,
 }

 body, err := json.Marshal(req)
 if err != nil {
  return fmt.Errorf("failed to marshal request: %w", err)
 }

 httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
 if err != nil {
  return fmt.Errorf("failed to create request: %w", err)
 }
 httpReq.Header.Set("Content-Type", "application/json")

 resp, err := c.httpClient.Do(httpReq)
 if err != nil {
  return fmt.Errorf("failed to call qdrant API: %w", err)
 }
 defer resp.Body.Close()

 if resp.StatusCode != http.StatusOK {
  return fmt.Errorf("qdrant API error: %d", resp.StatusCode)
 }

 return nil
}
```

**File:** `pkg/qdrant/types.go`

```go
package qdrant

// CreateCollectionRequest defines the schema for creating a collection.
type CreateCollectionRequest struct {
 Name    string          `json:"-"` // Collection name (in URL)
 Vectors VectorConfig    `json:"vectors"`
}

// VectorConfig defines vector dimension and distance metric.
type VectorConfig struct {
 Size     int    `json:"size"`     // Vector dimension (e.g., 768 for Gemini)
 Distance string `json:"distance"` // "Cosine", "Euclid", "Dot"
}

// Point represents a vector with payload (metadata).
// ⚠️ CRITICAL: Qdrant requires ID to be UUID or uint64, NOT arbitrary string!
type Point struct {
 ID      interface{}            `json:"id"`      // UUID string or uint64 (NOT arbitrary string!)
 Vector  []float32              `json:"vector"`  // Embedding vector
 Payload map[string]interface{} `json:"payload"` // Metadata (memo_id, title, tags, etc.)
}

// UpsertPointsRequest is the request to insert/update points.
type UpsertPointsRequest struct {
 Points []Point `json:"points"`
}

// SearchRequest is the request for semantic search.
type SearchRequest struct {
 Vector      []float32              `json:"vector"`       // Query vector
 Limit       int                    `json:"limit"`        // Top-K results
 WithPayload bool                   `json:"with_payload"` // Include metadata
 Filter      map[string]interface{} `json:"filter,omitempty"` // Optional filters
}

// SearchResponse contains search results.
type SearchResponse struct {
 Result []ScoredPoint `json:"result"`
}

// ScoredPoint is a search result with similarity score.
type ScoredPoint struct {
 ID      string                 `json:"id"`
 Score   float64                `json:"score"`
 Payload map[string]interface{} `json:"payload"`
}

// DeletePointsRequest is the request to delete points.
type DeletePointsRequest struct {
 Points []string `json:"points"`
}
```

---

### Task 3.2: Embedding Service

**Mục tiêu:** Generate embeddings từ text sử dụng Gemini Embedding API

**File:** `pkg/embedding/client.go`

```go
package embedding

import (
 "bytes"
 "context"
 "encoding/json"
 "fmt"
 "net/http"
)

const (
 defaultModel = "text-embedding-004" // Gemini embedding model
 vectorSize   = 768                  // Dimension của embedding
)

// Client is the embedding service client.
type Client struct {
 apiKey     string
 apiURL     string
 model      string
 httpClient *http.Client
}

// NewClient creates a new embedding client.
func NewClient(apiKey string) *Client {
 return &Client{
  apiKey:     apiKey,
  apiURL:     "https://generativelanguage.googleapis.com/v1beta",
  model:      defaultModel,
  httpClient: &http.Client{},
 }
}

// Embed generates an embedding vector from text.
func (c *Client) Embed(ctx context.Context, text string) ([]float32, error) {
 url := fmt.Sprintf("%s/models/%s:embedContent?key=%s", c.apiURL, c.model, c.apiKey)

 req := EmbedRequest{
  Content: Content{
   Parts: []Part{
    {Text: text},
   },
  },
 }

 body, err := json.Marshal(req)
 if err != nil {
  return nil, fmt.Errorf("failed to marshal request: %w", err)
 }

 httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
 if err != nil {
  return nil, fmt.Errorf("failed to create request: %w", err)
 }
 httpReq.Header.Set("Content-Type", "application/json")

 resp, err := c.httpClient.Do(httpReq)
 if err != nil {
  return nil, fmt.Errorf("failed to call embedding API: %w", err)
 }
 defer resp.Body.Close()

 if resp.StatusCode != http.StatusOK {
  return nil, fmt.Errorf("embedding API error: %d", resp.StatusCode)
 }

 var result EmbedResponse
 if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
  return nil, fmt.Errorf("failed to decode response: %w", err)
 }

 if len(result.Embedding.Values) == 0 {
  return nil, fmt.Errorf("empty embedding returned")
 }

 return result.Embedding.Values, nil
}

// EmbedBatch generates embeddings for multiple texts.
func (c *Client) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
 embeddings := make([][]float32, 0, len(texts))

 for _, text := range texts {
  embedding, err := c.Embed(ctx, text)
  if err != nil {
   return nil, fmt.Errorf("failed to embed text %q: %w", text, err)
  }
  embeddings = append(embeddings, embedding)
 }

 return embeddings, nil
}

// GetVectorSize returns the dimension of embeddings.
func (c *Client) GetVectorSize() int {
 return vectorSize
}
```

**File:** `pkg/embedding/types.go`

```go
package embedding

// EmbedRequest is the request to generate embeddings.
type EmbedRequest struct {
 Content Content `json:"content"`
}

// Content wraps the text to embed.
type Content struct {
 Parts []Part `json:"parts"`
}

// Part contains the text.
type Part struct {
 Text string `json:"text"`
}

// EmbedResponse contains the embedding vector.
type EmbedResponse struct {
 Embedding Embedding `json:"embedding"`
}

// Embedding contains the vector values.
type Embedding struct {
 Values []float32 `json:"values"`
}
```

---

### Task 3.3: Qdrant Repository

**Mục tiêu:** Repository layer cho Qdrant operations

**File:** `internal/task/repository/qdrant/task.go`

```go
package qdrant

import (
 "context"
 "fmt"
 "regexp"
 "strings"

 "github.com/google/uuid" // ✅ NEW: For UUID generation

 "autonomous-task-management/internal/model"
 "autonomous-task-management/internal/task/repository"
 "autonomous-task-management/pkg/embedding"
 pkgLog "autonomous-task-management/pkg/log"
 pkgQdrant "autonomous-task-management/pkg/qdrant"
)

type implRepository struct {
 client         *pkgQdrant.Client
 embedder       *embedding.Client
 collectionName string
 l              pkgLog.Logger
}

// New creates a new Qdrant repository.
func New(client *pkgQdrant.Client, embedder *embedding.Client, collectionName string, l pkgLog.Logger) repository.VectorRepository {
 return &implRepository{
  client:         client,
  embedder:       embedder,
  collectionName: collectionName,
  l:              l,
 }
}

// memoIDToUUID converts Memos ID (arbitrary string) to UUID for Qdrant.
// Qdrant requires ID to be UUID or uint64, NOT arbitrary string.
// We use deterministic UUID v5 (namespace + name) to ensure same ID for same memo.
func memoIDToUUID(memoID string) string {
 // Use UUID v5 with a custom namespace
 // This ensures: same memoID → same UUID (deterministic)
 namespace := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8") // DNS namespace
 return uuid.NewSHA1(namespace, []byte(memoID)).String()
}

// EmbedTask generates embedding and stores in Qdrant.
func (r *implRepository) EmbedTask(ctx context.Context, task model.Task) error {
 // Build text to embed: title + tags + summary (NOT full content)
 textToEmbed := buildEmbeddingText(task)

 // Generate embedding
 vector, err := r.embedder.Embed(ctx, textToEmbed)
 if err != nil {
  r.l.Errorf(ctx, "qdrant repository: failed to generate embedding: %v", err)
  return fmt.Errorf("failed to generate embedding: %w", err)
 }

 // ✅ CRITICAL FIX: Convert Memos ID to UUID for Qdrant
 // Qdrant requires ID to be UUID or uint64, NOT arbitrary string
 qdrantID := memoIDToUUID(task.ID)

 // Create point
 point := pkgQdrant.Point{
  ID:     qdrantID, // ✅ UUID string
  Vector: vector,
  Payload: map[string]interface{}{
   "memo_id":     task.ID,        // ✅ Store original Memos ID in payload
   "memo_url":    task.MemoURL,
   "content":     task.Content,
   "create_time": task.CreateTime,
   "update_time": task.UpdateTime,
  },
 }

 // Upsert to Qdrant
 req := pkgQdrant.UpsertPointsRequest{
  Points: []pkgQdrant.Point{point},
 }

 if err := r.client.UpsertPoints(ctx, r.collectionName, req); err != nil {
  r.l.Errorf(ctx, "qdrant repository: failed to upsert point: %v", err)
  return fmt.Errorf("failed to upsert point: %w", err)
 }

 r.l.Infof(ctx, "qdrant repository: embedded task %s (qdrant_id=%s)", task.ID, qdrantID)
 return nil
}

// SearchTasks performs semantic search.
func (r *implRepository) SearchTasks(ctx context.Context, opt repository.SearchTasksOptions) ([]repository.SearchResult, error) {
 // Generate query embedding
 queryVector, err := r.embedder.Embed(ctx, opt.Query)
 if err != nil {
  r.l.Errorf(ctx, "qdrant repository: failed to generate query embedding: %v", err)
  return nil, fmt.Errorf("failed to generate query embedding: %w", err)
 }

 // Build search request
 searchReq := pkgQdrant.SearchRequest{
  Vector:      queryVector,
  Limit:       opt.Limit,
  WithPayload: true, // ✅ CRITICAL: Need payload to get original memo_id
 }

 // Add filters if provided
 if len(opt.Tags) > 0 {
  // TODO: Implement tag filtering
 }

 // Search in Qdrant
 resp, err := r.client.SearchPoints(ctx, r.collectionName, searchReq)
 if err != nil {
  r.l.Errorf(ctx, "qdrant repository: failed to search: %v", err)
  return nil, fmt.Errorf("failed to search: %w", err)
 }

 // Convert to SearchResult
 // ✅ CRITICAL: Extract memo_id from payload (NOT from Qdrant ID)
 results := make([]repository.SearchResult, 0, len(resp.Result))
 for _, scored := range resp.Result {
  // ⚠️ NITPICK FIX: Safe type assertion with detailed error logging
  // Get original Memos ID from payload
  memoIDRaw, exists := scored.Payload["memo_id"]
  if !exists {
   r.l.Errorf(ctx, "qdrant repository: memo_id missing in payload for point %v, payload: %+v", 
    scored.ID, scored.Payload)
   continue
  }

  memoID, ok := memoIDRaw.(string)
  if !ok {
   r.l.Errorf(ctx, "qdrant repository: memo_id type assertion failed for point %v, got type %T, value: %v", 
    scored.ID, memoIDRaw, memoIDRaw)
   continue
  }

  results = append(results, repository.SearchResult{
   MemoID:  memoID, // ✅ Use original Memos ID, not Qdrant UUID
   Score:   scored.Score,
   Payload: scored.Payload,
  })
 }

 r.l.Infof(ctx, "qdrant repository: found %d results for query %q", len(results), opt.Query)
 return results, nil
}

// DeleteTask removes a task from Qdrant.
func (r *implRepository) DeleteTask(ctx context.Context, taskID string) error {
 // ✅ Convert Memos ID to UUID
 qdrantID := memoIDToUUID(taskID)

 if err := r.client.DeletePoints(ctx, r.collectionName, []string{qdrantID}); err != nil {
  r.l.Errorf(ctx, "qdrant repository: failed to delete point: %v", err)
  return fmt.Errorf("failed to delete point: %w", err)
 }

 r.l.Infof(ctx, "qdrant repository: deleted task %s (qdrant_id=%s)", taskID, qdrantID)
 return nil
}

// memoIDToUUID converts Memos ID (arbitrary string) to UUID for Qdrant.
// Qdrant requires ID to be UUID or uint64, NOT arbitrary string.
// We use deterministic UUID v5 (namespace + name) to ensure same ID for same memo.
func memoIDToUUID(memoID string) string {
 // Use UUID v5 with a custom namespace
 // This ensures: same memoID → same UUID (deterministic)
 namespace := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8") // DNS namespace
 return uuid.NewSHA1(namespace, []byte(memoID)).String()
}

// buildEmbeddingText constructs optimized text for embedding from task.
// ⚠️ OPTIMIZATION: Embed only title + tags + summary, NOT full content.
// Full content dilutes semantic density and reduces search accuracy.
func buildEmbeddingText(task model.Task) string {
 var parts []string

 // ⚠️ NITPICK FIX: Strip markdown code blocks first
 // Prevents code snippets from polluting semantic content
 content := stripMarkdownCodeBlocks(task.Content)
 
 // Extract title (first non-empty line, remove markdown)
 lines := strings.Split(content, "\n")
 for _, line := range lines {
  line = strings.TrimSpace(line)
  if line != "" && !strings.HasPrefix(line, "#") {
   // Remove markdown formatting
   title := strings.ReplaceAll(line, "**", "")
   title = strings.ReplaceAll(title, "*", "")
   parts = append(parts, title)
   break
  }
 }

 // Extract tags (lines starting with #)
 var tags []string
 for _, line := range lines {
  line = strings.TrimSpace(line)
  if strings.HasPrefix(line, "#") {
   tags = append(tags, line)
  }
 }
 if len(tags) > 0 {
  parts = append(parts, strings.Join(tags, " "))
 }

 // Extract first 2-3 sentences as summary (skip title line)
 var summaryLines []string
 skipFirst := true
 sentenceCount := 0
 for _, line := range lines {
  line = strings.TrimSpace(line)
  if line == "" || strings.HasPrefix(line, "#") {
   continue
  }
  if skipFirst {
   skipFirst = false
   continue
  }
  summaryLines = append(summaryLines, line)
  // Count sentences (rough approximation)
  sentenceCount += strings.Count(line, ".") + strings.Count(line, "!") + strings.Count(line, "?")
  if sentenceCount >= 2 {
   break
  }
 }
 if len(summaryLines) > 0 {
  parts = append(parts, strings.Join(summaryLines, " "))
 }

 // Combine: title + tags + summary
 result := strings.Join(parts, "\n")
 
 // Limit to 1000 chars to avoid embedding API limits
 if len(result) > 1000 {
  result = result[:1000]
 }

 return result
}

// stripMarkdownCodeBlocks removes code blocks (```...```) from text.
// ⚠️ NITPICK FIX: Prevents code snippets from polluting embeddings.
func stripMarkdownCodeBlocks(text string) string {
 // Remove code blocks: ```language\n...\n``` or ```\n...\n```
 re := regexp.MustCompile("(?s)```[a-z]*\\n.*?\\n```")
 return re.ReplaceAllString(text, "")
}
```

**File:** `internal/task/repository/interface.go` (update)

```go
package repository

import (
 "context"
 "autonomous-task-management/internal/model"
)

// MemosRepository handles Memos CRUD operations.
type MemosRepository interface {
 CreateTask(ctx context.Context, opt CreateTaskOptions) (model.Task, error)
 CreateTasksBatch(ctx context.Context, opts []CreateTaskOptions) ([]model.Task, error)
 GetTask(ctx context.Context, id string) (model.Task, error)
 ListTasks(ctx context.Context, opt ListTasksOptions) ([]model.Task, error)
}

// VectorRepository handles vector operations (Qdrant).
type VectorRepository interface {
 EmbedTask(ctx context.Context, task model.Task) error
 SearchTasks(ctx context.Context, opt SearchTasksOptions) ([]SearchResult, error)
 DeleteTask(ctx context.Context, taskID string) error
}

// SearchTasksOptions defines search parameters.
type SearchTasksOptions struct {
 Query string   // Natural language query
 Limit int      // Top-K results
 Tags  []string // Filter by tags (optional)
}

// SearchResult represents a semantic search result.
type SearchResult struct {
 MemoID  string
 Score   float64
 Payload map[string]interface{}
}
```

---

### Task 3.4: Update Task UseCase - Add Embedding

**Mục tiêu:** Sau khi tạo task trong Memos, tự động embed vào Qdrant

**File:** `internal/task/usecase/create_bulk.go` (update)

```go
// CreateBulk parses raw text, creates Memos tasks, embeds to Qdrant, and creates Calendar events.
func (uc *implUseCase) CreateBulk(ctx context.Context, sc model.Scope, input task.CreateBulkInput) (task.CreateBulkOutput, error) {
 if strings.TrimSpace(input.RawText) == "" {
  return task.CreateBulkOutput{}, task.ErrEmptyInput
 }

 uc.l.Infof(ctx, "CreateBulk: user=%s input_length=%d", sc.UserID, len(input.RawText))

 // Step 1: Parse tasks from raw text via LLM
 parsedTasks, err := uc.parseInputWithLLM(ctx, input.RawText)
 if err != nil {
  return task.CreateBulkOutput{}, fmt.Errorf("failed to parse input with LLM: %w", err)
 }

 if len(parsedTasks) == 0 {
  return task.CreateBulkOutput{}, task.ErrNoTasksParsed
 }

 uc.l.Infof(ctx, "CreateBulk: LLM parsed %d tasks", len(parsedTasks))

 // Step 2: Resolve relative dates to absolute times
 tasksWithDates := uc.resolveDueDates(parsedTasks)

 // Step 3: Create each task in Memos
 createdTasks := make([]task.CreatedTask, 0, len(tasksWithDates))

 for _, t := range tasksWithDates {
  // Build markdown content
  content := buildMarkdownContent(t)

  // Create in Memos
  memoTask, memoErr := uc.memosRepo.CreateTask(ctx, repository.CreateTaskOptions{
   Content:    content,
   Tags:       allTags(t),
   Visibility: "PRIVATE",
  })
  if memoErr != nil {
   uc.l.Errorf(ctx, "CreateBulk: failed to create Memos task %q: %v", t.Title, memoErr)
   continue
  }

  // ✅ NEW: Embed task to Qdrant (non-blocking on failure)
  if uc.vectorRepo != nil {
   if embedErr := uc.vectorRepo.EmbedTask(ctx, memoTask); embedErr != nil {
    uc.l.Warnf(ctx, "CreateBulk: failed to embed task %s to Qdrant: %v", memoTask.ID, embedErr)
    // Don't fail the whole operation
   }
  }

  // Attempt to create Google Calendar event (non-blocking on failure)
  calendarLink := uc.tryCreateCalendarEvent(ctx, t, memoTask)

  createdTasks = append(createdTasks, task.CreatedTask{
   MemoID:       memoTask.ID,
   MemoURL:      memoTask.MemoURL,
   CalendarLink: calendarLink,
   Title:        t.Title,
  })

  uc.l.Infof(ctx, "CreateBulk: created task %q memoID=%s", t.Title, memoTask.ID)
 }

 return task.CreateBulkOutput{
  Tasks:     createdTasks,
  TaskCount: len(createdTasks),
 }, nil
}
```

**File:** `internal/task/usecase/new.go` (update)

```go
type implUseCase struct {
 memosRepo  repository.MemosRepository
 vectorRepo repository.VectorRepository // ✅ NEW
 llm        *gemini.Client
 calendar   *gcalendar.Client
 dateParser *datemath.Parser
 l          pkgLog.Logger

 timezone   string
 memoBaseURL string
}

func New(
 l pkgLog.Logger,
 llm *gemini.Client,
 calendar *gcalendar.Client,
 memosRepo repository.MemosRepository,
 vectorRepo repository.VectorRepository, // ✅ NEW
 dateParser *datemath.Parser,
 timezone string,
 memoBaseURL string,
) task.UseCase {
 return &implUseCase{
  memosRepo:   memosRepo,
  vectorRepo:  vectorRepo, // ✅ NEW
  llm:         llm,
  calendar:    calendar,
  dateParser:  dateParser,
  l:           l,
  timezone:    timezone,
  memoBaseURL: memoBaseURL,
 }
}
```

---

### Task 3.5: Implement Semantic Search UseCase

**Mục tiêu:** Search tasks bằng natural language query

**File:** `internal/task/usecase/search.go` (NEW)

```go
package usecase

import (
 "context"
 "fmt"

 "autonomous-task-management/internal/model"
 "autonomous-task-management/internal/task"
 "autonomous-task-management/internal/task/repository"
)

// Search performs semantic search on tasks.
func (uc *implUseCase) Search(ctx context.Context, sc model.Scope, input task.SearchInput) (task.SearchOutput, error) {
 if input.Query == "" {
  return task.SearchOutput{}, task.ErrEmptyQuery
 }

 uc.l.Infof(ctx, "Search: user=%s query=%q", sc.UserID, input.Query)

 // Default limit
 limit := input.Limit
 if limit <= 0 {
  limit = 10
 }

 // Search in Qdrant
 searchResults, err := uc.vectorRepo.SearchTasks(ctx, repository.SearchTasksOptions{
  Query: input.Query,
  Limit: limit,
  Tags:  input.Tags,
 })
 if err != nil {
  uc.l.Errorf(ctx, "Search: failed to search in Qdrant: %v", err)
  return task.SearchOutput{}, fmt.Errorf("failed to search: %w", err)
 }

 if len(searchResults) == 0 {
  uc.l.Infof(ctx, "Search: no results found for query %q", input.Query)
  return task.SearchOutput{
   Results: []task.SearchResultItem{},
   Count:   0,
  }, nil
 }

 // Fetch full task details from Memos
 results := make([]task.SearchResultItem, 0, len(searchResults))
 for _, sr := range searchResults {
  // Fetch from Memos
  memoTask, err := uc.memosRepo.GetTask(ctx, sr.MemoID)
  if err != nil {
   uc.l.Warnf(ctx, "Search: failed to fetch task %s from Memos: %v", sr.MemoID, err)
   continue
  }

  results = append(results, task.SearchResultItem{
   MemoID:   memoTask.ID,
   MemoURL:  memoTask.MemoURL,
   Content:  memoTask.Content,
   Score:    sr.Score,
  })
 }

 uc.l.Infof(ctx, "Search: found %d results", len(results))

 return task.SearchOutput{
  Results: results,
  Count:   len(results),
 }, nil
}
```

**File:** `internal/task/types.go` (update)

```go
// SearchInput is the input for semantic search.
type SearchInput struct {
 Query string   // Natural language query
 Limit int      // Max results (default 10)
 Tags  []string // Filter by tags (optional)
}

// SearchResultItem represents a single search result.
type SearchResultItem struct {
 MemoID  string
 MemoURL string
 Content string
 Score   float64 // Similarity score (0-1)
}

// SearchOutput is the result of semantic search.
type SearchOutput struct {
 Results []SearchResultItem
 Count   int
}
```

**File:** `internal/task/interface.go` (update)

```go
package task

import (
 "context"
 "autonomous-task-management/internal/model"
)

// UseCase defines the business logic interface for the task domain.
type UseCase interface {
 // CreateBulk parses raw text from the user, creates tasks in Memos,
 // embeds to Qdrant, and schedules events in Google Calendar.
 CreateBulk(ctx context.Context, sc model.Scope, input CreateBulkInput) (CreateBulkOutput, error)

 // Search performs semantic search on tasks.
 Search(ctx context.Context, sc model.Scope, input SearchInput) (SearchOutput, error)
}
```

**File:** `internal/task/errors.go` (update)

```go
package task

import "errors"

var (
 ErrEmptyInput     = errors.New("input text is empty")
 ErrNoTasksParsed  = errors.New("no tasks parsed from input")
 ErrMemoCreate     = errors.New("failed to create memo")
 ErrEmptyQuery     = errors.New("search query is empty") // ✅ NEW
)
```

---

### Task 3.6: Update Telegram Handler - Detect Search Intent

**Mục tiêu:** Phân biệt giữa "create task" và "search task" intent

**File:** `internal/task/delivery/telegram/handler.go` (update)

```go
func (h *handler) processMessage(ctx context.Context, msg *pkgTelegram.Message) error {
 if msg.Text == "" {
  return nil
 }

 // Handle built-in commands
 if msg.Text == "/start" {
  return h.bot.SendMessage(msg.Chat.ID, "Chào mừng đến với Autonomous Task Management!\n\n"+
   "Bạn có thể:\n"+
   "- Tạo task: gửi mô tả công việc (mặc định)\n"+
   "- Tìm task: dùng lệnh /search <query>\n"+
   "- Ví dụ: /search task SMAP đang block")
 }

 if msg.Text == "/help" {
  return h.bot.SendMessage(msg.Chat.ID, "📖 Hướng dẫn sử dụng:\n\n"+
   "**Tạo task:**\n"+
   "Finish SMAP report by tomorrow\n"+
   "Review code today p1\n"+
   "Tìm hiểu cách tích hợp VNPay (sẽ tạo task, KHÔNG search)\n\n"+
   "**Tìm task:**\n"+
   "/search task SMAP đang block\n"+
   "/search ahamove high priority\n"+
   "/search tasks due this week")
 }

 // Build scope from Telegram user
 sc := model.Scope{
  UserID: fmt.Sprintf("telegram_%d", msg.From.ID),
 }

 // ✅ CRITICAL FIX: Use explicit command instead of regex intent detection
 // Problem: "Tìm hiểu cách tích hợp VNPay" would be detected as search intent
 // Solution: Use /search command for explicit search, default to create task
 if strings.HasPrefix(msg.Text, "/search ") {
  return h.handleSearch(ctx, sc, msg)
 }

 // Default: create task (safer than regex intent detection)
 return h.handleCreateTask(ctx, sc, msg)
}

// handleSearch processes search requests.
func (h *handler) handleSearch(ctx context.Context, sc model.Scope, msg *pkgTelegram.Message) error {
 // Extract query (remove /search command)
 query := strings.TrimSpace(strings.TrimPrefix(msg.Text, "/search"))

 input := task.SearchInput{
  Query: query,
  Limit: 5, // Top 5 results
 }

 output, err := h.uc.Search(ctx, sc, input)
 if err != nil {
  h.l.Errorf(ctx, "Search failed: %v", err)
  return h.bot.SendMessage(msg.Chat.ID, "Có lỗi khi tìm kiếm. Vui lòng thử lại.")
 }

 if output.Count == 0 {
  return h.bot.SendMessage(msg.Chat.ID, "Không tìm thấy task nào phù hợp.")
 }

 // Format results
 response := fmt.Sprintf("🔍 Tìm thấy %d task:\n\n", output.Count)
 for i, result := range output.Results {
  // Extract title from content (first line)
  title := extractTitle(result.Content)
  score := int(result.Score * 100)

  response += fmt.Sprintf("%d. %s\n", i+1, title)
  response += fmt.Sprintf("   📊 Độ phù hợp: %d%%\n", score)
  response += fmt.Sprintf("   🔗 [Xem chi tiết](%s)\n\n", result.MemoURL)
 }

 return h.bot.SendMessageWithMode(msg.Chat.ID, response, "Markdown")
}

// handleCreateTask processes task creation requests.
func (h *handler) handleCreateTask(ctx context.Context, sc model.Scope, msg *pkgTelegram.Message) error {
 input := task.CreateBulkInput{
  RawText:        msg.Text,
  TelegramChatID: msg.Chat.ID,
 }

 output, err := h.uc.CreateBulk(ctx, sc, input)
 if err != nil {
  h.l.Errorf(ctx, "CreateBulk failed: %v", err)
  return h.bot.SendMessage(msg.Chat.ID, "Có lỗi khi tạo task. Vui lòng thử lại.")
 }

 // Format success message
 response := fmt.Sprintf("✅ Đã tạo %d task thành công!\n\n", output.TaskCount)
 for i, t := range output.Tasks {
  response += fmt.Sprintf("%d. %s\n", i+1, t.Title)
  response += fmt.Sprintf("   🔗 [Xem chi tiết](%s)\n", t.MemoURL)
  if t.CalendarLink != "" {
   response += fmt.Sprintf("   📅 [Lịch](%s)\n", t.CalendarLink)
  }
  response += "\n"
 }

 return h.bot.SendMessageWithMode(msg.Chat.ID, response, "Markdown")
}

// extractTitle extracts the first line from markdown content.
func extractTitle(content string) string {
 lines := strings.Split(content, "\n")
 for _, line := range lines {
  line = strings.TrimSpace(line)
  if line != "" && !strings.HasPrefix(line, "#") {
   // Remove markdown formatting
   line = strings.ReplaceAll(line, "**", "")
   line = strings.ReplaceAll(line, "*", "")
   if len(line) > 100 {
    return line[:100] + "..."
   }
   return line
  }
 }
 return "Untitled"
}
```

**File:** `pkg/telegram/bot.go` (update - add SendMessageWithMode)

```go
// SendMessageWithMode sends a message with parse mode (Markdown, HTML).
func (b *Bot) SendMessageWithMode(chatID int64, text string, parseMode string) error {
 url := fmt.Sprintf("%s/sendMessage", b.apiURL)
 payload := map[string]interface{}{
  "chat_id":    chatID,
  "text":       text,
  "parse_mode": parseMode,
 }

 body, _ := json.Marshal(payload)
 resp, err := b.httpClient.Post(url, "application/json", bytes.NewBuffer(body))
 if err != nil {
  return fmt.Errorf("failed to send message: %w", err)
 }
 defer resp.Body.Close()

 if resp.StatusCode != http.StatusOK {
  return fmt.Errorf("telegram API error: %d", resp.StatusCode)
 }

 return nil
}
```

---

### Task 3.7: Agent Tools Framework (Foundation)

**Mục tiêu:** Chuẩn bị framework cho LLM Function Calling (Phase 3 advanced)

**File:** `internal/agent/types.go` (NEW)

```go
package agent

import "context"

// Tool represents an agent tool that can be called by LLM.
type Tool interface {
 // Name returns the tool name (used in function calling).
 Name() string

 // Description returns what the tool does (for LLM).
 Description() string

 // Parameters returns JSON schema for tool parameters.
 Parameters() map[string]interface{}

 // Execute runs the tool with given parameters.
 Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)
}

// ToolRegistry manages available tools.
type ToolRegistry struct {
 tools map[string]Tool
}

// NewToolRegistry creates a new tool registry.
func NewToolRegistry() *ToolRegistry {
 return &ToolRegistry{
  tools: make(map[string]Tool),
 }
}

// Register adds a tool to the registry.
func (r *ToolRegistry) Register(tool Tool) {
 r.tools[tool.Name()] = tool
}

// Get retrieves a tool by name.
func (r *ToolRegistry) Get(name string) (Tool, bool) {
 tool, ok := r.tools[name]
 return tool, ok
}

// List returns all registered tools.
func (r *ToolRegistry) List() []Tool {
 tools := make([]Tool, 0, len(r.tools))
 for _, tool := range r.tools {
  tools = append(tools, tool)
 }
 return tools
}

// ToFunctionDefinitions converts tools to Gemini function calling format.
func (r *ToolRegistry) ToFunctionDefinitions() []FunctionDefinition {
 defs := make([]FunctionDefinition, 0, len(r.tools))
 for _, tool := range r.tools {
  defs = append(defs, FunctionDefinition{
   Name:        tool.Name(),
   Description: tool.Description(),
   Parameters:  tool.Parameters(),
  })
 }
 return defs
}

// FunctionDefinition is the schema for Gemini function calling.
type FunctionDefinition struct {
 Name        string                 `json:"name"`
 Description string                 `json:"description"`
 Parameters  map[string]interface{} `json:"parameters"`
}
```

**File:** `internal/agent/tools/search_tasks.go` (NEW - Example Tool)

```go
package tools

import (
 "context"
 "fmt"

 "autonomous-task-management/internal/agent"
 "autonomous-task-management/internal/model"
 "autonomous-task-management/internal/task"
)

// SearchTasksTool implements semantic search tool.
type SearchTasksTool struct {
 uc task.UseCase
}

// NewSearchTasksTool creates a new search tasks tool.
func NewSearchTasksTool(uc task.UseCase) agent.Tool {
 return &SearchTasksTool{uc: uc}
}

func (t *SearchTasksTool) Name() string {
 return "search_tasks"
}

func (t *SearchTasksTool) Description() string {
 return "Search for tasks using natural language query. Returns relevant tasks with similarity scores."
}

func (t *SearchTasksTool) Parameters() map[string]interface{} {
 return map[string]interface{}{
  "type": "object",
  "properties": map[string]interface{}{
   "query": map[string]interface{}{
    "type":        "string",
    "description": "Natural language search query",
   },
   "limit": map[string]interface{}{
    "type":        "integer",
    "description": "Maximum number of results (default 10)",
   },
  },
  "required": []string{"query"},
 }
}

func (t *SearchTasksTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
 // Extract parameters
 query, ok := params["query"].(string)
 if !ok || query == "" {
  return nil, fmt.Errorf("query parameter is required")
 }

 limit := 10
 if l, ok := params["limit"].(float64); ok {
  limit = int(l)
 }

 // Execute search
 sc := model.Scope{UserID: "agent"} // TODO: Get from context
 output, err := t.uc.Search(ctx, sc, task.SearchInput{
  Query: query,
  Limit: limit,
 })
 if err != nil {
  return nil, fmt.Errorf("search failed: %w", err)
 }

 // Format results for LLM
 results := make([]map[string]interface{}, 0, len(output.Results))
 for _, r := range output.Results {
  results = append(results, map[string]interface{}{
   "memo_id":  r.MemoID,
   "memo_url": r.MemoURL,
   "content":  r.Content,
   "score":    r.Score,
  })
 }

 return map[string]interface{}{
  "count":   output.Count,
  "results": results,
 }, nil
}
```

---

### Task 3.8: Qdrant Collection Initialization

**Mục tiêu:** Auto-create Qdrant collection on startup

**File:** `cmd/api/main.go` (update - add Qdrant initialization)

```go
// Initialize Qdrant collection
if cfg.Qdrant.URL != "" {
 logger.Info(ctx, "Initializing Qdrant...")

 qdrantClient := qdrant.NewClient(cfg.Qdrant.URL)

 // Create collection if not exists
 collectionReq := qdrant.CreateCollectionRequest{
  Name: cfg.Qdrant.CollectionName,
  Vectors: qdrant.VectorConfig{
   Size:     768, // Gemini embedding dimension
   Distance: "Cosine",
  },
 }

 if err := qdrantClient.CreateCollection(ctx, collectionReq); err != nil {
  // Collection might already exist, log warning only
  logger.Warnf(ctx, "Qdrant collection creation: %v (may already exist)", err)
 } else {
  logger.Infof(ctx, "Qdrant collection %q created", cfg.Qdrant.CollectionName)
 }

 // Initialize embedding client
 embeddingClient := embedding.NewClient(cfg.Gemini.APIKey)

 // Initialize Qdrant repository
 vectorRepo := qdrantRepo.New(qdrantClient, embeddingClient, cfg.Qdrant.CollectionName, logger)

 // Update task UseCase with vector repository
 taskUC = usecase.New(
  logger,
  geminiClient,
  calendarClient,
  taskRepo,
  vectorRepo, // ✅ Pass vector repository
  dateMathParser,
  timezone,
  cfg.Memos.URL,
 )

 logger.Info(ctx, "Qdrant initialized successfully")
}
```

---

### Task 3.9: Configuration Updates

**File:** `config/config.yaml` (update)

```yaml
app:
  name: "Autonomous Task Management"
  version: "0.3.0" # Phase 3
  env: "development"
  port: 8080

log:
  level: "info"
  format: "json"

memos:
  url: "http://memos:5230"
  access_token: ""
  api_version: "v1"

# Qdrant Configuration (Phase 3)
qdrant:
  url: "http://qdrant:6333"
  collection_name: "task_embeddings"
  vector_size: 768 # Gemini embedding dimension

telegram:
  bot_token: ""
  webhook_url: ""

# Gemini Configuration (Phase 3 - add embedding)
gemini:
  api_key: ""
  timezone: "Asia/Ho_Chi_Minh"
  embedding_model: "text-embedding-004"

google_calendar:
  credentials_path: ""
  calendar_id: "primary"
```

---

## Testing Strategy

### Unit Tests

**1. Qdrant Client Tests**

```go
// pkg/qdrant/client_test.go
func TestClient_CreateCollection(t *testing.T) {
 // Mock HTTP server
 // Test collection creation
}

func TestClient_UpsertPoints(t *testing.T) {
 // Test point insertion
}

func TestClient_SearchPoints(t *testing.T) {
 // Test semantic search
}
```

**2. Embedding Client Tests**

```go
// pkg/embedding/client_test.go
func TestClient_Embed(t *testing.T) {
 // Mock Gemini API
 // Test embedding generation
}

func TestClient_EmbedBatch(t *testing.T) {
 // Test batch embedding
}
```

**3. Search UseCase Tests**

```go
// internal/task/usecase/search_test.go
func TestUseCase_Search(t *testing.T) {
 // Mock vector repository
 // Mock memos repository
 // Test search flow
}
```

### Integration Tests

**1. End-to-End Embedding Flow**

```bash
# Create task → Verify in Qdrant
curl -X POST http://localhost:8080/webhook/telegram \
  -d '{"message": {"text": "Test task for embedding", "chat": {"id": 123}, "from": {"id": 456}}}'

# Check Qdrant
curl http://localhost:6333/collections/task_embeddings/points
```

**2. Semantic Search Flow**

```bash
# Search for task
curl -X POST http://localhost:8080/webhook/telegram \
  -d '{"message": {"text": "tìm task test", "chat": {"id": 123}, "from": {"id": 456}}}'

# Verify results returned
```

### Manual Testing

**Test Cases:**

1. **Create task → Auto-embed:**
   - Send: "Finish SMAP report by tomorrow"
   - Verify: Task created in Memos
   - Verify: Vector stored in Qdrant
   - Check: `curl http://localhost:6333/collections/task_embeddings/points`

2. **Semantic search:**
   - Send: "tìm task SMAP"
   - Verify: Returns relevant tasks
   - Check: Similarity scores > 0.7

3. **Search with no results:**
   - Send: "tìm task xyz không tồn tại"
   - Verify: Returns "Không tìm thấy task nào"

4. **Mixed create and search:**
   - Create 5 tasks with different domains
   - Search by domain: "tìm task ahamove"
   - Verify: Only Ahamove tasks returned

---

## Performance Considerations

### Embedding Generation

**Current:** Sequential embedding (one at a time)

```go
for _, task := range tasks {
    embedding, _ := embedder.Embed(ctx, task.Content)
}
```

**Optimization (if needed):**

```go
// Batch embedding with goroutines
g, ctx := errgroup.WithContext(ctx)
for _, task := range tasks {
    task := task
    g.Go(func() error {
        return vectorRepo.EmbedTask(ctx, task)
    })
}
g.Wait()
```

### Qdrant Search Performance

**Metrics to monitor:**

- Search latency (target: <100ms)
- Embedding generation time (target: <500ms)
- End-to-end search time (target: <1s)

**Optimization strategies:**

- Use Qdrant's HNSW index (default)
- Adjust `ef_construct` and `m` parameters if needed
- Consider caching frequent queries

---

## Deployment Checklist

### Phase 3 Prerequisites

- [ ] Qdrant running and healthy
- [ ] Gemini API key with embedding access
- [ ] Sufficient Qdrant storage (estimate: 1KB per task)
- [ ] Phase 2 fully tested and stable

### Configuration

- [ ] Update `config.yaml` with Qdrant settings
- [ ] Set `GEMINI_API_KEY` in `.env`
- [ ] Verify Qdrant collection name
- [ ] Check vector dimension (768 for Gemini)

### Initialization

- [ ] Qdrant collection auto-created on startup
- [ ] Test collection creation idempotency
- [ ] Verify embedding client connectivity

### Monitoring

- [ ] Add metrics for embedding generation time
- [ ] Add metrics for search latency
- [ ] Add metrics for Qdrant storage usage
- [ ] Add error rate tracking for embedding failures

---

## Migration Strategy

### Backfill Existing Tasks

**Scenario:** Phase 2 đã tạo tasks trong Memos, nhưng chưa có embeddings trong Qdrant.

**Solution:** Backfill script

```go
// scripts/backfill-embeddings/main.go
package main

import (
 "context"
 "log"

 "autonomous-task-management/config"
 "autonomous-task-management/internal/task/repository/memos"
 "autonomous-task-management/internal/task/repository/qdrant"
 "autonomous-task-management/pkg/embedding"
 pkgQdrant "autonomous-task-management/pkg/qdrant"
)

func main() {
 ctx := context.Background()

 // Load config
 cfg, _ := config.Load("config/config.yaml")

 // Initialize clients
 memosClient := memos.NewClient(cfg.Memos.URL, cfg.Memos.AccessToken)
 memosRepo := memos.New(memosClient, cfg.Memos.URL, logger)

 qdrantClient := pkgQdrant.NewClient(cfg.Qdrant.URL)
 embeddingClient := embedding.NewClient(cfg.Gemini.APIKey)
 vectorRepo := qdrant.New(qdrantClient, embeddingClient, cfg.Qdrant.CollectionName, logger)

 // Fetch all tasks from Memos
 tasks, err := memosRepo.ListTasks(ctx, repository.ListTasksOptions{
  Limit: 1000, // Adjust as needed
 })
 if err != nil {
  log.Fatalf("Failed to list tasks: %v", err)
 }

 log.Printf("Found %d tasks to backfill", len(tasks))

 // Embed each task
 for i, task := range tasks {
  if err := vectorRepo.EmbedTask(ctx, task); err != nil {
   log.Printf("Failed to embed task %s: %v", task.ID, err)
   continue
  }
  log.Printf("Embedded task %d/%d: %s", i+1, len(tasks), task.ID)
 }

 log.Println("Backfill complete!")
}
```

**Usage:**

```bash
go run scripts/backfill-embeddings/main.go
```

---

## 💡 Expert Review - Additional Nitpicks

### Nitpick 1: Safe Type Assertion for Qdrant Payload

**Context:** Khi fetch search results từ Qdrant, cần extract `memo_id` từ payload.

**Issue:** Type assertion `.(string)` có thể panic nếu payload bị corrupt hoặc thiếu field.

**Solution:** Defensive programming với detailed error logging

```go
// ❌ RISKY: Simple type assertion
memoID, ok := scored.Payload["memo_id"].(string)
if !ok {
 r.l.Warnf(ctx, "missing memo_id")  // Not enough info
 continue
}

// ✅ SAFE: Two-step check with detailed logging
memoIDRaw, exists := scored.Payload["memo_id"]
if !exists {
 r.l.Errorf(ctx, "memo_id missing in payload for point %v, payload: %+v", 
  scored.ID, scored.Payload)
 continue
}

memoID, ok := memoIDRaw.(string)
if !ok {
 r.l.Errorf(ctx, "memo_id type assertion failed, got type %T, value: %v", 
  memoIDRaw, memoIDRaw)
 continue
}
```

**Benefits:**

- Prevents panic in production
- Detailed logs for debugging
- Graceful degradation (skip bad points)

---

### Nitpick 2: Strip Markdown Code Blocks

**Context:** Users có thể paste code snippets vào task description.

**Issue:** Code blocks pollute embedding semantic content.

**Example:**

```markdown
# Fix authentication bug

```go
func Login(user string) error {
    // 100 lines of code
}
```

The bug is in session validation.

# bug #priority/p1

```

**Problem:** Embedding includes code → dilutes semantic signal.

**Solution:** Strip code blocks before extracting summary

```go
func stripMarkdownCodeBlocks(text string) string {
 // Remove ```language\n...\n``` or ```\n...\n```
 re := regexp.MustCompile("(?s)```[a-z]*\\n.*?\\n```")
 return re.ReplaceAllString(text, "")
}

func buildEmbeddingText(task model.Task) string {
 // ✅ Strip code blocks first
 content := stripMarkdownCodeBlocks(task.Content)
 
 // Then extract title, tags, summary
 // ...
}
```

**Benefits:**

- Cleaner semantic content
- Better search accuracy
- Shorter embedding text (lower cost)

**Test case:**

```go
input := "# Fix bug\n\n```go\ncode here\n```\n\nDescription\n\n#bug"
cleaned := stripMarkdownCodeBlocks(input)
// Expected: "# Fix bug\n\nDescription\n\n#bug"
assert.NotContains(cleaned, "code here")
```

---

### Why These Matter

**Idempotency (UUID v5):**

- Memos webhook có thể trigger multiple times
- Same memoID → Same UUID → Upsert (not duplicate)
- Zero vector garbage in Qdrant

**Explicit Command (/search):**

- Avoids over-engineering (no LLM for intent classification)
- Zero false positives
- Saves API cost and latency
- User experience: clear and predictable

**Semantic Filtering (buildEmbeddingText):**

- Title + Tags + Summary = high signal-to-noise ratio
- Code blocks stripped = cleaner embeddings
- Better search accuracy with less data

**Safe Type Assertions:**

- Production-grade error handling
- Detailed logs for debugging
- Graceful degradation (skip bad data, don't crash)

---

## Deliverables Phase 3

Sau khi hoàn thành Phase 3, hệ thống sẽ có:

1. ✅ **Qdrant Integration:** Vector database hoạt động với auto-collection creation
2. ✅ **Embedding Service:** Generate embeddings từ Gemini API
3. ✅ **Auto-Embed:** Tasks tự động được embed sau khi tạo
4. ✅ **Semantic Search:** Tìm kiếm tasks bằng natural language
5. ✅ **Smart Telegram Handler:** Phân biệt create vs search intent
6. ✅ **Agent Tools Framework:** Foundation cho function calling (Phase 3 advanced)
7. ✅ **Backfill Script:** Migrate existing tasks to Qdrant

**Chưa có trong Phase 3 Basic:**

- ❌ LLM Function Calling với tools (Phase 3 Advanced)
- ❌ RAG với context enhancement (Phase 3 Advanced)
- ❌ Calendar conflict detection tool (Phase 3 Advanced)
- ❌ Memos webhook sync (Phase 3 Advanced)

---

## Phase 3 Advanced (Optional Extensions)

### Extension 1: LLM Function Calling

**Mục tiêu:** LLM tự động quyết định gọi tools

**Implementation:**

```go
// internal/agent/orchestrator/orchestrator.go
func (o *Orchestrator) ProcessQuery(ctx context.Context, query string) (string, error) {
 // Step 1: Send query to LLM with tool definitions
 req := gemini.GenerateRequest{
  Contents: []gemini.Content{
   {Parts: []gemini.Part{{Text: query}}},
  },
  Tools: o.registry.ToFunctionDefinitions(),
 }

 resp, err := o.llm.GenerateContent(ctx, req)
 if err != nil {
  return "", err
 }

 // Step 2: Check if LLM wants to call a tool
 if resp.FunctionCall != nil {
  // Execute tool
  tool, ok := o.registry.Get(resp.FunctionCall.Name)
  if !ok {
   return "", fmt.Errorf("unknown tool: %s", resp.FunctionCall.Name)
  }

  result, err := tool.Execute(ctx, resp.FunctionCall.Args)
  if err != nil {
   return "", fmt.Errorf("tool execution failed: %w", err)
  }

  // Step 3: Send tool result back to LLM
  req.Contents = append(req.Contents, gemini.Content{
   Parts: []gemini.Part{{FunctionResponse: result}},
  })

  resp, err = o.llm.GenerateContent(ctx, req)
  if err != nil {
   return "", err
  }
 }

 // Step 4: Return final answer
 return resp.Text, nil
}
```

### Extension 2: RAG (Retrieval Augmented Generation)

**Mục tiêu:** Enhance LLM context với retrieved tasks

**Implementation:**

```go
// internal/task/usecase/answer_query.go
func (uc *implUseCase) AnswerQuery(ctx context.Context, sc model.Scope, input task.QueryInput) (task.QueryOutput, error) {
 // Step 1: Search for relevant tasks
 searchResults, err := uc.vectorRepo.SearchTasks(ctx, repository.SearchTasksOptions{
  Query: input.Query,
  Limit: 5,
 })
 if err != nil {
  return task.QueryOutput{}, err
 }

 // Step 2: Fetch full task content
 var contextTasks []string
 for _, sr := range searchResults {
  memoTask, _ := uc.memosRepo.GetTask(ctx, sr.MemoID)
  contextTasks = append(contextTasks, memoTask.Content)
 }

 // Step 3: Build enhanced prompt
 prompt := fmt.Sprintf(`Based on the following tasks:

%s

Answer this question: %s`, strings.Join(contextTasks, "\n\n---\n\n"), input.Query)

 // Step 4: Send to LLM
 req := gemini.GenerateRequest{
  Contents: []gemini.Content{
   {Parts: []gemini.Part{{Text: prompt}}},
  },
 }

 resp, err := uc.llm.GenerateContent(ctx, req)
 if err != nil {
  return task.QueryOutput{}, err
 }

 return task.QueryOutput{
  Answer:        resp.Candidates[0].Content.Parts[0].Text,
  SourceTasks:   searchResults,
  SourceCount:   len(searchResults),
 }, nil
}
```

### Extension 3: Calendar Conflict Detection Tool

**Implementation:**

```go
// internal/agent/tools/check_calendar.go
type CheckCalendarTool struct {
 calendar *gcalendar.Client
}

func (t *CheckCalendarTool) Name() string {
 return "check_calendar"
}

func (t *CheckCalendarTool) Description() string {
 return "Check for scheduling conflicts in Google Calendar for a given time range"
}

func (t *CheckCalendarTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
 startTime, _ := time.Parse(time.RFC3339, params["start_time"].(string))
 endTime, _ := time.Parse(time.RFC3339, params["end_time"].(string))

 // Query calendar for events in time range
 events, err := t.calendar.ListEvents(ctx, gcalendar.ListEventsRequest{
  TimeMin: startTime,
  TimeMax: endTime,
 })
 if err != nil {
  return nil, err
 }

 return map[string]interface{}{
  "has_conflict": len(events) > 0,
  "events":       events,
  "count":        len(events),
 }, nil
}
```

### Extension 4: Memos Webhook Sync

**Mục tiêu:** Auto-update Qdrant khi task được sửa/xóa trong Memos

**Implementation:**

```go
// internal/sync/handler.go
type Handler struct {
 vectorRepo repository.VectorRepository
 memosRepo  repository.MemosRepository
 l          pkgLog.Logger
}

func (h *Handler) HandleMemosWebhook(c *gin.Context) {
 ctx := c.Request.Context()

 var event MemosWebhookEvent
 if err := c.ShouldBindJSON(&event); err != nil {
  h.l.Errorf(ctx, "Failed to parse webhook: %v", err)
  pkgResponse.Error(c, err, nil)
  return
 }

 switch event.Action {
 case "created":
  // Embed new task
  task, _ := h.memosRepo.GetTask(ctx, event.MemoID)
  h.vectorRepo.EmbedTask(ctx, task)

 case "updated":
  // Re-embed updated task
  task, _ := h.memosRepo.GetTask(ctx, event.MemoID)
  h.vectorRepo.EmbedTask(ctx, task)

 case "deleted":
  // Remove from Qdrant
  h.vectorRepo.DeleteTask(ctx, event.MemoID)
 }

 pkgResponse.OK(c, map[string]string{"status": "ok"})
}
```

---

## Troubleshooting

### Qdrant ID constraint error

**Symptoms:** Error "HTTP 400 Bad Request" khi upsert points

**Root cause:** Trying to use arbitrary string as Point ID

**Debug:**

```bash
# Check error message from Qdrant
docker compose logs qdrant | grep "400"

# Typical error: "Invalid point ID format"
```

**Fix:**

```go
// ❌ WRONG: Using Memos UID directly
point := pkgQdrant.Point{
 ID: task.ID,  // "abc123xyz" → 400 error
}

// ✅ CORRECT: Convert to UUID
qdrantID := memoIDToUUID(task.ID)
point := pkgQdrant.Point{
 ID: qdrantID,  // "550e8400-e29b-41d4-a716-446655440000" → OK
 Payload: map[string]interface{}{
  "memo_id": task.ID,  // Store original ID
 },
}
```

**Verify fix:**

```bash
# Check points in Qdrant
curl http://localhost:6333/collections/task_embeddings/points | jq '.result.points[0].id'
# Should return UUID format
```

---

### Intent detection false positive

**Symptoms:**

- Message "Tìm hiểu X" triggers search instead of creating task
- Task creation lost

**Debug:**

```go
// Add logging in handler
h.l.Infof(ctx, "Message: %q, IsSearch: %v", msg.Text, isSearchIntent(msg.Text))
```

**Fix:**

```go
// ❌ WRONG: Regex detection
if strings.HasPrefix(strings.ToLower(msg.Text), "tìm") {
 // "Tìm hiểu..." triggers this!
}

// ✅ CORRECT: Explicit command
if strings.HasPrefix(msg.Text, "/search ") {
 // Only "/search ..." triggers this
}
```

**User education:**

```
Update /help command:
- Create task: Just send text (e.g., "Tìm hiểu VNPay integration")
- Search: Use /search command (e.g., "/search VNPay tasks")
```

---

### Search returns wrong tasks

**Symptoms:** Search results not relevant to query

**Root cause:** Embedding full content (including logs, stack traces)

**Debug:**

```go
// Log what's being embedded
h.l.Infof(ctx, "Embedding text: %q", buildEmbeddingText(task))
```

**Fix:**

```go
// ❌ WRONG: Embed full content
func buildEmbeddingText(task model.Task) string {
 return task.Content  // Includes noise
}

// ✅ CORRECT: Extract semantic parts only
func buildEmbeddingText(task model.Task) string {
 title := extractTitle(task.Content)
 tags := extractTags(task.Content)
 summary := extractSummary(task.Content, 2)
 return title + "\n" + tags + "\n" + summary
}
```

**Test:**

```bash
# Create test task with noise
curl -X POST /webhook/telegram -d '{
  "message": {
    "text": "Fix bug\n\n[1000 lines of logs]\n\n#bug",
    "chat": {"id": 123},
    "from": {"id": 456}
  }
}'

# Search should still find it
curl -X POST /webhook/telegram -d '{
  "message": {
    "text": "/search fix bug",
    "chat": {"id": 123},
    "from": {"id": 456}
  }
}'
```

---

### Qdrant connection failed

**Symptoms:** Error "failed to call qdrant API"

**Fix:**

```bash
# Check Qdrant is running
docker compose ps qdrant

# Check Qdrant health
curl http://localhost:6333/health

# Restart Qdrant
docker compose restart qdrant
```

### Embedding generation failed

**Symptoms:** Error "failed to generate embedding"

**Root causes:**

1. Invalid Gemini API key
2. Rate limit exceeded
3. Text too long (>10k tokens)

**Fix:**

```bash
# Verify API key
echo $GEMINI_API_KEY

# Check Gemini API quota
# https://makersuite.google.com/app/apikey

# Add retry logic with exponential backoff
```

### Search returns no results

**Symptoms:** Search always returns empty

**Debug:**

```bash
# Check if collection exists
curl http://localhost:6333/collections/task_embeddings

# Check if points exist
curl http://localhost:6333/collections/task_embeddings/points

# Check vector dimension matches
# Should be 768 for Gemini
```

### Embedding dimension mismatch

**Symptoms:** Error "vector dimension mismatch"

**Fix:**

```bash
# Delete collection and recreate
curl -X DELETE http://localhost:6333/collections/task_embeddings

# Restart backend (will recreate collection)
docker compose restart backend
```

---

## Performance Benchmarks

### Target Metrics

| Operation              | Target | Acceptable |
| ---------------------- | ------ | ---------- |
| Embedding generation   | <500ms | <1s        |
| Qdrant search          | <100ms | <300ms     |
| End-to-end search      | <1s    | <2s        |
| Batch embed (10 tasks) | <5s    | <10s       |

### Optimization Tips

**1. Batch Operations:**

```go
// Instead of sequential
for _, task := range tasks {
    embedder.Embed(ctx, task.Content)
}

// Use batch
embedder.EmbedBatch(ctx, texts)
```

**2. Caching:**

```go
// Cache frequent queries
type SearchCache struct {
    cache map[string][]SearchResult
    ttl   time.Duration
}
```

**3. Qdrant Tuning:**

```yaml
# Adjust HNSW parameters in collection config
vectors:
  size: 768
  distance: Cosine
  hnsw_config:
    m: 16 # Connections per node
    ef_construct: 100 # Build-time accuracy
```

---

## Security Considerations

### API Key Management

**Gemini API Key:**

- Store in environment variable
- Rotate periodically
- Monitor usage quota
- Set up billing alerts

### Qdrant Access Control

**Current:** No authentication (local only)

**Production recommendations:**

- Enable Qdrant API key authentication
- Use TLS for Qdrant connections
- Restrict network access to Qdrant port

### Data Privacy

**Embeddings contain semantic information:**

- Consider encrypting Qdrant storage
- Implement data retention policies
- Add GDPR compliance (right to be forgotten)

---

## Thời gian Ước tính

- Qdrant client implementation: 3-4 giờ
- Embedding service: 2-3 giờ
- Qdrant repository: 3-4 giờ
- Update create_bulk with embedding: 2 giờ
- Implement search usecase: 3-4 giờ
- Update Telegram handler: 3-4 giờ
- Agent tools framework: 4-5 giờ
- Testing & debugging: 4-5 giờ
- Documentation: 2-3 giờ

**Tổng Phase 3 Basic: 26-34 giờ** (3-5 ngày làm việc)

**Phase 3 Advanced (Optional):**

- LLM Function Calling: 4-6 giờ
- RAG implementation: 3-4 giờ
- Calendar conflict tool: 2-3 giờ
- Memos webhook sync: 3-4 giờ

**Tổng Phase 3 Advanced: 12-17 giờ** (2-3 ngày làm việc)

---

## Next Steps (Phase 4 Preview)

Phase 4 sẽ tập trung vào automation và webhooks:

**1. Webhook Automation:**

- Receive webhooks từ Git (GitHub/GitLab)
- Auto-update checklist trong Memos
- Regex parser cho markdown checklist

**2. Advanced Agent Orchestration:**

- Multi-step reasoning
- Tool chaining
- Error recovery strategies

**3. Notification System:**

- Proactive reminders
- Deadline alerts
- Blocked task detection

**4. Analytics & Insights:**

- Task completion metrics
- Time tracking
- Productivity insights

---

## References

- [Qdrant Documentation](https://qdrant.tech/documentation/)
- [Qdrant HTTP API](https://qdrant.tech/documentation/interfaces/)
- [Gemini Embedding API](https://ai.google.dev/docs/embeddings_guide)
- [Vector Search Best Practices](https://qdrant.tech/documentation/tutorials/search-beginners/)
- [HNSW Algorithm](https://arxiv.org/abs/1603.09320)
- [RAG Pattern](https://www.pinecone.io/learn/retrieval-augmented-generation/)

---

## 🎓 Phase 3 Summary

**Achievements:**

- ✅ Semantic memory với Qdrant
- ✅ Auto-embedding pipeline
- ✅ Natural language search
- ✅ Smart intent detection
- ✅ Agent tools foundation

**Key Innovations:**

- Tách bạch embedding logic khỏi business logic
- Graceful degradation (embedding failures không block task creation)
- Extensible tool framework cho future enhancements

**Production Readiness:**

- Core functionality: 100% ✅
- Testing coverage: Needs improvement ⚠️
- Performance optimization: Good enough for MVP ✅
- Security: Basic (needs hardening for production) ⚠️

**Recommendation:**

- Implement Phase 3 Basic first
- Test thoroughly with real data
- Monitor performance metrics
- Consider Phase 3 Advanced based on user feedback

Bạn đã sẵn sàng để implement Phase 3! 🚀
