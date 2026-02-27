# ARCHITECTURE OVERVIEW: AUTONOMOUS TASK MANAGEMENT

*Tài liệu này mô tả kiến trúc hệ thống từ high-level đến low-level, giúp người đọc có cái nhìn tổng quan về cách hệ thống được xây dựng.*

---

## 1. Tổng quan Hệ thống (System Overview)

### 1.1. Vấn đề Giải quyết

Hệ thống **Autonomous Task Management** giải quyết bài toán quản lý công việc phức tạp trong thời đại đa nền tảng:

- **Phân mảnh công cụ**: Người dùng phải chuyển đổi giữa nhiều ứng dụng (Notion, Google Calendar, Slack, GitHub) chỉ để quản lý một task
- **Mất ngữ cảnh**: Task bị chôn vùi trong đống notes, khó tìm kiếm lại context của dự án cũ
- **Thiếu tự động hóa**: Phải manually update trạng thái task sau khi merge PR, deploy code
- **Rào cản giao tiếp**: Cần học cú pháp phức tạp thay vì chat tự nhiên

### 1.2. Giải pháp Cốt lõi

Hệ thống cung cấp một **giao diện duy nhất** (Telegram) kết hợp với **AI Agent tự trị** để:

1. **Tạo task tự nhiên**: Chat như với người, AI tự parse và tạo task
2. **Tìm kiếm ngữ nghĩa**: Tìm theo ý nghĩa, không cần khớp từ khóa chính xác
3. **Tự động hóa workflow**: Webhook từ Git tự động cập nhật trạng thái task
4. **Quản lý lịch thông minh**: Agent tự check conflict và đặt lịch

### 1.3. Đặc điểm Nổi bật

- **Agentic AI**: ReAct framework với tool calling (Multi-provider LLM: Qwen primary, Gemini fallback)
- **Semantic Search**: Vector database (Qdrant) với embeddings (Voyage AI)
- **Markdown-native**: Lưu trữ dạng plain text, dễ backup và migrate
- **Self-hosted**: Chạy hoàn toàn local, zero cloud dependency
- **Event-driven**: Webhook-based automation với GitHub/GitLab

---

## 2. Kiến trúc Tổng thể (High-Level Architecture)


### 2.1. Sơ đồ Kiến trúc

```
┌─────────────────────────────────────────────────────────────────────┐
│                         USER INTERFACE LAYER                         │
│                                                                       │
│  ┌──────────────┐         ┌──────────────┐        ┌──────────────┐ │
│  │   Telegram   │         │    GitHub    │        │   GitLab     │ │
│  │     Bot      │         │   Webhooks   │        │  Webhooks    │ │
│  └──────┬───────┘         └──────┬───────┘        └──────┬───────┘ │
│         │                        │                       │          │
└─────────┼────────────────────────┼───────────────────────┼──────────┘
          │                        │                       │
          │                        │                       │
┌─────────▼────────────────────────▼───────────────────────▼──────────┐
│                      GOLANG BACKEND (Orchestrator)                   │
│                                                                       │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │                    HTTP Server (Gin)                           │ │
│  │  • /webhook/telegram  • /webhook/github  • /webhook/gitlab    │ │
│  │  • /webhook/memos     • /health          • /ready             │ │
│  └────────────────────────────────────────────────────────────────┘ │
│                                                                       │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │                    Agent Orchestrator (ReAct)                  │ │
│  │  • Session Memory    • Tool Registry    • Multi-step Reasoning│ │
│  └────────────────────────────────────────────────────────────────┘ │
│                                                                       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────┐ │
│  │   Task UC    │  │ Automation UC│  │   Checklist Service      │ │
│  │ • CreateBulk │  │ • Webhook    │  │ • Parse Markdown         │ │
│  │ • Search     │  │   Processing │  │ • Update Checkboxes      │ │
│  │ • AnswerQuery│  │ • Task Match │  │ • Progress Tracking      │ │
│  └──────────────┘  └──────────────┘  └──────────────────────────┘ │
│                                                                       │
└───────────────────────────────────────────────────────────────────────┘
          │                        │                       │
          │                        │                       │
┌─────────▼────────────────────────▼───────────────────────▼──────────┐
│                      EXTERNAL SERVICES LAYER                         │
│                                                                       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────────┐ │
│  │    Memos     │  │   Qdrant     │  │   Google Calendar        │ │
│  │  (Storage)   │  │  (Vector DB) │  │   (Scheduler)            │ │
│  │              │  │              │  │                          │ │
│  │ • Tasks      │  │ • Embeddings │  │ • Events                 │ │
│  │ • Markdown   │  │ • Semantic   │  │ • Conflict Detection     │ │
│  │ • Tags       │  │   Search     │  │ • Deep Links             │ │
│  └──────────────┘  └──────────────┘  └──────────────────────────┘ │
│                                                                       │
│  ┌──────────────┐  ┌──────────────┐                                │
│  │  LLM Providers│ │  Voyage AI   │                                │
│  │ (Qwen/Gemini)│  │ (Embeddings) │                                │
│  │              │  │              │                                │
│  │ • Reasoning  │  │ • voyage-3   │                                │
│  │ • Parsing    │  │ • 1024-dim   │                                │
│  │ • Tool Call  │  │ • Multilang  │                                │
│  └──────────────┘  └──────────────┘                                │
└───────────────────────────────────────────────────────────────────────┘
```

### 2.2. Các Thành phần Chính

#### 2.2.1. User Interface Layer

**Telegram Bot**
- Giao diện chat duy nhất cho người dùng
- Hỗ trợ text và voice input
- Webhook-based (không polling)
- Commands: `/start`, `/help`, `/search`, `/ask`, `/check`, `/progress`

**Git Webhooks (GitHub/GitLab)**
- Nhận events: push, pull_request, merge_request
- HMAC signature verification
- Rate limiting: 60 requests/minute
- Trigger automation workflows

#### 2.2.2. Golang Backend (Orchestrator)

**HTTP Server (Gin Framework)**
- RESTful endpoints cho webhooks
- Health checks: `/health`, `/ready`, `/live`
- Graceful shutdown với timeout
- CORS và recovery middleware

**Agent Orchestrator (ReAct Framework)**
- Multi-step reasoning: Reason → Act → Observe
- Session memory: 5-turn conversation history
- Tool registry: Dynamic tool registration
- Timeout protection: Max 5 agent steps

**Business Logic Layer**
- Task UseCase: CreateBulk, Search, AnswerQuery
- Automation UseCase: Webhook processing, task matching
- Checklist Service: Markdown parsing, progress tracking

#### 2.2.3. External Services Layer

**Memos (Self-hosted Storage)**
- Markdown-native task storage
- Tag-based organization
- REST API với access token
- Webhook support cho sync

**Qdrant (Vector Database)**
- 1024-dimensional embeddings
- Semantic search với cosine similarity
- Tag-based filtering
- Collection: `tasks`

**Google Calendar (Optional)**
- OAuth2 authentication
- Event creation với deep links
- Conflict detection
- Timezone-aware scheduling

**LLM Providers (DeepSeek/Gemini/Qwen)**
- Multi-provider support with fallback
- Primary: `deepseek-chat` (priority 1)
- Secondary: `gemini-2.5-flash` (priority 2)
- Tertiary: `qwen-turbo` (priority 3)
- Function calling support
- 1M tokens context window (Gemini)
- Rate limit: varies by provider

**Voyage AI (Embeddings)**
- Model: `voyage-3`
- 1024 dimensions
- Multilingual support
- Rate limit: 100M tokens/month (FREE)

---

## 3. Luồng Dữ liệu (Data Flow)

### 3.1. Write Flow - Tạo Task

```
User: "Deadline dự án ABC vào 15/3, ước tính 4 giờ"
  │
  ▼
[Telegram Bot] POST /webhook/telegram
  │
  ▼
[Telegram Handler] Parse update, extract message
  │
  ▼
[Task UseCase] CreateBulk(rawText)
  │
  ├─► [LLM Provider] Parse natural language → JSON array
  │   Response: [{title: "Deadline dự án ABC", dueDate: "2026-03-15", ...}]
  │
  ├─► [DateMath Parser] Resolve relative dates → absolute timestamps
  │   "15/3" → 2026-03-15T23:59:59+07:00
  │
  ├─► [Memos API] CreateTask(content, tags)
  │   Response: {id: "memos/123", uid: "abc123", url: "http://..."}
  │
  ├─► [Voyage AI] Generate embedding(content)
  │   Response: [0.123, -0.456, ...] (1024 dims)
  │
  ├─► [Qdrant] Upsert point(id, vector, payload)
  │   Stored: {memo_id: "abc123", tags: [...], content: "..."}
  │
  └─► [Google Calendar] CreateEvent(title, start, end, description)
      Response: {htmlLink: "https://calendar.google.com/..."}
  │
  ▼
[Telegram Bot] SendMessage(chatID, "✅ Đã tạo task!\n📝 Memo: ...\n📅 Calendar: ...")
```

**Đặc điểm:**
- **Bulk processing**: Một message có thể tạo nhiều tasks
- **Graceful degradation**: Calendar fail không ảnh hưởng task creation
- **Non-blocking**: Embedding fail chỉ log warning
- **Async response**: Telegram webhook trả 200 ngay, xử lý trong goroutine

### 3.2. Read Flow - Tìm kiếm Ngữ nghĩa

```
User: /search deadline tuần này
  │
  ▼
[Telegram Handler] Extract query: "deadline tuần này"
  │
  ▼
[Task UseCase] Search(query, limit=10)
  │
  ├─► [Voyage AI] Generate embedding(query)
  │   Response: [0.789, -0.234, ...] (1024 dims)
  │
  ├─► [Qdrant] Search(vector, limit=10, filter={tags: [...]})
  │   Response: [{id: "abc123", score: 0.92, payload: {...}}, ...]
  │
  └─► [Memos API] GetTask(id) for each result
      Response: [{content: "...", tags: [...], url: "..."}, ...]
  │
  ▼
[Telegram Bot] SendMessage(chatID, "Tìm thấy 3 tasks:\n1. Deadline dự án ABC (score: 0.92)\n...")
```

**Đặc điểm:**
- **Semantic matching**: Không cần khớp từ khóa chính xác
- **Score-based ranking**: Kết quả sắp xếp theo similarity score
- **Tag filtering**: Có thể filter theo tags (optional)
- **Fast response**: <500ms cho 10 results

### 3.3. Agent Flow - Intelligent Query

```
User: /ask Tôi có deadline nào trong tuần này?
  │
  ▼
[Telegram Handler] Route to Agent Orchestrator
  │
  ▼
[Agent Orchestrator] ProcessQuery(userID, query)
  │
  ├─► [Session Memory] Load conversation history (5 turns)
  │
  ├─► [LLM Provider] GenerateContent(systemPrompt + history + query + tools)
  │   LLM thinks: "User asks about deadlines this week"
  │   LLM decides: "I need to call search_tasks tool"
  │   Response: {functionCall: {name: "search_tasks", args: {query: "deadline", ...}}}
  │
  ├─► [Tool Registry] Get("search_tasks") → Execute(args)
  │   │
  │   ├─► [Task UseCase] Search(query="deadline", tags=[], limit=10)
  │   │   (Same flow as 3.2)
  │   │
  │   └─► Tool returns: [{title: "Deadline dự án ABC", dueDate: "2026-03-15", ...}]
  │
  ├─► [LLM Provider] GenerateContent(history + toolResult)
  │   LLM synthesizes: "Bạn có 1 deadline trong tuần này: Dự án ABC vào 15/3"
  │   Response: {text: "Bạn có 1 deadline..."}
  │
  └─► [Session Memory] Save turn (user query + assistant response)
  │
  ▼
[Telegram Bot] SendMessage(chatID, "Bạn có 1 deadline trong tuần này: Dự án ABC vào 15/3")
```

**Đặc điểm:**
- **Multi-step reasoning**: Agent có thể gọi nhiều tools liên tiếp
- **Context-aware**: Nhớ 5 turns trước đó
- **Tool chaining**: Có thể gọi search_tasks → check_calendar → update_checklist
- **Timeout protection**: Max 5 steps để tránh infinite loop

### 3.4. Automation Flow - Git Webhook

```
[GitHub] PR #123 merged
  │
  ▼
[GitHub Webhook] POST /webhook/github
  Headers: X-Hub-Signature-256, X-GitHub-Event
  Body: {action: "closed", pull_request: {merged: true, number: 123, ...}}
  │
  ▼
[Webhook Handler] Verify HMAC signature
  │
  ├─► [Security] ValidateGitHubSignature(body, signature)
  │   ✓ Signature valid
  │
  ├─► [Security] CheckRateLimit("github")
  │   ✓ Within limit (60 req/min)
  │
  └─► [Webhook Parser] ParsePushEvent(body)
      Response: {eventType: "pull_request", action: "merged", prNumber: 123, ...}
  │
  ▼
[Automation UseCase] ProcessWebhook(event)
  │
  ├─► [Task Matcher] FindMatchingTasks(event)
  │   │
  │   ├─► [Qdrant] Search by tags: #pr/123
  │   │   Response: [{memo_id: "abc123", score: 1.0, ...}]
  │   │
  │   └─► [Memos API] GetTask("abc123")
  │       Response: {content: "- [ ] Review code\n- [ ] Merge PR #123\n...", ...}
  │
  ├─► [Checklist Service] ParseMarkdown(content)
  │   Response: [{text: "Review code", checked: false}, {text: "Merge PR #123", checked: false}]
  │
  ├─► [Checklist Service] UpdateChecklistItem(content, "Merge PR #123", checked=true)
  │   Response: "- [x] Review code\n- [x] Merge PR #123\n..."
  │
  └─► [Memos API] UpdateTask("abc123", newContent)
      ✓ Task updated
  │
  ▼
[Webhook Handler] Return 200 OK
```

**Đặc điểm:**
- **Tag-based matching**: Dùng `#pr/123` để link task với PR
- **Semantic fallback**: Nếu không có tag, dùng semantic search
- **Partial matching**: "Merge PR" khớp với "Merge PR #123"
- **Idempotent**: Gọi nhiều lần không tạo duplicate updates

### 3.5. Sync Flow - Memos Webhook

```
[Memos] User edits task in web UI
  │
  ▼
[Memos Webhook] POST /webhook/memos
  Body: {activityType: "memos.memo.updated", memo: {uid: "abc123", content: "...", ...}}
  │
  ▼
[Sync Handler] HandleMemosWebhook(payload)
  │
  ├─► Return 200 OK immediately (non-blocking)
  │
  └─► [Background Goroutine] syncWithRetry(memoID)
      │
      ├─► [Memos API] GetTask("abc123")
      │   Response: {content: "Updated content...", tags: [...], ...}
      │
      ├─► [Voyage AI] Generate embedding(content)
      │   Response: [0.456, -0.789, ...] (1024 dims)
      │
      └─► [Qdrant] Upsert point(id="abc123", vector, payload)
          ✓ Vector updated
```

**Đặc điểm:**
- **Async processing**: Webhook trả 200 ngay, sync trong background
- **Retry logic**: Exponential backoff (2s, 4s, 8s)
- **Timeout protection**: 2 minutes max per sync
- **Data consistency**: Đảm bảo Qdrant luôn sync với Memos

---

## 4. Kiến trúc Code (Code Architecture)

### 4.1. Clean Architecture Pattern

Hệ thống tuân theo **Clean Architecture** với 4 layers:

```
┌─────────────────────────────────────────────────────────────┐
│                    DELIVERY LAYER                           │
│  • internal/task/delivery/telegram/                         │
│  • internal/webhook/                                        │
│  • internal/httpserver/                                     │
│  Responsibility: HTTP handlers, webhook parsers             │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                    USECASE LAYER                            │
│  • internal/task/usecase/                                   │
│  • internal/automation/                                     │
│  • internal/agent/orchestrator/                             │
│  Responsibility: Business logic, orchestration              │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                   REPOSITORY LAYER                          │
│  • internal/task/repository/memos/                          │
│  • internal/task/repository/qdrant/                         │
│  Responsibility: Data access, external API calls            │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│                     DOMAIN LAYER                            │
│  • internal/model/                                          │
│  • internal/task/types.go                                   │
│  Responsibility: Core entities, interfaces                  │
└─────────────────────────────────────────────────────────────┘
```

### 4.2. Cấu trúc Thư mục

```
autonomous-task-management/
├── cmd/
│   └── api/
│       ├── main.go              # Entry point, dependency injection
│       ├── Dockerfile           # Production build
│       └── Dockerfile.dev       # Development với Air live-reload
│
├── config/
│   ├── config.go                # Config loader (Viper)
│   ├── config.yaml              # Default config
│   └── config.example.yaml      # Template
│
├── internal/                    # Private application code
│   ├── agent/                   # AI Agent components
│   │   ├── orchestrator/        # ReAct framework implementation
│   │   │   ├── orchestrator.go  # Main orchestrator logic
│   │   │   ├── types.go         # SessionMemory, AgentStep
│   │   │   └── new.go           # Constructor
│   │   ├── tools/               # Agent tools (function calling)
│   │   │   ├── search_tasks.go
│   │   │   ├── check_calendar.go
│   │   │   ├── get_checklist_progress.go
│   │   │   └── update_checklist_item.go
│   │   └── types.go             # Tool interface, ToolRegistry
│   │
│   ├── task/                    # Task domain
│   │   ├── delivery/
│   │   │   └── telegram/        # Telegram bot handler
│   │   │       ├── handler.go   # Webhook handler, command routing
│   │   │       └── new.go       # Constructor
│   │   ├── usecase/             # Business logic
│   │   │   ├── create_bulk.go   # Parse + create tasks
│   │   │   ├── search.go        # Semantic search
│   │   │   ├── answer_query.go  # RAG-based Q&A
│   │   │   └── helpers.go       # Date parsing, markdown builder
│   │   ├── repository/          # Data access
│   │   │   ├── memos/           # Memos API client
│   │   │   │   ├── client.go    # HTTP client
│   │   │   │   └── task.go      # CRUD operations
│   │   │   ├── qdrant/          # Qdrant client
│   │   │   │   └── task.go      # Vector operations
│   │   │   └── interface.go     # Repository interfaces
│   │   ├── interface.go         # UseCase interface
│   │   └── types.go             # DTOs (Input/Output)
│   │
│   ├── automation/              # Webhook automation
│   │   ├── usecase.go           # ProcessWebhook logic
│   │   ├── matcher.go           # Task matching (tags + semantic)
│   │   └── types.go             # ProcessWebhookInput/Output
│   │
│   ├── checklist/               # Checklist service
│   │   ├── service.go           # Markdown parsing, checkbox update
│   │   └── types.go             # ChecklistItem
│   │
│   ├── sync/                    # Memos webhook handler
│   │   ├── handler.go           # HandleMemosWebhook
│   │   └── types.go             # MemosWebhookPayload
│   │
│   ├── webhook/                 # Git webhook handlers
│   │   ├── handler.go           # HandleGitHubWebhook, HandleGitLabWebhook
│   │   ├── github.go            # GitHub parser
│   │   ├── gitlab.go            # GitLab parser
│   │   └── security.go          # HMAC verification, rate limiting
│   │
│   ├── httpserver/              # HTTP server
│   │   ├── httpserver.go        # Gin server setup
│   │   ├── handler.go           # Route registration
│   │   ├── health.go            # Health checks
│   │   └── new.go               # Constructor
│   │
│   └── model/                   # Domain models
│       ├── task.go              # Task entity
│       ├── event.go             # WebhookEvent
│       ├── scope.go             # Scope (UserID context)
│       └── constant.go          # Constants
│
├── pkg/                         # Shared libraries (reusable)
│   ├── deepseek/                # DeepSeek LLM client
│   │   ├── client.go            # HTTP client
│   │   ├── types.go             # Request/Response types
│   │   └── constant.go          # Constants
│   ├── gemini/                  # Gemini LLM client
│   │   ├── client.go            # HTTP client
│   │   ├── types.go             # Request/Response types
│   │   └── prompt.go            # System prompts
│   ├── qwen/                    # Qwen LLM client
│   │   ├── client.go            # HTTP client
│   │   ├── types.go             # Request/Response types
│   │   └── constant.go          # Constants
│   ├── llmprovider/             # LLM provider manager
│   │   ├── provider.go          # Provider interface
│   │   ├── manager.go           # Multi-provider manager
│   │   ├── adapter.go           # Provider adapters
│   │   ├── factory.go           # Provider factory
│   │   └── errors.go            # Error types
│   ├── voyage/                  # Voyage AI client
│   │   ├── client.go            # Embedding API
│   │   └── types.go             # EmbedRequest/Response
│   ├── qdrant/                  # Qdrant client
│   │   ├── client.go            # HTTP client
│   │   └── types.go             # Collection, Point types
│   ├── gcalendar/               # Google Calendar client
│   │   ├── client.go            # OAuth2 + Calendar API
│   │   └── types.go             # Event types
│   ├── telegram/                # Telegram Bot API client
│   │   ├── bot.go               # SendMessage, SetWebhook
│   │   └── types.go             # Update, Message types
│   ├── datemath/                # Date parsing library
│   │   ├── parser.go            # Parse relative dates
│   │   └── types.go             # ParsedDate
│   ├── log/                     # Structured logging (Zap)
│   │   ├── log.go               # Logger interface
│   │   └── type.go              # ZapConfig
│   └── response/                # HTTP response helpers
│       ├── response.go          # OK, Error wrappers
│       └── type.go              # APIResponse
│
├── documents/                   # Documentation
│   ├── master-plan.md
│   ├── architecture-overview.md # This file
│   ├── phase-*.md               # Implementation plans
│   └── convention/              # Coding conventions
│
├── manifests/                   # Deployment configs
│   ├── docker-compose/
│   │   ├── docker-compose.yml
│   │   └── docker-compose.override.yml
│   └── tags-schema.json         # Tag taxonomy
│
├── scripts/                     # Utility scripts
│   ├── verify-setup.sh
│   └── backfill-embeddings/
│
├── .env.example                 # Environment template
├── .gitignore
├── go.mod
├── go.sum
├── Makefile                     # Build commands
└── README.md
```

### 4.3. Dependency Injection

**File:** `cmd/api/main.go`

```go
func main() {
    // 1. Load config
    cfg := config.Load()
    
    // 2. Initialize logger
    logger := log.Init(cfg.Logger)
    
    // 3. Initialize external clients (pkg/)
    llmManager := llmprovider.InitializeProviders(cfg.LLM)
    voyageClient := voyage.New(cfg.Voyage.APIKey)
    qdrantClient := qdrant.NewClient(cfg.Qdrant.URL)
    telegramBot := telegram.NewBot(cfg.Telegram.BotToken)
    calendarClient := gcalendar.NewClient(cfg.GoogleCalendar.CredentialsPath)
    
    // 4. Initialize repositories (internal/task/repository/)
    memosRepo := memos.NewRepository(cfg.Memos.URL, cfg.Memos.AccessToken)
    qdrantRepo := qdrant.NewRepository(qdrantClient, voyageClient)
    
    // 5. Initialize services (internal/)
    checklistSvc := checklist.NewService()
    
    // 6. Initialize use cases (internal/*/usecase/)
    taskUC := usecase.New(memosRepo, qdrantRepo, llmManager, calendarClient)
    automationUC := automation.New(memosRepo, qdrantRepo, checklistSvc)
    
    // 7. Initialize agent (internal/agent/)
    toolRegistry := agent.NewToolRegistry()
    toolRegistry.Register(tools.NewSearchTasks(taskUC))
    toolRegistry.Register(tools.NewCheckCalendar(calendarClient))
    toolRegistry.Register(tools.NewGetChecklistProgress(checklistSvc, memosRepo))
    toolRegistry.Register(tools.NewUpdateChecklistItem(checklistSvc, memosRepo))
    
    orchestrator := orchestrator.New(llmManager, toolRegistry, cfg.LLM.Timezone)
    
    // 8. Initialize handlers (internal/*/delivery/)
    telegramHandler := telegram.NewHandler(taskUC, telegramBot, orchestrator, automationUC, checklistSvc)
    webhookHandler := sync.NewHandler(memosRepo, qdrantRepo)
    gitWebhookHandler := webhook.NewHandler(automationUC, cfg.Webhook.Secret)
    
    // 9. Initialize HTTP server (internal/httpserver/)
    httpServer := httpserver.New(logger, telegramHandler, webhookHandler, gitWebhookHandler)
    
    // 10. Start server
    httpServer.Start(cfg.HTTPServer.Port)
}
```

**Đặc điểm:**
- **Constructor injection**: Tất cả dependencies truyền qua constructor
- **Interface-based**: UseCase và Repository đều là interfaces
- **Single responsibility**: Mỗi component có một nhiệm vụ rõ ràng
- **Testable**: Dễ dàng mock dependencies cho unit tests

---

## 5. Các Thành phần Chi tiết (Component Deep Dive)

### 5.1. Agent Orchestrator (ReAct Framework)

**File:** `internal/agent/orchestrator/orchestrator.go`

**Nhiệm vụ:**
- Điều phối multi-step reasoning của AI agent
- Quản lý session memory (conversation history)
- Gọi tools dựa trên LLM function calling
- Timeout protection và error handling

**ReAct Loop:**

```go
func (o *Orchestrator) ProcessQuery(ctx context.Context, userID, query string) (string, error) {
    session := o.getSession(userID)
    
    // Inject temporal context (current date, week boundaries)
    queryWithContext := injectTimeContext(query, o.timezone)
    
    // Add user message to history
    session.Messages = append(session.Messages, gemini.Content{
        Role: "user",
        Parts: []gemini.Part{{Text: queryWithContext}},
    })
    
    // ReAct loop: max 5 steps
    for step := 0; step < MaxAgentSteps; step++ {
        // 1. REASON: Ask LLM what to do next
        resp, err := o.llm.GenerateContent(ctx, gemini.GenerateRequest{
            Contents: session.Messages,
            Tools: o.toolRegistry.ToFunctionDefinitions(),
            SystemInstruction: systemPrompt,
        })
        
        // 2. ACT: Execute tool if LLM requests
        if resp.FunctionCall != nil {
            tool := o.toolRegistry.Get(resp.FunctionCall.Name)
            result := tool.Execute(ctx, resp.FunctionCall.Arguments)
            
            // 3. OBSERVE: Add tool result to history
            session.Messages = append(session.Messages, gemini.Content{
                Role: "function",
                Parts: []gemini.Part{{FunctionResponse: result}},
            })
            continue // Loop back to REASON
        }
        
        // No more tools needed, return final answer
        return resp.Text, nil
    }
    
    return "", ErrMaxStepsExceeded
}
```

**Session Memory:**
- Lưu 5 turns gần nhất (user + assistant)
- TTL: 30 minutes
- Cleanup: Goroutine chạy mỗi 5 phút
- Thread-safe: sync.Mutex

**Tools Available:**
1. `search_tasks`: Semantic search trong Qdrant
2. `check_calendar`: Kiểm tra lịch trống
3. `get_checklist_progress`: Xem tiến độ checklist
4. `update_checklist_item`: Tick checkbox

### 5.2. Task UseCase - CreateBulk

**File:** `internal/task/usecase/create_bulk.go`

**Pipeline:**

```
Input: "Deadline dự án ABC vào 15/3, ước tính 4 giờ"
  │
  ▼
[1. Parse with LLM]
  Prompt: "Parse this text into JSON array of tasks..."
  Response: [
    {
      "title": "Deadline dự án ABC",
      "dueDate": "15/3",
      "estimatedDurationMinutes": 240,
      "tags": ["#project/abc", "#priority/p0"]
    }
  ]
  │
  ▼
[2. Resolve Dates]
  DateMath.Parse("15/3", timezone="Asia/Ho_Chi_Minh")
  → 2026-03-15T23:59:59+07:00
  │
  ▼
[3. Build Markdown]
  Content: "# Deadline dự án ABC\n\nDue: 2026-03-15\nEstimate: 4h\n\n#project/abc #priority/p0"
  │
  ▼
[4. Create in Memos]
  POST /api/v1/memo
  Headers: Authorization: Bearer <token>
  Body: {content: "...", visibility: "PRIVATE"}
  Response: {name: "memos/123", uid: "abc123", ...}
  │
  ▼
[5. Embed to Qdrant]
  Voyage.Embed(content) → [0.123, -0.456, ...] (1024 dims)
  Qdrant.Upsert(id="abc123", vector, payload={tags, content})
  │
  ▼
[6. Create Calendar Event]
  Calendar.CreateEvent({
    summary: "Deadline dự án ABC",
    start: 2026-03-15T19:00:00+07:00,  // 4h before deadline
    end: 2026-03-15T23:00:00+07:00,
    description: "📝 Memos: http://localhost:5230/m/abc123"
  })
  Response: {htmlLink: "https://calendar.google.com/..."}
  │
  ▼
Output: CreatedTask{
  MemoID: "memos/123",
  MemoURL: "http://localhost:5230/m/abc123",
  CalendarLink: "https://calendar.google.com/...",
  Title: "Deadline dự án ABC"
}
```

**Error Handling:**
- LLM parse fail → Return `ErrNoTasksParsed`
- Memos create fail → Skip task, log error, continue
- Qdrant embed fail → Log warning, continue (non-blocking)
- Calendar create fail → Log warning, continue (graceful degradation)

### 5.3. Automation UseCase - Webhook Processing

**File:** `internal/automation/usecase.go`

**Task Matching Strategy:**

```go
func (uc *usecase) ProcessWebhook(ctx context.Context, event model.WebhookEvent) error {
    // 1. Extract identifiers from event
    identifiers := []string{
        fmt.Sprintf("#pr/%d", event.PRNumber),
        fmt.Sprintf("#issue/%d", event.IssueNumber),
        event.Branch,
        event.Repository,
    }
    
    // 2. Search by tags (exact match)
    tagMatches := uc.vectorRepo.SearchTasksWithFilter(ctx, repository.SearchTasksOptions{
        Filter: repository.PayloadFilter{
            Should: []repository.Condition{
                {Key: "tags", Match: repository.MatchAny{Values: identifiers}},
            },
        },
        Limit: 50,
    })
    
    // 3. Fallback: Semantic search
    if len(tagMatches) == 0 {
        query := fmt.Sprintf("%s %s", event.Repository, event.Message)
        tagMatches = uc.vectorRepo.SearchTasks(ctx, repository.SearchTasksOptions{
            Query: query,
            Limit: 10,
        })
    }
    
    // 4. Update each matched task
    for _, match := range tagMatches {
        task := uc.memosRepo.GetTask(ctx, match.MemoID)
        
        // Parse checklist
        items := uc.checklistSvc.ParseMarkdown(task.Content)
        
        // Find matching checkbox
        for _, item := range items {
            if strings.Contains(item.Text, fmt.Sprintf("PR #%d", event.PRNumber)) {
                // Update checkbox
                newContent := uc.checklistSvc.UpdateChecklistItem(
                    task.Content,
                    item.Text,
                    true, // checked
                )
                
                // Save to Memos
                uc.memosRepo.UpdateTask(ctx, task.ID, newContent)
            }
        }
    }
}
```

**Matching Priority:**
1. **Tag-based** (exact): `#pr/123`, `#issue/456`
2. **Semantic** (fuzzy): Cosine similarity > 0.7
3. **Partial text** (checkbox): "Merge PR" matches "Merge PR #123"

### 5.4. Checklist Service

**File:** `internal/checklist/service.go`

**Markdown Parsing:**

```go
func (s *Service) ParseMarkdown(content string) []ChecklistItem {
    lines := strings.Split(content, "\n")
    items := []ChecklistItem{}
    
    for _, line := range lines {
        // Match: - [ ] Task or - [x] Task
        if match := checkboxRegex.FindStringSubmatch(line); match != nil {
            items = append(items, ChecklistItem{
                Text: strings.TrimSpace(match[2]),
                Checked: match[1] == "x" || match[1] == "X",
                Line: line,
            })
        }
    }
    
    return items
}
```

**Checkbox Update:**

```go
func (s *Service) UpdateChecklistItem(content, itemText string, checked bool) string {
    lines := strings.Split(content, "\n")
    
    for i, line := range lines {
        if strings.Contains(line, itemText) {
            if checked {
                lines[i] = strings.Replace(line, "- [ ]", "- [x]", 1)
            } else {
                lines[i] = strings.Replace(line, "- [x]", "- [ ]", 1)
            }
        }
    }
    
    return strings.Join(lines, "\n")
}
```

**Progress Tracking:**

```go
func (s *Service) GetProgress(content string) (completed, total int) {
    items := s.ParseMarkdown(content)
    total = len(items)
    
    for _, item := range items {
        if item.Checked {
            completed++
        }
    }
    
    return completed, total
}
```

### 5.5. Repository Pattern

**Interface:** `internal/task/repository/interface.go`

```go
type MemosRepository interface {
    CreateTask(ctx context.Context, opt CreateTaskOptions) (model.Task, error)
    GetTask(ctx context.Context, id string) (model.Task, error)
    ListTasks(ctx context.Context, opt ListTasksOptions) ([]model.Task, error)
    UpdateTask(ctx context.Context, id string, content string) error
}

type VectorRepository interface {
    EmbedTask(ctx context.Context, task model.Task) error
    SearchTasks(ctx context.Context, opt SearchTasksOptions) ([]SearchResult, error)
    DeleteTask(ctx context.Context, taskID string) error
}
```

**Implementation:** `internal/task/repository/memos/task.go`

```go
func (r *Repository) CreateTask(ctx context.Context, opt CreateTaskOptions) (model.Task, error) {
    url := fmt.Sprintf("%s/api/v1/memo", r.baseURL)
    
    body := map[string]interface{}{
        "content": opt.Content,
        "visibility": opt.Visibility,
    }
    
    req, _ := http.NewRequestWithContext(ctx, "POST", url, jsonBody(body))
    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", r.accessToken))
    req.Header.Set("Content-Type", "application/json")
    
    resp, err := r.httpClient.Do(req)
    // ... error handling
    
    var memoResp MemosAPIResponse
    json.NewDecoder(resp.Body).Decode(&memoResp)
    
    return model.Task{
        ID: memoResp.Name,
        UID: memoResp.UID,
        Content: memoResp.Content,
        Tags: extractTags(memoResp.Content),
        MemoURL: fmt.Sprintf("%s/m/%s", r.externalURL, memoResp.UID),
    }, nil
}
```

**Benefits:**
- **Testability**: Dễ mock cho unit tests
- **Flexibility**: Dễ swap implementation (Memos → Notion)
- **Separation**: Business logic không phụ thuộc vào data source

---

## 6. Công nghệ và Thư viện (Technology Stack)

### 6.1. Backend Framework

**Go 1.21+**
- Compiled language: Fast, low memory footprint
- Goroutines: Efficient concurrency
- Static typing: Compile-time error detection
- Cross-platform: Linux, macOS, Windows

**Gin Web Framework**
- High performance: 40x faster than Martini
- Middleware support: CORS, recovery, logging
- JSON validation: Struct tags
- Route grouping: Clean API organization

### 6.2. AI & ML Services

**Google Gemini 2.5 Flash**
- **Model**: `gemini-2.5-flash`
- **Context**: 1M tokens input, 8K output
- **Features**: Function calling, multimodal
- **Rate limit**: 15 req/min (FREE), 1000 req/min (Paid)
- **Use cases**: Task parsing, reasoning, tool orchestration (secondary fallback)

**DeepSeek Chat**
- **Model**: `deepseek-chat`
- **Context**: 64K tokens input
- **Features**: Function calling, fast inference
- **Rate limit**: varies by plan
- **Use cases**: Task parsing, reasoning, tool orchestration (primary)

**Alibaba Qwen**
- **Model**: `qwen-turbo`
- **Context**: 8K tokens input
- **Features**: Function calling, multilingual
- **Rate limit**: varies by plan
- **Use cases**: Task parsing, reasoning (tertiary fallback)

**Voyage AI**
- **Model**: `voyage-3`
- **Dimensions**: 1024
- **Features**: Multilingual, SOTA performance
- **Rate limit**: 100M tokens/month (FREE)
- **Use cases**: Semantic search, RAG, clustering

### 6.3. Storage & Database

**Memos (Self-hosted)**
- **Type**: Markdown-native note-taking
- **Storage**: SQLite (default) or PostgreSQL
- **API**: RESTful with JWT auth
- **Features**: Tags, webhooks, full-text search
- **Deployment**: Docker, single binary

**Qdrant (Vector Database)**
- **Type**: Vector similarity search
- **Storage**: HNSW index
- **API**: HTTP + gRPC
- **Features**: Filtering, payload storage, clustering
- **Deployment**: Docker, Kubernetes

### 6.4. External Integrations

**Telegram Bot API**
- **Protocol**: HTTPS webhooks
- **Features**: Text, voice, inline keyboards
- **Rate limit**: 30 msg/sec per chat
- **Security**: Webhook secret verification

**Google Calendar API**
- **Auth**: OAuth2 (Desktop App or Service Account)
- **Features**: Event CRUD, conflict detection
- **Rate limit**: 1M requests/day (FREE)
- **Timezone**: IANA timezone support

**GitHub/GitLab Webhooks**
- **Events**: push, pull_request, merge_request
- **Security**: HMAC-SHA256 signature
- **Delivery**: Retry with exponential backoff
- **Payload**: JSON with full event context

### 6.5. Libraries & Tools

**Configuration**
- `spf13/viper`: Config management (YAML, ENV)
- `joho/godotenv`: .env file loader

**Logging**
- `uber-go/zap`: Structured logging
- Levels: DEBUG, INFO, WARN, ERROR
- Outputs: Console, JSON

**HTTP Client**
- `net/http`: Standard library
- Timeout: 30s default
- Retry: Exponential backoff

**Date/Time**
- `time`: Standard library
- Timezone: IANA database
- Parsing: Custom datemath package

**Testing**
- `testing`: Standard library
- `stretchr/testify`: Assertions, mocks
- Coverage: `go test -cover`

**Development**
- `cosmtrek/air`: Live reload
- `swaggo/swag`: Swagger docs generation
- `golangci-lint`: Linting

### 6.6. Deployment

**Docker**
- Multi-stage builds: Builder + runtime
- Base images: `golang:1.21-alpine`, `alpine:latest`
- Volumes: Persistent data for Memos, Qdrant
- Networks: Bridge network for inter-service communication

**Docker Compose**
- Services: backend, memos, qdrant, ngrok
- Healthchecks: Ensure services ready before start
- Depends_on: Service startup order
- Override: Development vs production configs

**Ngrok (Development)**
- Expose localhost to internet
- HTTPS tunnels for webhooks
- Dashboard: <http://localhost:4040>
- Alternative: Cloudflare Tunnel, Tailscale Funnel

---

## 7. Patterns và Best Practices

### 7.1. Design Patterns

**Repository Pattern**
- Abstraction: Interface-based data access
- Implementation: Memos, Qdrant repositories
- Benefits: Testability, flexibility, separation of concerns

**Dependency Injection**
- Constructor injection: All dependencies via `New()` functions
- Interface-based: UseCase, Repository interfaces
- Benefits: Testability, loose coupling

**Factory Pattern**
- Tool registry: Dynamic tool registration
- Client factories: Gemini, Voyage, Qdrant clients
- Benefits: Extensibility, encapsulation

**Strategy Pattern**
- Task matching: Tag-based vs semantic
- Date parsing: Relative vs absolute
- Benefits: Flexibility, maintainability

**Observer Pattern**
- Webhooks: Event-driven updates
- Memos sync: Automatic vector updates
- Benefits: Loose coupling, scalability

### 7.2. Error Handling

**Sentinel Errors**
```go
var (
    ErrEmptyInput = errors.New("empty input")
    ErrNoTasksParsed = errors.New("no tasks parsed")
    ErrMaxStepsExceeded = errors.New("max agent steps exceeded")
)
```

**Error Wrapping**
```go
if err := repo.CreateTask(ctx, opt); err != nil {
    return fmt.Errorf("failed to create task: %w", err)
}
```

**Graceful Degradation**
```go
// Calendar creation fails → Log warning, continue
if calendarLink, err := createCalendarEvent(task); err != nil {
    logger.Warnf(ctx, "calendar creation failed (non-fatal): %v", err)
    calendarLink = "" // Empty link, task still created
}
```

### 7.3. Concurrency

**Goroutines**
```go
// Telegram webhook: Return 200 immediately, process in background
go func() {
    bgCtx := context.Background()
    if err := processMessage(bgCtx, msg); err != nil {
        logger.Errorf(bgCtx, "background processing failed: %v", err)
    }
}()
```

**Mutex for Shared State**
```go
type Orchestrator struct {
    sessionCache map[string]*SessionMemory
    cacheMutex   sync.Mutex
}

func (o *Orchestrator) getSession(userID string) *SessionMemory {
    o.cacheMutex.Lock()
    defer o.cacheMutex.Unlock()
    return o.sessionCache[userID]
}
```

**Context for Cancellation**
```go
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
defer cancel()

if err := vectorRepo.EmbedTask(ctx, task); err != nil {
    // Timeout or cancellation
}
```

### 7.4. Testing

**Unit Tests**
```go
func TestCreateBulk(t *testing.T) {
    // Arrange
    mockRepo := &MockMemosRepository{}
    mockRepo.On("CreateTask", mock.Anything, mock.Anything).Return(model.Task{}, nil)
    
    uc := usecase.New(mockRepo, nil, nil, nil)
    
    // Act
    output, err := uc.CreateBulk(ctx, scope, input)
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, 1, output.TaskCount)
    mockRepo.AssertExpectations(t)
}
```

**Integration Tests**
```go
func TestMemosRepository_CreateTask(t *testing.T) {
    // Requires Memos running on localhost:5230
    repo := memos.NewRepository("http://localhost:5230", "test-token")
    
    task, err := repo.CreateTask(ctx, repository.CreateTaskOptions{
        Content: "Test task",
        Visibility: "PRIVATE",
    })
    
    assert.NoError(t, err)
    assert.NotEmpty(t, task.ID)
}
```

**Table-Driven Tests**
```go
func TestDateMathParser(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected time.Time
    }{
        {"tomorrow", "ngày mai", time.Now().AddDate(0, 0, 1)},
        {"next week", "tuần sau", time.Now().AddDate(0, 0, 7)},
        {"absolute", "15/3", time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := parser.Parse(tt.input)
            assert.Equal(t, tt.expected.Day(), result.Day())
        })
    }
}
```

### 7.5. Security

**HMAC Signature Verification**
```go
func ValidateGitHubSignature(body []byte, signature string) error {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(body)
    expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
    
    if !hmac.Equal([]byte(expected), []byte(signature)) {
        return ErrInvalidSignature
    }
    return nil
}
```

**Rate Limiting**
```go
type RateLimiter struct {
    requests map[string][]time.Time
    limit    int
    window   time.Duration
}

func (rl *RateLimiter) CheckRateLimit(key string) error {
    now := time.Now()
    requests := rl.requests[key]
    
    // Remove old requests outside window
    valid := []time.Time{}
    for _, t := range requests {
        if now.Sub(t) < rl.window {
            valid = append(valid, t)
        }
    }
    
    if len(valid) >= rl.limit {
        return ErrRateLimitExceeded
    }
    
    rl.requests[key] = append(valid, now)
    return nil
}
```

**Environment Variables**
```bash
# Never commit secrets to Git
TELEGRAM_BOT_TOKEN=secret
MEMOS_ACCESS_TOKEN=secret
GEMINI_API_KEY=secret
WEBHOOK_SECRET=secret
```

---

## 8. Deployment và Operations

### 8.1. Local Development

**Quick Start**
```bash
# 1. Clone repository
git clone <repo-url>
cd autonomous-task-management

# 2. Copy environment template
cp .env.example .env

# 3. Edit .env with your API keys
nano .env

# 4. Start all services
docker compose up -d

# 5. Verify setup
bash scripts/verify-setup.sh

# 6. Access services
# - Memos: http://localhost:5230
# - Qdrant: http://localhost:6333/dashboard
# - Backend: http://localhost:8080
# - Ngrok: http://localhost:4040
```

**Development Mode (Live Reload)**
```bash
# Uses docker-compose.override.yml automatically
docker compose up

# Watch logs
docker compose logs -f backend

# Rebuild after dependency changes
docker compose build backend
docker compose up -d
```

**Makefile Commands**
```bash
make up          # Start all services
make down        # Stop all services
make restart     # Restart backend
make logs        # View backend logs
make build       # Build Go binary
make test        # Run tests
make lint        # Run linter
make clean       # Clean build artifacts
```

### 8.2. Production Deployment

**Environment Variables**
```bash
# Production .env
ENVIRONMENT=production
HTTP_SERVER_MODE=release
LOGGER_LEVEL=info
LOGGER_MODE=production

# External URLs (not localhost)
MEMOS_EXTERNAL_URL=https://memos.yourdomain.com
TELEGRAM_WEBHOOK_URL=https://api.yourdomain.com/webhook/telegram

# Secrets (use secret management)
TELEGRAM_BOT_TOKEN=${SECRET_TELEGRAM_TOKEN}
MEMOS_ACCESS_TOKEN=${SECRET_MEMOS_TOKEN}
GEMINI_API_KEY=${SECRET_GEMINI_KEY}
WEBHOOK_SECRET=${SECRET_WEBHOOK_SECRET}
```

**Docker Compose (Production)**
```yaml
# docker-compose.prod.yml
version: "3.8"

services:
  backend:
    image: your-registry/atm-backend:latest
    restart: always
    environment:
      - ENVIRONMENT=production
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
        reservations:
          cpus: '1'
          memory: 1G
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

  memos:
    image: neosmemo/memos:latest
    restart: always
    volumes:
      - memos-data:/var/opt/memos
    deploy:
      resources:
        limits:
          cpus: '1'
          memory: 1G

  qdrant:
    image: qdrant/qdrant:latest
    restart: always
    volumes:
      - qdrant-data:/qdrant/storage
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 4G
```

**Kubernetes Deployment**
```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: atm-backend
spec:
  replicas: 3
  selector:
    matchLabels:
      app: atm-backend
  template:
    metadata:
      labels:
        app: atm-backend
    spec:
      containers:
      - name: backend
        image: your-registry/atm-backend:latest
        ports:
        - containerPort: 8080
        env:
        - name: MEMOS_URL
          value: "http://memos-service:5230"
        - name: QDRANT_URL
          value: "http://qdrant-service:6333"
        - name: TELEGRAM_BOT_TOKEN
          valueFrom:
            secretKeyRef:
              name: atm-secrets
              key: telegram-token
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "2000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 5
```

### 8.3. Monitoring & Observability

**Health Checks**
```go
// /health - Liveness probe
func (srv *HTTPServer) healthCheck(c *gin.Context) {
    c.JSON(200, gin.H{"status": "ok"})
}

// /ready - Readiness probe
func (srv *HTTPServer) readyCheck(c *gin.Context) {
    // Check dependencies
    if err := srv.checkMemos(); err != nil {
        c.JSON(503, gin.H{"status": "not ready", "reason": "memos unavailable"})
        return
    }
    if err := srv.checkQdrant(); err != nil {
        c.JSON(503, gin.H{"status": "not ready", "reason": "qdrant unavailable"})
        return
    }
    c.JSON(200, gin.H{"status": "ready"})
}
```

**Structured Logging**
```go
// Zap logger with structured fields
logger.Infof(ctx, "task created",
    zap.String("user_id", userID),
    zap.String("memo_id", memoID),
    zap.Int("task_count", count),
    zap.Duration("duration", elapsed),
)
```

**Metrics (Future)**
```go
// Prometheus metrics
var (
    tasksCreated = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "atm_tasks_created_total",
            Help: "Total number of tasks created",
        },
        []string{"user_id"},
    )
    
    agentSteps = prometheus.NewHistogram(
        prometheus.HistogramOpts{
            Name: "atm_agent_steps",
            Help: "Number of steps taken by agent",
            Buckets: []float64{1, 2, 3, 4, 5},
        },
    )
)
```

**Tracing (Future)**
```go
// OpenTelemetry tracing
import "go.opentelemetry.io/otel"

func (uc *implUseCase) CreateBulk(ctx context.Context, ...) {
    ctx, span := otel.Tracer("task").Start(ctx, "CreateBulk")
    defer span.End()
    
    // ... business logic
    
    span.SetAttributes(
        attribute.String("user_id", sc.UserID),
        attribute.Int("task_count", len(tasks)),
    )
}
```

### 8.4. Backup & Recovery

**Memos Backup**
```bash
# Backup SQLite database
docker compose exec memos sqlite3 /var/opt/memos/memos_prod.db ".backup /var/opt/memos/backup.db"
docker compose cp memos:/var/opt/memos/backup.db ./backups/memos-$(date +%Y%m%d).db

# Restore
docker compose cp ./backups/memos-20260227.db memos:/var/opt/memos/restore.db
docker compose exec memos sqlite3 /var/opt/memos/memos_prod.db ".restore /var/opt/memos/restore.db"
```

**Qdrant Backup**
```bash
# Create snapshot
curl -X POST "http://localhost:6333/collections/tasks/snapshots"

# Download snapshot
curl "http://localhost:6333/collections/tasks/snapshots/snapshot-2026-02-27.snapshot" -o qdrant-backup.snapshot

# Restore
curl -X PUT "http://localhost:6333/collections/tasks/snapshots/upload" \
  --data-binary @qdrant-backup.snapshot
```

**Automated Backup Script**
```bash
#!/bin/bash
# scripts/backup.sh

DATE=$(date +%Y%m%d)
BACKUP_DIR="./backups/$DATE"
mkdir -p "$BACKUP_DIR"

# Backup Memos
docker compose exec memos sqlite3 /var/opt/memos/memos_prod.db ".backup /var/opt/memos/backup.db"
docker compose cp memos:/var/opt/memos/backup.db "$BACKUP_DIR/memos.db"

# Backup Qdrant
curl -X POST "http://localhost:6333/collections/tasks/snapshots"
SNAPSHOT=$(curl -s "http://localhost:6333/collections/tasks/snapshots" | jq -r '.result[0].name')
curl "http://localhost:6333/collections/tasks/snapshots/$SNAPSHOT" -o "$BACKUP_DIR/qdrant.snapshot"

# Upload to S3 (optional)
aws s3 sync "$BACKUP_DIR" "s3://your-bucket/backups/$DATE/"

echo "Backup completed: $BACKUP_DIR"
```

### 8.5. Scaling Considerations

**Horizontal Scaling**
- Backend: Stateless, scale to N replicas
- Session memory: Move to Redis for shared state
- Webhook processing: Use message queue (RabbitMQ, Kafka)

**Vertical Scaling**
- Qdrant: Increase memory for larger vector index
- Memos: Switch from SQLite to PostgreSQL
- Backend: Increase CPU for concurrent requests

**Caching**
- LLM responses: Cache parsed tasks for 1 hour
- Embeddings: Cache vectors for frequently accessed tasks
- Calendar events: Cache availability for 5 minutes

**Rate Limiting**
- Gemini API: 15 req/min (FREE) → Queue requests
- Voyage API: 100M tokens/month → Batch embeddings
- Telegram: 30 msg/sec → Use sendMessage queue

---

## 9. Roadmap và Future Enhancements

### 9.1. Completed Phases

**Phase 1: Infrastructure Setup** ✅
- Docker Compose với healthchecks
- Memos + Qdrant + Backend
- Configuration management
- Development environment với live reload

**Phase 2: Core Engine** ✅
- Telegram bot integration
- LLM-based task parsing
- Bulk task creation
- Google Calendar integration

**Phase 3: RAG & Agent Tools** ✅
- Semantic search với Qdrant
- ReAct agent orchestrator
- Tool registry (search, calendar, checklist)
- Session memory

**Phase 4: Automation & Webhooks** ✅
- GitHub/GitLab webhook handlers
- Task matching (tag-based + semantic)
- Checklist auto-update
- Memos sync webhook

**Phase 5: Hotfixes & Verification** ✅
- HMAC signature verification
- Rate limiting
- Error handling improvements
- Documentation updates

### 9.2. Future Enhancements

**Phase 6: Advanced AI Features**
- [ ] Multi-agent collaboration (planning agent + execution agent)
- [ ] Voice input processing (Whisper API)
- [ ] Image understanding (attach screenshots to tasks)
- [ ] Proactive suggestions ("You have 3 overdue tasks")

**Phase 7: Enhanced Search**
- [ ] Hybrid search (vector + full-text)
- [ ] Faceted search (filter by date, priority, status)
- [ ] Search history and saved queries
- [ ] Related tasks recommendation

**Phase 8: Collaboration**
- [ ] Multi-user support (shared workspaces)
- [ ] Task assignment and delegation
- [ ] Comments and discussions
- [ ] Activity feed

**Phase 9: Integrations**
- [ ] Notion sync (two-way)
- [ ] Jira integration
- [ ] Slack notifications
- [ ] Email parsing (forward emails → tasks)

**Phase 10: Mobile & Web UI**
- [ ] Progressive Web App (PWA)
- [ ] React Native mobile app
- [ ] Desktop app (Electron)
- [ ] Browser extension

### 9.3. Technical Debt

**Performance**
- [ ] Connection pooling for HTTP clients
- [ ] Batch embedding generation
- [ ] Lazy loading for large task lists
- [ ] Query result caching

**Testing**
- [ ] Increase unit test coverage to 80%
- [ ] Add integration tests for all workflows
- [ ] E2E tests with Playwright
- [ ] Load testing with k6

**Documentation**
- [ ] API documentation (OpenAPI/Swagger)
- [ ] Architecture decision records (ADRs)
- [ ] Runbooks for common issues
- [ ] Video tutorials

**Security**
- [ ] Audit logging
- [ ] Encryption at rest
- [ ] RBAC (Role-Based Access Control)
- [ ] Security scanning (Snyk, Trivy)

---

## 10. Tổng kết

### 10.1. Điểm Mạnh

**Kiến trúc**
- Clean Architecture: Dễ maintain và extend
- Interface-based: Testable và flexible
- Event-driven: Scalable và loosely coupled

**AI Integration**
- Agentic AI: Intelligent multi-step reasoning
- Semantic search: Tìm kiếm theo ý nghĩa
- Tool calling: Extensible capabilities

**Developer Experience**
- One-command setup: `docker compose up`
- Live reload: Fast iteration
- Comprehensive docs: Easy onboarding

**Self-hosted**
- Zero cloud dependency: Full control
- Privacy: Data stays local
- Cost-effective: No API costs for storage

### 10.2. Trade-offs

**Complexity vs Flexibility**
- Pro: Highly extensible architecture
- Con: Steeper learning curve for new developers

**Self-hosted vs Managed**
- Pro: Full control, no vendor lock-in
- Con: Requires infrastructure management

**AI-first vs Traditional**
- Pro: Natural language interface
- Con: LLM costs and rate limits

**Markdown vs Structured**
- Pro: Human-readable, easy backup
- Con: Limited query capabilities

### 10.3. Lessons Learned

**Architecture**
- Start with interfaces, not implementations
- Dependency injection from day one
- Graceful degradation for non-critical features

**AI Integration**
- Always have fallbacks for LLM failures
- Cache expensive operations (embeddings)
- Set timeouts and max steps for agents

**Operations**
- Healthchecks are critical for Docker Compose
- Structured logging saves debugging time
- Backup automation is non-negotiable

**Development**
- Live reload dramatically improves DX
- Table-driven tests scale better
- Documentation is code

---

## Tài liệu Liên quan

- [README.md](../README.md) - Tổng quan hệ thống
- [Master Plan](version-1.0/master-plan.md) - Kế hoạch tổng thể (v1.0)
- [Configuration Guide](guidance/configuration-guide.md) - Hướng dẫn cấu hình
- [Phase Implementation Plans](version-1.0/phase-1-implementation-plan.md) - Chi tiết từng phase
- [Coding Conventions](convention/convention.md) - Quy ước code

---

**Document Version:** 1.0  
**Last Updated:** 2026-02-27  
**Author:** AI Assistant  
**Status:** Complete

