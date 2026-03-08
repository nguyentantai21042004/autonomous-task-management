# HIGH-LEVEL ARCHITECTURE — Autonomous Task Management

> Tài liệu mô tả kiến trúc tổng quan và toàn bộ khả năng hiện tại của hệ thống.
> Cập nhật: Tháng 3/2026

---

## 1. Tổng quan Dự án

**Autonomous Task Management** là một hệ thống quản lý công việc tự trị, sử dụng AI Agent để xử lý mọi tác vụ qua một giao diện duy nhất — **Telegram Bot**.

### 1.1. Vấn đề Giải quyết

| Vấn đề | Giải pháp |
|--------|-----------|
| Phân mảnh công cụ (Notion, Slack, Calendar, GitHub) | Một giao diện Telegram duy nhất |
| Tìm kiếm task bằng keyword không hiệu quả | Semantic search bằng vector database |
| Quên deadline, thiếu nhắc nhở | Tự động đồng bộ Google Calendar |
| Phải update task thủ công sau khi merge PR | Webhook automation tự cập nhật |
| Cú pháp phức tạp, khó nhớ command | Chat tự nhiên bằng tiếng Việt/Anh |

### 1.2. Tech Stack

| Thành phần | Công nghệ |
|-----------|-----------|
| Language | Go 1.25 |
| HTTP Framework | Gin |
| Task Storage | Memos (self-hosted, Markdown) |
| Vector Database | Qdrant |
| Embeddings | Voyage AI (`voyage-3`, 1024 dims) |
| LLM | Multi-provider: Qwen, DeepSeek, Gemini |
| Calendar | Google Calendar API (OAuth2) |
| Chat Interface | Telegram Bot (Webhook) |
| Logging | Zap |
| Config | Viper |

---

## 2. Kiến trúc Tổng thể

```
┌──────────────────────── USER INTERFACE ────────────────────────┐
│  Telegram Bot        GitHub Webhooks        GitLab Webhooks    │
└────────┬─────────────────┬──────────────────────┬──────────────┘
         │                 │                      │
         ▼                 ▼                      ▼
┌──────────────────── GO BACKEND (Gin) ─────────────────────────┐
│                                                                │
│  ┌─────────────┐  ┌──────────────┐  ┌───────────────────────┐ │
│  │  Semantic    │  │    Agent     │  │    Webhook Parser     │ │
│  │  Router      │──▶ Orchestrator │  │  (GitHub / GitLab)    │ │
│  │ (Intent)     │  │ (Graph V2)   │  └───────────┬───────────┘ │
│  └─────────────┘  └──────┬───────┘              │             │
│                          │                      │             │
│          ┌───────────────┼──────────────────────┤             │
│          ▼               ▼                      ▼             │
│  ┌─────────────┐ ┌─────────────┐ ┌──────────────────────┐    │
│  │  Task UC    │ │ Checklist UC│ │   Automation UC      │    │
│  │ Create/     │ │ Parse/      │ │   Webhook→Task Match │    │
│  │ Search/RAG  │ │ Update/Stats│ │   Auto-complete      │    │
│  └──────┬──────┘ └─────────────┘ └──────────────────────┘    │
│         │                                                     │
│  ┌──────▼──────┐  ┌──────────┐  ┌──────────┐                │
│  │ Sync UC     │  │ Indexer  │  │ DateMath │                 │
│  │ Memos↔Qdrant│  │ Enricher │  │ Parser   │                 │
│  └─────────────┘  └──────────┘  └──────────┘                │
└───────────────────────────┬───────────────────────────────────┘
                            │
         ┌──────────────────┼──────────────────────┐
         ▼                  ▼                      ▼
┌─────────────┐    ┌──────────────┐    ┌──────────────────────┐
│   Memos     │    │   Qdrant     │    │   Google Calendar    │
│  (Storage)  │    │ (Vector DB)  │    │   (Scheduling)       │
└─────────────┘    └──────────────┘    └──────────────────────┘

┌──────────────────── LLM & EMBEDDING ──────────────────────────┐
│  Multi-provider fallback (configurable priority order)        │
│  Global timeout: 20s, retry: 2 attempts per provider          │
│  Voyage AI (embeddings: voyage-3, 1024 dims)                  │
└───────────────────────────────────────────────────────────────┘
```

---

## 3. Kiến trúc Phần mềm — Clean Architecture

Dự án tuân thủ **Clean Architecture** với 4 lớp tách biệt:

```
Delivery (HTTP Handlers)
    ↓
UseCase (Business Logic)
    ↓
Repository (Data Access)
    ↓
Model (Domain Entities)
```

### 3.1. Quy tắc phụ thuộc

- **Delivery** chỉ gọi UseCase (thông qua interface)
- **UseCase** chỉ gọi Repository (thông qua interface)
- **Repository** tương tác trực tiếp với external services (Memos API, Qdrant API)
- **Model** là layer trong cùng, không phụ thuộc bất kỳ layer nào khác

### 3.2. Cấu trúc Domain

Mỗi domain (task, agent, automation, checklist...) đều có cùng structure:

```
internal/<domain>/
├── interface.go          # Interface contracts (UseCase, Repository, Handler)
├── types.go              # Input/Output DTOs
├── errors.go             # Domain-specific errors
├── usecase/              # Business logic implementation
├── repository/           # Data access implementation
├── delivery/             # HTTP/Telegram handlers
└── tools/                # Agent tool implementations
```

---

## 4. Các Domain Chính

### 4.1. Agent — AI Orchestrator

**Nhiệm vụ**: Điều phối toàn bộ luồng xử lý message từ user.

**Kiến trúc V2.0 — Stateful Graph Engine**:

```
User Message
    │
    ▼
┌─────────────┐    ┌──────────────────┐
│  NodeAgent   │───▶│ NodeExecuteTool  │
│ (LLM Think)  │◀───│ (Run Tool)       │
└──────┬───────┘    └──────────────────┘
       │
       ├─── RUNNING: tiếp tục loop
       ├─── WAITING_FOR_HUMAN: tạm dừng, chờ xác nhận
       ├─── FINISHED: trả kết quả
       └─── ERROR: xử lý lỗi
```

**Đặc điểm**:

- **ReAct Pattern**: Reason → Act → Observe, lặp lại đến khi hoàn thành
- **Session Memory**: Lưu 5 lượt hội thoại gần nhất (LRU cache, TTL 30 phút)
- **Tool Registry**: Các domain tự đăng ký tool của mình (decoupled)
- **Max 10 bước**: Giới hạn số bước xử lý để tránh loop vô hạn
- **Human-in-the-loop**: Tạm dừng tại thao tác nguy hiểm (xóa, cập nhật hàng loạt)
- **Temperature = 0.7**: Agent LLM request set rõ ràng cho conversational tone
- **Time Context in System Prompt**: Thông tin thời gian inject vào system prompt (không lặp trong history)
- **Context Compression**: Tự động nén khi history > 8 messages (truncate 120 ký tự + summary)

**Tools được đăng ký**:

| Tool | Domain | Mô tả |
|------|--------|--------|
| `search_tasks` | Task | Tìm kiếm task bằng semantic search |
| `create_task` | Task | Tạo task mới từ mô tả tự nhiên |
| `answer_query` | Task | Trả lời câu hỏi dựa trên context của tasks (RAG) |
| `check_checkbox` | Checklist | Đánh dấu checkbox trong task |
| `get_checklist_stats` | Checklist | Lấy thống kê tiến độ checklist |

---

### 4.2. Semantic Router — Phân loại Intent

**Nhiệm vụ**: Dùng LLM để phân loại ý định của user trước khi chuyển sang Agent.

| Intent | Mô tả | Ví dụ |
|--------|--------|-------|
| `CREATE_TASK` | Tạo task mới | "Tạo task review PR backend" |
| `SEARCH_TASK` | Tìm kiếm task | "Task nào liên quan tới API?" |
| `MANAGE_CHECKLIST` | Quản lý checklist | "Tick checkbox deploy" |
| `CONVERSATION` | Hội thoại chung | "Hôm nay có gì cần làm?" |

Output: `{ Intent, Confidence (0-100), Reasoning }`

---

### 4.3. Task — Quản lý Công việc

**Capabilities**:

1. **CreateBulk** — Tạo task hàng loạt từ tin nhắn tự nhiên
   - LLM parse tin nhắn → JSON array các task
   - DateMath parse ngày tương đối ("thứ 6 tuần sau") → timestamp
   - Tạo memo trong Memos (Markdown format)
   - Tạo embedding → upsert vào Qdrant
   - Nếu có deadline → tạo event Google Calendar

2. **Search** — Tìm kiếm ngữ nghĩa
   - Hybrid search: Dense vector + Full-text parallel → RRF fusion
   - Parallel fetch task details từ Memos (sync.WaitGroup)
   - Hỗ trợ lọc theo tags
   - Self-healing: tự cleanup zombie vectors khi task bị xóa
   - Trả về kết quả ranked theo relevance score

3. **AnswerQuery** — Hỏi đáp RAG
   - Hybrid search: Dense + Full-text parallel → RRF fusion (repo layer)
   - Optional Voyage cross-encoder reranking (usecase layer)
   - Parallel fetch task details từ Memos (sync.WaitGroup)
   - Self-healing: tự cleanup zombie vectors (task đã xóa trong Memos)
   - Gửi context + câu hỏi vào LLM
   - Trả lời kèm source tasks (trích dẫn nguồn)

---

### 4.4. Automation — Tự động hóa Webhook

**Nhiệm vụ**: Xử lý events từ GitHub/GitLab để tự động cập nhật task.

**Luồng xử lý**:

```
GitHub/GitLab Webhook Event
    │
    ▼
Webhook UC: Parse & verify (HMAC/token)
    │
    ▼
Automation UC: Match event → tasks (by tags: #pr/123, #repo/backend)
    │
    ▼
Auto-complete matched tasks → Update Memos → Re-sync Qdrant
```

**Hỗ trợ events**: push, pull_request (opened/merged/closed), issue (opened/closed)

---

### 4.5. Checklist — Quản lý Markdown Checklist

**Nhiệm vụ**: Parse và cập nhật checkbox trong nội dung Markdown của task.

**Capabilities**:

- Parse `- [ ] item` / `- [x] item` thành structured data
- Cập nhật checkbox theo text match (partial OK)
- Tính tiến độ: `{ Total, Completed, Pending, Progress% }`
- Cập nhật hàng loạt (check/uncheck all)
- Phát hiện task hoàn thành 100%

---

### 4.6. Sync — Đồng bộ Memos ↔ Qdrant

**Nhiệm vụ**: Khi task thay đổi trong Memos, tự động cập nhật vector trong Qdrant.

**Trigger**: Memos webhook → `/webhook/memos`

**Actions**:

- Task created/updated → Re-embed content → Upsert vector vào Qdrant
- Task deleted → Xóa vector tương ứng khỏi Qdrant

---

## 5. Tích hợp Bên ngoài (External Integrations)

### 5.1. LLM Provider Manager

Hệ thống hỗ trợ **multi-provider LLM** với cơ chế **failover tự động**:

```
Request → Provider priority 1 (configurable)
             │
             ├── Success → Return
             │
             ├── Fail → Retry (2 lần, exponential backoff)
             │              │
             │              └── All retries fail
             ▼
         Provider priority 2
             │
             ├── Success → Return
             │
             ├── Fail → Retry
             ▼
         Provider priority 3
             │
             └── ... (same pattern)

Global timeout: 20s cho toàn bộ fallback chain
```

**Tính năng**:

- Mỗi provider implement cùng `Provider` interface
- Hỗ trợ **function calling** (tool use) ở tất cả providers
- Token usage tracking
- Retry với exponential backoff (max 2 attempts per provider)
- HTTP timeout: Qdrant 10s, Voyage 15s

### 5.2. Voyage AI — Embeddings

- Model: `voyage-3` (1024 dimensions, multilingual)
- Dùng cho: Embedding task content trước khi lưu vào Qdrant
- **Indexer Enricher**: Trước khi embed, nội dung task được làm giàu thêm:
  - Gắn temporal context ("deadline tuần này", "quá hạn 3 ngày")
  - Gắn tags
  - → Cải thiện chất lượng semantic search

### 5.3. Qdrant — Vector Database

- Collection: `tasks`
- Vector size: 1024 (khớp với Voyage AI)
- Metric: Cosine similarity
- Payload: `memo_id`, `tags`, metadata
- Operations: Upsert, Search, Scroll, Delete

### 5.4. Memos — Task Storage

- Self-hosted note-taking app
- REST API với access token
- Markdown-native (task = memo)
- Tag-based organization
- Webhook support cho sync

### 5.5. Google Calendar

- OAuth2 authentication
- Tạo event khi task có deadline
- Liệt kê events để kiểm tra conflict
- Deep link cho mobile access
- Timezone-aware (Asia/Ho_Chi_Minh)

### 5.6. Telegram Bot

- Webhook mode (không polling)
- Hỗ trợ: text input, voice message
- Parse modes: HTML, Markdown, MarkdownV2
- Gửi response với rich formatting

---

## 6. Luồng Xử lý Chính

### 6.1. User gửi tin nhắn tạo task

```
User: "Tạo task review PR #42 backend, deadline thứ 6"
  │
  ▼
Telegram Webhook → Handler
  │
  ▼
Router.Classify() → Intent: CREATE_TASK (confidence: 95)
  │
  ▼
Agent.ProcessQuery()
  │
  ▼
Graph Engine: NodeAgent (LLM reasoning)
  │ → Quyết định gọi tool "create_task"
  ▼
Graph Engine: NodeExecuteTool
  │
  ├── LLM parse: {title: "Review PR #42 backend", tags: [#pr/42, #backend], due: "thứ 6"}
  ├── DateMath: "thứ 6" → 2026-03-13T23:59:59+07:00
  ├── Memos API: CreateMemo(content with markdown)
  ├── Voyage AI: Embed(enriched content) → [0.12, -0.45, ...]
  ├── Qdrant: Upsert(vector, payload)
  └── Google Calendar: CreateEvent(summary, startTime)
  │
  ▼
Telegram: Gửi response "✅ Đã tạo task: Review PR #42 backend"
```

### 6.2. User tìm kiếm task

```
User: "Tìm các task liên quan tới API authentication"
  │
  ▼
Router → SEARCH_TASK → Agent
  │
  ▼
Agent gọi tool "search_tasks"
  │
  ├── Voyage AI: Embed("API authentication") → query vector
  ├── Qdrant: Search(query vector, limit=5) → ranked results
  └── Format kết quả với score + link
  │
  ▼
Telegram: Response với danh sách task + relevance score
```

### 6.3. Webhook automation — PR merge

```
GitHub: PR #42 merged
  │
  ▼
POST /webhook/github (HMAC verified)
  │
  ▼
Webhook UC: Parse event → {source: github, action: merged, PRNumber: 42}
  │
  ▼
Automation UC: Tìm task có tag #pr/42
  │
  ├── Tìm thấy → Auto-complete task
  ├── Update Memos: Đánh dấu hoàn thành
  └── Re-sync Qdrant
  │
  ▼
Telegram: Thông báo "✅ Task #42 đã tự động hoàn thành"
```

---

## 7. Cấu trúc Thư mục

```
autonomous-task-management/
├── cmd/api/                    # Entry point, server initialization
│   ├── main.go
│   ├── Dockerfile / Dockerfile.dev
│
├── config/                     # Configuration (Viper)
│   ├── config.go
│   ├── config.yaml / config.example.yaml
│
├── internal/                   # Business logic (Clean Architecture)
│   ├── agent/                  # AI Agent orchestrator
│   │   ├── graph/              # V2.0 Stateful Graph Engine
│   │   └── usecase/
│   ├── automation/             # Webhook → Task automation
│   ├── checklist/              # Markdown checklist parsing
│   ├── httpserver/             # Gin server, routing, DI
│   ├── model/                  # Domain entities (Task, Event, Scope)
│   ├── router/                 # Semantic intent classification
│   ├── sync/                   # Memos ↔ Qdrant synchronization
│   ├── task/                   # Core task CRUD + search + RAG
│   │   ├── usecase/
│   │   ├── repository/
│   │   ├── delivery/
│   │   └── tools/
│   ├── test/                   # E2E testing endpoints
│   └── webhook/                # GitHub/GitLab event parsing
│
├── pkg/                        # Reusable packages (external clients)
│   ├── llmprovider/            # Multi-provider LLM manager + fallback
│   ├── gemini/                 # Google Gemini client
│   ├── deepseek/               # DeepSeek client
│   ├── qwen/                   # Alibaba Qwen client
│   ├── telegram/               # Telegram Bot API client
│   ├── gcalendar/              # Google Calendar API client
│   ├── qdrant/                 # Qdrant vector DB client
│   ├── voyage/                 # Voyage AI embeddings client
│   ├── indexer/                # Task content enrichment for embeddings
│   ├── datemath/               # Temporal expression parser
│   ├── log/                    # Zap logger wrapper
│   └── response/               # HTTP response helpers
│
├── documents/                  # Documentation & plans
├── manifests/                  # Docker Compose, schemas
├── scripts/                    # Setup & utility scripts
└── vendor/                     # Go module vendoring
```

---

## 8. Design Patterns & Quyết định Kiến trúc

| Pattern | Áp dụng |
|---------|---------|
| **Clean Architecture** | 4 layers: Delivery → UseCase → Repository → Model |
| **Dependency Injection** | Constructor injection, không global state |
| **Interface-Driven** | Mọi layer giao tiếp qua interface |
| **Tool Registry** | Domain tự đăng ký tool vào Agent (plugin-like) |
| **ReAct + Graph** | Agent thinking loop với state machine |
| **Multi-Provider Fallback** | LLM failover tự động với retry |
| **Event-Driven Sync** | Memos webhook trigger Qdrant sync |
| **Content Enrichment** | Làm giàu nội dung trước khi embed (temporal, tags) |

---

## 9. Tổng kết — Những gì hệ thống đã làm được

### Tính năng hoàn chỉnh

1. **Chat tự nhiên qua Telegram** — Nhận tin nhắn tiếng Việt/Anh, AI hiểu và xử lý
2. **Tạo task từ ngôn ngữ tự nhiên** — Tự parse title, deadline, tags, ước tính thời gian
3. **Tìm kiếm ngữ nghĩa** — Vector search, không cần khớp từ khóa chính xác
4. **Hỏi đáp RAG** — Truy vấn context từ tất cả tasks, LLM tổng hợp câu trả lời
5. **Quản lý checklist** — Parse/update checkbox trong Markdown, theo dõi tiến độ %
6. **Tự động hóa GitHub/GitLab** — PR merge → task auto-complete
7. **Đồng bộ Memos ↔ Qdrant** — Task thay đổi → vector tự cập nhật
8. **Google Calendar** — Tạo event khi task có deadline, kiểm tra conflict
9. **Multi-provider LLM** — Failover tự động với retry 2 lần, global timeout 20s
10. **Session Memory** — Ghi nhớ 5 lượt hội thoại, hỗ trợ multi-turn

### Điểm mạnh kiến trúc

- **Self-hosted hoàn toàn** — Không phụ thuộc cloud nào (trừ LLM API)
- **Modular & Extensible** — Thêm domain mới chỉ cần implement interface
- **Markdown-native** — Dữ liệu plain text, dễ backup/migrate
- **Fault-tolerant LLM** — LLM chết 1 provider vẫn hoạt động bình thường
- **Human-in-the-loop** — Agent tạm dừng khi cần xác nhận thao tác nguy hiểm
- **Temporal-aware search** — Tìm "deadline tuần này" hoạt động chính xác

### Phiên bản phát triển

| Version | Nội dung chính |
|---------|----------------|
| **V1.0** | Core: Task CRUD, Telegram bot, Memos + Qdrant, LLM single provider |
| **V1.1** | Semantic router, multi-turn conversation, checklist management |
| **V1.2** | Multi-provider LLM fallback, content enrichment, date math parser |
| **V2.0** | Stateful Graph Engine, human-in-the-loop, GitHub/GitLab webhooks |
