# Autonomous Task Management

*Read this in [English](README.en.md).*
![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white) ![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=for-the-badge&logo=docker&logoColor=white) ![Telegram](https://img.shields.io/badge/Telegram-2CA5E0?style=for-the-badge&logo=telegram&logoColor=white)

> **"Một tin nhắn Telegram, AI lo hết"**  
> Hệ thống quản lý công việc tự trị kết hợp Agentic AI, tìm kiếm ngữ nghĩa và tự động hóa workflow qua Git webhooks.

---

## Hệ thống này giải quyết vấn đề gì?

Bạn có bao giờ:

- Phải mở 5-6 app khác nhau (Notion, Google Calendar, Slack, GitHub...) chỉ để ghi nhớ một task?
- Quên mất deadline vì task bị chôn vùi trong đống notes?
- Phải manually update trạng thái task sau khi merge PR?
- Tốn hàng giờ tìm kiếm lại context của một dự án cũ?

**Autonomous Task Management** giải quyết TẤT CẢ bằng một giao diện duy nhất: **Telegram**.

### Ví dụ thực tế

**Scenario 1: Tạo task nhanh như chớp**

```
Bạn: "Nhắc tôi lúc 9h sáng mai review PR số 123 của dự án SMAP"

Bot: Đã tạo task!
     Memo: memos.local/m/456
     Calendar: Đã đặt lịch 9:00 AM ngày mai
     Tags: #project/smap #pr/123
```

**Scenario 2: Tìm kiếm thông minh**

```
Bạn: /ask Tôi có deadline nào trong tuần này?

Bot: Để tôi kiểm tra...
     [Agent tự động gọi tool search_tasks và check_calendar]
     
     Bạn có 3 deadlines:
     1. Review PR #123 - Thứ 2, 9:00 AM
     2. Deploy staging - Thứ 4, 2:00 PM  
     3. Meeting với client - Thứ 6, 10:00 AM
```

**Scenario 3: Tự động hóa 100%**

```
[Bạn merge PR #123 trên GitHub]

Bot: Đã tự động đánh dấu hoàn thành:
     - [x] Review code
     - [x] Fix bugs
     - [x] Update docs
     
     Task "Review PR #123" đã hoàn thành!
```

---

## Tính năng nổi bật

### AI Agent tự trị (ReAct Framework)

- **Tự động suy luận đa bước**: Agent tự quyết định cần gọi tool nào (search, calendar, checklist)
- **Hiểu ngữ cảnh**: Không cần câu lệnh cứng nhắc, chat tự nhiên như với người
- **Xử lý bulk**: Paste cả một plan dài, AI tự tách thành từng task riêng biệt

### Tìm kiếm ngữ nghĩa (Semantic Search)

- **Vector Database (Qdrant)**: Tìm kiếm theo ý nghĩa, không cần khớp từ khóa chính xác
- **Multilingual**: Hỗ trợ tiếng Việt, tiếng Anh và nhiều ngôn ngữ khác
- **Tốc độ cao**: Kết quả trong <500ms

### Quản lý Checklist thông minh

- **Markdown-native**: Viết checklist như bình thường với `- [ ]` và `- [x]`
- **Partial matching**: `/check abc123 code` sẽ tìm tất cả checkbox có chữ "code"
- **Progress tracking**: Xem tiến độ real-time với `/progress <taskID>`

### Tự động hóa Git Workflow

- **GitHub/GitLab webhooks**: Tự động cập nhật khi PR merged, issue closed
- **Tag-based matching**: Dùng `#pr/123` để liên kết task với Pull Request
- **Zero manual work**: Merge code → Task tự động hoàn thành

### Tích hợp Google Calendar

- **Auto-scheduling**: Tạo task có thời gian → Tự động lên lịch
- **Conflict detection**: Agent kiểm tra lịch trống trước khi đặt
- **Deep links**: Click vào event → Mở ngay Memo với full context

---

## Kiến trúc kỹ thuật

![System Architecture](documents/architecture.png)

### Tech Stack

**Backend:**

- **Language**: Go 1.25.7 (Clean Architecture + DDD)
- **Framework**: Gin (HTTP), Air (Hot reload)
- **Deployment**: Docker Compose (100% containerized)

**AI & ML:**

- **LLM**: Google Gemini 2.0 Flash (Agent orchestration, NLU)
- **Embeddings**: Voyage AI voyage-3 (1024 dimensions, multilingual)
- **Vector DB**: Qdrant (Semantic search, RAG)

**Storage:**

- **Primary**: Memos (Self-hosted, Markdown-native)
- **Vector**: Qdrant (Embeddings storage)

**Integrations:**

- **Chat**: Telegram Bot API
- **Calendar**: Google Calendar API (OAuth2)
- **Git**: GitHub/GitLab Webhooks (HMAC-secured)

---

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Go 1.25+ (chỉ cần nếu dev)
- Ngrok account (để nhận webhooks)

### 1. Cấu hình API Keys

Bạn cần lấy các API keys sau (hướng dẫn chi tiết trong [Configuration Guide](documents/configuration-guide.md)):

- **Telegram Bot Token** - Giao diện chat
- **Gemini API Key** - AI brain
- **Voyage AI API Key** - Embeddings
- **Memos Access Token** - Storage
- **Google Calendar Credentials** - Scheduling (optional)
- **Ngrok Auth Token** - Webhook tunneling
- **Webhook Secret** - Git integration security (optional)

👉 **[Xem hướng dẫn lấy API keys chi tiết](documents/configuration-guide.md)**

### 2. Khởi động hệ thống

```bash
# Copy environment template
cp .env.example .env

# Tạo thư mục secrets
mkdir -p secrets

# Start all services
make up
```

Services sẽ chạy tại:

- **Backend API**: <http://localhost:8080>
- **Memos**: <http://localhost:5230>
- **Qdrant**: <http://localhost:6333>
- **Ngrok Dashboard**: <http://localhost:4040>

---

## Cách sử dụng

### Tạo task tự nhiên

Chỉ cần chat bình thường, AI sẽ tự hiểu:

```
"Deadline dự án SMAP vào 15/3"
"Gọi điện cho khách hàng XYZ lúc 10h sáng thứ 2"
"Review PR số 456 của repo backend"
```

### Tìm kiếm nhanh

```bash
/search meeting tomorrow
/search deadline march
/search bug login
```

### Agent thông minh

```bash
/ask Tôi có meeting nào tuần này?
/ask Deadline nào gần nhất?
/ask Tóm tắt công việc dự án SMAP

# Xóa lịch sử hội thoại (bắt đầu lại)
/reset
```

> **💡 Tip:** Agent có session memory - nhớ 5 turns hội thoại gần nhất. Bạn có thể hỏi follow-up mà không cần lặp lại context!

### Quản lý Checklist

```bash
# Xem tiến độ
/progress abc123

# Đánh dấu hoàn thành toàn bộ
/complete abc123

# Check một item cụ thể
/check abc123 Write tests

# Uncheck một item
/uncheck abc123 Review code
```

### Bulk create

Paste cả một plan dài:

```
Plan tuần này:
- Thứ 2: Review PR #123
- Thứ 3: Meeting với client lúc 10am
- Thứ 4: Deploy staging
- Thứ 5: Write documentation
- Thứ 6: Code review session
```

AI sẽ tự tách thành 5 tasks riêng biệt với đúng thời gian!

---

## Development

### Project Structure

```
.
├── cmd/api/              # Main application
├── internal/
│   ├── agent/           # AI Agent orchestrator
│   ├── automation/      # Webhook automation logic
│   ├── checklist/       # Markdown checklist parser
│   ├── task/            # Task management (usecase, repo, delivery)
│   ├── webhook/         # Git webhook handlers
│   └── httpserver/      # HTTP server & routing
├── pkg/                 # Shared packages
│   ├── gemini/         # Gemini LLM client
│   ├── voyage/         # Voyage AI embeddings
│   ├── qdrant/         # Qdrant vector DB client
│   ├── telegram/       # Telegram bot client
│   └── gcalendar/      # Google Calendar client
├── config/             # Configuration
├── documents/          # Documentation & guides
└── scripts/            # Utility scripts
```

### Makefile Commands

```bash
make up          # Start all services
make down        # Stop all services
make restart     # Restart backend only
make logs        # View backend logs
make test        # Run tests
make build       # Build binary
```

---

## Security

### Webhook Security

- **HMAC Signature Verification**: GitHub/GitLab webhooks được verify bằng HMAC-SHA256
- **Rate Limiting**: 60 requests/minute per source (configurable)
- **IP Whitelist**: Optional IP restriction
- **Constant-time Comparison**: Chống timing attacks

### API Keys

- Tất cả secrets được lưu trong thư mục `secrets` và file `.env` (không commit vào Git)
- Google Calendar dùng OAuth2 với refresh token
- Memos access token có thể set expiration

---

## Performance

- **Webhook acknowledgment**: <100ms
- **Background processing**: <2s
- **Checklist parsing**: <10ms
- **Semantic search**: <500ms
- **Memory usage**: ~150MB (all services)

---

## Troubleshooting

### Bot không phản hồi

```bash
# Check logs
make logs

# Verify webhook
curl http://localhost:4040/api/tunnels

# Test bot token
curl https://api.telegram.org/bot<YOUR_TOKEN>/getMe
```

### Qdrant không tìm thấy tasks

```bash
# Check collection
curl http://localhost:6333/collections/tasks

# Re-embed all tasks
go run scripts/backfill-embeddings/main.go
```

### Webhook không hoạt động

1. Check webhook secret khớp với GitHub/GitLab
2. Verify ngrok đang chạy: <http://localhost:4040>
3. Check logs: `make logs`

---

## Documentation

- [Configuration Guide](documents/configuration-guide.md) - Hướng dẫn lấy API keys
- [Master Plan](documents/master-plan.md) - Kiến trúc tổng thể
- [Phase 1-5 Plans](documents/) - Chi tiết implementation
- [Google Calendar Setup](documents/google-calendar-setup.md) - Setup OAuth2
- [Phase 5 Review](documents/phase-5-implementation-review-v2.md) - Test coverage & production readiness
- [Walkthrough](walkthrough.md) - Tổng quan implementation

---

## Roadmap

- [x] Phase 1: Infrastructure setup
- [x] Phase 2: Core task management + Telegram
- [x] Phase 3: RAG + Agent orchestrator
- [x] Phase 4: Automation + Git webhooks
- [x] **Phase 5: Verification & Testing** ✅ (95% complete - Production ready!)
  - ✅ Temporal context injection (Agent hiểu "tuần này", "ngày mai")
  - ✅ Conversational fallback (Chat tự nhiên không cần lệnh)
  - ✅ Session memory (Nhớ 5 turns hội thoại)
  - ✅ Test coverage 85% (vượt target 80%)
- [ ] Phase 6: Mobile app (React Native)
- [ ] Phase 7: Team collaboration features
- [ ] Phase 8: Analytics & insights

---

## Contributing

Contributions are welcome! Please read our contributing guidelines first.

---

## License

MIT License - feel free to use for personal or commercial projects.

---

## Acknowledgments

Built with:

- [Memos](https://github.com/usememos/memos) - Self-hosted note-taking
- [Qdrant](https://qdrant.tech/) - Vector database
- [Gemini](https://ai.google.dev/) - Google's LLM
- [Voyage AI](https://www.voyageai.com/) - Embeddings
- [Gin](https://gin-gonic.com/) - Go web framework

---

**Made with ❤️ by developers, for developers**
