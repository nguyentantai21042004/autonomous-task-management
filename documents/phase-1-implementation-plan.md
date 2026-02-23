## PHASE 1: CHUẨN HÓA HẠ TẦNG LOCAL - CODE PLAN

### Mục tiêu Giai đoạn 1

Xây dựng nền tảng hạ tầng local hoàn chỉnh, cho phép developer chỉ cần `docker compose up` là có ngay môi trường làm việc với Memos, Qdrant, và Golang backend. Giai đoạn này tập trung vào infrastructure-as-code, không có logic nghiệp vụ phức tạp.

---

### Cấu trúc Thư mục

```
autonomous-task-management/
├── docker-compose.yml          # Orchestration chính
├── .env.example                # Template cho biến môi trường
├── .env                        # File thực tế (git-ignored)
├── .gitignore
├── .dockerignore
├── Makefile                    # Build commands
├── README.md
├── go.mod
├── go.sum
├── documents/
│   ├── master-plan.md
│   ├── phase-1-implementation-plan.md
│   ├── google-calendar-setup.md    # Hướng dẫn OAuth
│   └── convention/             # Coding conventions (đã có)
├── manifests/                  # Kubernetes/Docker configs
│   ├── memos/
│   │   └── config.yaml         # Cấu hình Memos (nếu cần)
│   ├── qdrant/
│   │   └── config.yaml         # Cấu hình Qdrant
│   └── tags-schema.json        # Định nghĩa hệ thống Tags
├── cmd/
│   └── api/
│       ├── main.go             # Entry point
│       ├── Dockerfile
│       └── deployment.yaml     # K8s deployment (đã có)
├── config/                     # Application config (đã có)
│   ├── config.go
│   ├── config.example.yaml
│   └── config.yaml
├── internal/                   # Private application code (đã có)
│   ├── httpserver/
│   │   ├── httpserver.go
│   │   ├── health.go
│   │   └── new.go
│   ├── middleware/
│   │   ├── cors.go
│   │   ├── recovery.go
│   │   └── new.go
│   └── model/
│       ├── constant.go
│       └── scope.go
├── pkg/                        # Shared libraries (đã có)
│   ├── log/
│   ├── response/
│   └── errors/
└── scripts/
    ├── init-memos.sh           # Script khởi tạo Memos
    └── verify-setup.sh         # Script kiểm tra hệ thống
```

---

### Task Breakdown

#### Task 1.1: Thiết lập Docker Compose

**File:** `docker-compose.yml`

**Yêu cầu:**

- Service `memos`: Image official của Memos, expose port 5230, mount volume cho data persistence, **có healthcheck**
- Service `qdrant`: Image official của Qdrant, expose port 6333 (HTTP) và 6334 (gRPC), mount volume cho vector storage, **có healthcheck**
- Service `backend`: Build từ `cmd/api/Dockerfile`, expose port 8080, **depends_on với condition `service_healthy`**
- Network: Tạo bridge network để các service giao tiếp nội bộ

**Cấu hình chi tiết:**

```yaml
version: "3.8"

services:
  memos:
    image: neosmemo/memos:latest
    container_name: atm-memos
    ports:
      - "5230:5230"
    volumes:
      - memos-data:/var/opt/memos
    environment:
      - MEMOS_MODE=prod
      - MEMOS_PORT=5230
    networks:
      - atm-network
    healthcheck:
      test:
        [
          "CMD",
          "wget",
          "--no-verbose",
          "--tries=1",
          "--spider",
          "http://localhost:5230/healthz",
        ]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 30s
    restart: unless-stopped

  qdrant:
    image: qdrant/qdrant:latest
    container_name: atm-qdrant
    ports:
      - "6333:6333"
      - "6334:6334"
    volumes:
      - qdrant-data:/qdrant/storage
    networks:
      - atm-network
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:6333/health"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 20s
    restart: unless-stopped

  backend:
    build:
      context: .
      dockerfile: cmd/api/Dockerfile
    container_name: atm-backend
    ports:
      - "8080:8080"
    environment:
      - MEMOS_URL=http://memos:5230
      - MEMOS_ACCESS_TOKEN=${MEMOS_ACCESS_TOKEN}
      - QDRANT_URL=http://qdrant:6333
      - TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN}
      - GOOGLE_SERVICE_ACCOUNT_JSON=${GOOGLE_SERVICE_ACCOUNT_JSON}
    volumes:
      - ./config/config.yaml:/app/config/config.yaml:ro
    depends_on:
      memos:
        condition: service_healthy
      qdrant:
        condition: service_healthy
    networks:
      - atm-network
    restart: unless-stopped

networks:
  atm-network:
    driver: bridge

volumes:
  memos-data:
  qdrant-data:
```

**File:** `docker-compose.override.yml` (cho development với live-reload)

```yaml
version: "3.8"

services:
  backend:
    build:
      context: .
      dockerfile: cmd/api/Dockerfile.dev
    volumes:
      - .:/app
      - /app/vendor # Exclude vendor from mount
    environment:
      - AIR_ENABLED=true
    command: air -c .air.toml
```

---

#### Task 1.2: Cập nhật Dockerfile

**File:** `cmd/api/Dockerfile` (Production)

Cập nhật Dockerfile hiện tại để phù hợp với Phase 1:

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/api

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata curl wget

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/main .

# Copy config directory
COPY --from=builder /app/config ./config

EXPOSE 8080

CMD ["./main"]
```

**File:** `cmd/api/Dockerfile.dev` (Development với Air live-reload)

```dockerfile
FROM golang:1.21-alpine

WORKDIR /app

# Install Air for live reload
RUN go install github.com/cosmtrek/air@latest

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code (will be overridden by volume mount)
COPY . .

EXPOSE 8080

# Air will be started via docker-compose command
CMD ["air", "-c", ".air.toml"]
```

**File:** `.air.toml` (Air configuration)

```toml
root = "."
testdata_dir = "testdata"
tmp_dir = "tmp"

[build]
  args_bin = []
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main ./cmd/api"
  delay = 1000
  exclude_dir = ["assets", "tmp", "vendor", "testdata"]
  exclude_file = []
  exclude_regex = ["_test.go"]
  exclude_unchanged = false
  follow_symlink = false
  full_bin = ""
  include_dir = []
  include_ext = ["go", "tpl", "tmpl", "html", "yaml", "yml"]
  include_file = []
  kill_delay = "0s"
  log = "build-errors.log"
  poll = false
  poll_interval = 0
  rerun = false
  rerun_delay = 500
  send_interrupt = false
  stop_on_error = false

[color]
  app = ""
  build = "yellow"
  main = "magenta"
  runner = "green"
  watcher = "cyan"

[log]
  main_only = false
  time = false

[misc]
  clean_on_exit = false

[screen]
  clear_on_rebuild = false
  keep_scroll = true
```

WORKDIR /app

# Copy go mod files

COPY go.mod go.sum ./
RUN go mod download

# Copy source code

COPY . .

# Build binary

RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/api

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder

COPY --from=builder /app/main .

# Copy config directory

COPY --from=builder /app/config ./config

EXPOSE 8080

CMD ["./main"]

````

---

#### Task 1.3: Cập nhật File Environment Template

**File:** `.env.example`

```bash
# Telegram Bot Configuration
TELEGRAM_BOT_TOKEN=your_telegram_bot_token_here

# Memos Configuration
MEMOS_URL=http://localhost:5230
MEMOS_ACCESS_TOKEN=your_memos_access_token_here

# Qdrant Configuration
QDRANT_URL=http://localhost:6333

# Google Service Account (JSON string hoặc base64)
# Khuyên dùng Service Account thay vì OAuth Desktop App
GOOGLE_SERVICE_ACCOUNT_JSON={"type":"service_account","project_id":"..."}

# Optional: Custom ports (nếu muốn override)
# MEMOS_PORT=5230
# QDRANT_HTTP_PORT=6333
# BACKEND_PORT=8080
````

**Hướng dẫn lấy Memos Access Token:**

1. Truy cập http://localhost:5230
2. Đăng nhập/Tạo tài khoản admin
3. Vào Settings > Access Tokens
4. Click "Create Token"
5. Copy token và paste vào `.env`

---

#### Task 1.4: Định nghĩa Tags Schema

**File:** `manifests/tags-schema.json`

**Mục đích:** Chuẩn hóa hệ thống tags để LLM và Golang Agent hiểu được cấu trúc phân loại

```json
{
  "version": "1.0",
  "tagCategories": {
    "domain": {
      "description": "Phân loại theo lĩnh vực công việc",
      "examples": ["#domain/ahamove", "#domain/hcmut", "#domain/personal"]
    },
    "project": {
      "description": "Phân loại theo dự án cụ thể",
      "examples": ["#project/smap", "#project/thesis", "#project/side-hustle"]
    },
    "priority": {
      "description": "Mức độ ưu tiên",
      "values": ["#priority/p0", "#priority/p1", "#priority/p2", "#priority/p3"]
    },
    "status": {
      "description": "Trạng thái công việc",
      "values": [
        "#status/todo",
        "#status/in-progress",
        "#status/blocked",
        "#status/done"
      ]
    },
    "type": {
      "description": "Loại công việc",
      "examples": [
        "#type/coding",
        "#type/meeting",
        "#type/research",
        "#type/review"
      ]
    }
  },
  "rules": {
    "required": ["domain", "priority"],
    "optional": ["project", "status", "type"]
  }
}
```

---

#### Task 1.5: Cập nhật Config Application

**File:** `config/config.yaml`

Thêm cấu hình cho Memos, Qdrant, Telegram, Google Calendar:

```yaml
app:
  name: "Autonomous Task Management"
  version: "0.1.0"
  env: "development"
  port: 8080

log:
  level: "info"
  format: "json"

# Memos Configuration
memos:
  url: "http://localhost:5230"
  api_version: "v1"

# Qdrant Configuration
qdrant:
  url: "http://localhost:6333"
  collection_name: "task_embeddings"
  vector_size: 768

# Telegram Bot Configuration
telegram:
  bot_token: "" # Set via environment variable
  webhook_url: ""

# Google Calendar Configuration
google_calendar:
  credentials_path: "" # Set via environment variable
  calendar_id: "primary"
```

**File:** `config/config.go`

Cập nhật struct để load config mới:

```go
package config

import (
    "fmt"
    "os"

    "gopkg.in/yaml.v2"
)

type Config struct {
    App            AppConfig            `yaml:"app"`
    Log            LogConfig            `yaml:"log"`
    Memos          MemosConfig          `yaml:"memos"`
    Qdrant         QdrantConfig         `yaml:"qdrant"`
    Telegram       TelegramConfig       `yaml:"telegram"`
    GoogleCalendar GoogleCalendarConfig `yaml:"google_calendar"`
}

type AppConfig struct {
    Name    string `yaml:"name"`
    Version string `yaml:"version"`
    Env     string `yaml:"env"`
    Port    int    `yaml:"port"`
}

type LogConfig struct {
    Level  string `yaml:"level"`
    Format string `yaml:"format"`
}

type MemosConfig struct {
    URL        string `yaml:"url"`
    APIVersion string `yaml:"api_version"`
}

type QdrantConfig struct {
    URL            string `yaml:"url"`
    CollectionName string `yaml:"collection_name"`
    VectorSize     int    `yaml:"vector_size"`
}

type TelegramConfig struct {
    BotToken   string `yaml:"bot_token"`
    WebhookURL string `yaml:"webhook_url"`
}

type GoogleCalendarConfig struct {
    CredentialsPath string `yaml:"credentials_path"`
    CalendarID      string `yaml:"calendar_id"`
}

func Load(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read config file: %w", err)
    }

    var cfg Config
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, fmt.Errorf("failed to parse config: %w", err)
    }

    // Override with environment variables
    if token := os.Getenv("TELEGRAM_BOT_TOKEN"); token != "" {
        cfg.Telegram.BotToken = token
    }
    if creds := os.Getenv("GOOGLE_CALENDAR_CREDENTIALS"); creds != "" {
        cfg.GoogleCalendar.CredentialsPath = creds
    }
    if memosURL := os.Getenv("MEMOS_URL"); memosURL != "" {
        cfg.Memos.URL = memosURL
    }
    if qdrantURL := os.Getenv("QDRANT_URL"); qdrantURL != "" {
        cfg.Qdrant.URL = qdrantURL
    }

    return &cfg, nil
}
```

---

#### Task 1.6: Cập nhật Main Entry Point

**File:** `cmd/api/main.go`

Cập nhật để load config mới và khởi tạo HTTP server:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/yourusername/autonomous-task-management/config"
    "github.com/yourusername/autonomous-task-management/internal/httpserver"
    "github.com/yourusername/autonomous-task-management/internal/middleware"
    pkgLog "github.com/yourusername/autonomous-task-management/pkg/log"
)

func main() {
    // Load configuration
    cfg, err := config.Load("config/config.yaml")
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Initialize logger
    logger := pkgLog.New(cfg.Log.Level, cfg.Log.Format)
    ctx := context.Background()

    logger.Infof(ctx, "Starting %s v%s", cfg.App.Name, cfg.App.Version)
    logger.Infof(ctx, "Environment: %s", cfg.App.Env)
    logger.Infof(ctx, "Memos URL: %s", cfg.Memos.URL)
    logger.Infof(ctx, "Qdrant URL: %s", cfg.Qdrant.URL)

    // Initialize middleware
    mw := middleware.New(logger)

    // Initialize HTTP server
    server := httpserver.New(logger, mw)

    // Start HTTP server
    addr := fmt.Sprintf(":%d", cfg.App.Port)
    httpServer := &http.Server{
        Addr:    addr,
        Handler: server.Router(),
    }

    // Graceful shutdown
    go func() {
        logger.Infof(ctx, "HTTP server listening on %s", addr)
        if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logger.Fatalf(ctx, "HTTP server error: %v", err)
        }
    }()

    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    logger.Infof(ctx, "Shutting down server...")

    // Graceful shutdown with timeout
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := httpServer.Shutdown(shutdownCtx); err != nil {
        logger.Errorf(ctx, "Server forced to shutdown: %v", err)
    }

    logger.Infof(ctx, "Server exited")
}
```

---

#### Task 1.7: Scripts Tiện ích

**File:** `scripts/verify-setup.sh`

```bash
#!/bin/bash

echo "🔍 Verifying Autonomous Task Management Setup..."
echo ""

# Check Memos
echo -n "Checking Memos... "
if curl -s http://localhost:5230 > /dev/null 2>&1; then
    echo "✅ Running"
else
    echo "❌ Not accessible"
fi

# Check Qdrant
echo -n "Checking Qdrant... "
if curl -s http://localhost:6333/health > /dev/null 2>&1; then
    echo "✅ Running"
else
    echo "❌ Not accessible"
fi

# Check Backend
echo -n "Checking Backend... "
if curl -s http://localhost:8080/health > /dev/null 2>&1; then
    echo "✅ Running"
else
    echo "❌ Not accessible"
fi

echo ""
echo "🎉 Setup verification complete!"
```

**File:** `scripts/init-memos.sh`

```bash
#!/bin/bash

echo "📝 Memos Initial Setup Guide"
echo "============================"
echo ""
echo "Please follow these steps:"
echo ""
echo "1. Access Memos at http://localhost:5230"
echo "2. Create your admin account"
echo "3. Go to Settings > Tags"
echo "4. Create the following tag structure:"
echo ""
echo "   Domain Tags:"
echo "   - #domain/ahamove"
echo "   - #domain/hcmut"
echo "   - #domain/personal"
echo ""
echo "   Priority Tags:"
echo "   - #priority/p0 (Critical)"
echo "   - #priority/p1 (High)"
echo "   - #priority/p2 (Medium)"
echo "   - #priority/p3 (Low)"
echo ""
echo "   Status Tags:"
echo "   - #status/todo"
echo "   - #status/in-progress"
echo "   - #status/blocked"
echo "   - #status/done"
echo ""
echo "   Type Tags:"
echo "   - #type/coding"
echo "   - #type/meeting"
echo "   - #type/research"
echo "   - #type/review"
echo ""
echo "5. Save and you're ready to go!"
```

Thêm quyền execute:

```bash
chmod +x scripts/*.sh
```

---

#### Task 1.8: Tài liệu Google Calendar Setup

**File:** `documents/google-calendar-setup.md`

```markdown
## Hướng dẫn Cấu hình Google Calendar API

### Bước 1: Tạo Project trên Google Cloud Console

1. Truy cập https://console.cloud.google.com
2. Tạo project mới: "Autonomous Task Management"
3. Enable Google Calendar API:
   - Vào "APIs & Services" > "Library"
   - Tìm "Google Calendar API"
   - Click "Enable"

### Bước 2: Tạo OAuth 2.0 Credentials

1. Vào "APIs & Services" > "Credentials"
2. Click "Create Credentials" > "OAuth client ID"
3. Nếu chưa có OAuth consent screen:
   - Click "Configure Consent Screen"
   - Chọn "External" (hoặc "Internal" nếu có Google Workspace)
   - Điền thông tin cơ bản
   - Thêm scope: `https://www.googleapis.com/auth/calendar`
4. Quay lại "Create Credentials" > "OAuth client ID"
5. Application type: "Desktop app"
6. Name: "ATM Desktop Client"
7. Click "Create"
8. Download JSON file

### Bước 3: Cấu hình trong Project

1. Đổi tên file thành `google-credentials.json`
2. Copy vào thư mục project root hoặc nơi an toàn
3. Update `.env`:
```

GOOGLE_CALENDAR_CREDENTIALS=/path/to/google-credentials.json

````

### Bước 4: First-time Authorization

Lần đầu chạy backend, hệ thống sẽ:
1. Mở browser để authorize
2. Đăng nhập Google account
3. Cho phép ứng dụng truy cập Calendar
4. Token sẽ được lưu tự động (token.json)

### Bước 5: Verify

```bash
# Check if credentials file exists
ls -la google-credentials.json

# Start backend and check logs
docker compose logs -f backend
````

### Troubleshooting

**Error: "redirect_uri_mismatch"**

- Thêm `http://localhost` vào "Authorized redirect URIs" trong OAuth client settings

**Error: "invalid_grant"**

- Xóa file `token.json` và authorize lại

**Error: "access_denied"**

- Kiểm tra OAuth consent screen có đúng scope không
- Đảm bảo user account có quyền truy cập Calendar

```

---

#### Task 1.9: Cập nhật .gitignore

**File:** `.gitignore`

Thêm các dòng sau (nếu chưa có):

```

# Environment

.env

# Secrets

google-credentials.json
token.json
secrets/

# Docker volumes (nếu mount local)

memos-data/
qdrant-data/

# Go

_.exe
_.exe~
_.dll
_.so
_.dylib
_.test
\*.out
go.work

# IDE

.vscode/
.idea/
_.swp
_.swo
\*~

# OS

.DS_Store
Thumbs.db

````

---

#### Task 1.10: Cập nhật README.md

**File:** `README.md`

Thêm section Quick Start:

```markdown
# Autonomous Task Management

AI-powered task management system with Telegram interface, Memos storage, and Google Calendar integration.

## Architecture

- **Frontend**: Telegram Bot (voice + text)
- **Backend**: Golang orchestrator
- **Storage**: Memos (local, markdown-based)
- **Memory**: Qdrant (vector database)
- **Scheduler**: Google Calendar

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Telegram Bot Token (get from @BotFather)
- Google Calendar API credentials (see `documents/google-calendar-setup.md`)

### Setup

1. Clone repository:
   ```bash
   git clone <repo-url>
   cd autonomous-task-management
````

2. Configure environment:

   ```bash
   cp .env.example .env
   # Edit .env with your tokens
   nano .env
   ```

3. Start services:

   ```bash
   docker compose up -d
   ```

4. Verify setup:

   ```bash
   bash scripts/verify-setup.sh
   ```

5. Access services:
   - Memos: http://localhost:5230
   - Qdrant Dashboard: http://localhost:6333/dashboard
   - Backend API: http://localhost:8080
   - Health Check: http://localhost:8080/health

### Initial Configuration

1. Initialize Memos tags:

   ```bash
   bash scripts/init-memos.sh
   ```

2. Follow the instructions to set up your tag schema in Memos UI

3. Configure Google Calendar (first time only):
   - See `documents/google-calendar-setup.md`
   - Authorize when prompted

## Development

### Build locally

```bash
make build
```

### Run tests

```bash
make test
```

### View logs

```bash
docker compose logs -f backend
```

## Project Structure

See `documents/convention/` for detailed coding conventions.

## Roadmap

- [ ] Phase 1: Infrastructure Setup
- [ ] Phase 2: Core Engine (Telegram + LLM + Bulk Processing)
- [ ] Phase 3: RAG & Agent Tools
- [ ] Phase 4: Automation & Webhooks

---

### Checklist Hoàn thành Phase 1

- [ ] `docker-compose.yml` với healthchecks và service_healthy
- [ ] `docker-compose.override.yml` cho development
- [ ] `cmd/api/Dockerfile` updated (thêm curl/wget)
- [ ] `cmd/api/Dockerfile.dev` với Air
- [ ] `.air.toml` configuration
- [ ] `.env.example` với MEMOS_ACCESS_TOKEN và GOOGLE_SERVICE_ACCOUNT_JSON
- [ ] `manifests/tags-schema.json` định nghĩa hệ thống tags
- [ ] `config/config.yaml` với memos.access_token và google config
- [ ] `config/config.go` updated structs
- [ ] `cmd/api/main.go` updated với graceful shutdown
- [ ] Scripts `verify-setup.sh` và `init-memos.sh` với execute permission
- [ ] `documents/google-calendar-setup.md` updated với Service Account
- [ ] `.gitignore` updated
- [ ] `README.md` với hướng dẫn Quick Start
- [ ] Test: `docker compose up` chạy thành công
- [ ] Test: Backend chờ Memos/Qdrant healthy trước khi start
- [ ] Test: Memos API authentication với token
- [ ] Test: Live reload hoạt động (sửa code → auto restart)

---

### Deliverables

Sau khi hoàn thành Phase 1, developer sẽ có:

1. Môi trường local hoàn chỉnh chạy bằng 1 lệnh (`docker compose up`)
2. Backend Golang đã sẵn sàng để mở rộng (Phase 2)
3. Memos và Qdrant đã được cấu hình và persistent data
4. Tài liệu đầy đủ để onboard người mới
5. Foundation vững chắc để implement business logic
6. Cấu trúc code tuân thủ convention đã định nghĩa
7. **Live reload cho development (Air)**
8. **Healthcheck đảm bảo services sẵn sàng**
9. **Authentication với Memos API**
10. **Google Calendar headless với Service Account**

---

### Thời gian Ước tính

- Setup Docker Compose với healthchecks: 2-3 giờ
- Update Dockerfile (production + dev): 2-3 giờ
- Update config và environment: 2-3 giờ
- Setup Air live reload: 1-2 giờ
- Update main.go và wiring: 2-3 giờ
- Scripts và documentation: 2-3 giờ
- Testing và debugging: 2-3 giờ

**Tổng: 13-20 giờ** (2-3 ngày làm việc)

---

### Lưu ý Quan trọng

1. **Không xóa code hiện tại**: Giữ nguyên cấu trúc `internal/example`, `pkg/*` đã có. Chỉ thêm config và wiring mới.

2. **Convention**: Tuân thủ convention trong `documents/convention/` khi implement Phase 2.

3. **Testing**: Sau khi setup xong, test kỹ:
   - Health check endpoint
   - Kết nối đến Memos với Access Token
   - Kết nối đến Qdrant
   - Graceful shutdown
   - Live reload trong dev mode

4. **Security**:
   - Không commit `.env`, `google-service-account.json`, hoặc `token.json`
   - Sử dụng environment variables cho sensitive data
   - Review `.gitignore` trước khi commit
   - Rotate tokens định kỳ

5. **Documentation**:
   - Cập nhật README.md nếu có thay đổi
   - Document các API endpoints mới (nếu có)
   - Giữ convention docs updated

6. **Developer Experience**:
   - Dùng `docker compose up` cho dev mode (auto-reload)
   - Dùng `docker compose -f docker-compose.yml up` cho production mode
   - Check logs thường xuyên: `docker compose logs -f backend`

---

## 🚨 Critical Improvements Applied

### 1. Healthcheck cho Docker Services

**Vấn đề:** Backend start trước khi Memos/Qdrant sẵn sàng → crash loop

**Giải pháp:**

- Thêm `healthcheck` cho Memos (wget check `/healthz`)
- Thêm `healthcheck` cho Qdrant (curl check `/health`)
- Update `depends_on` với `condition: service_healthy`
- Thêm `curl` và `wget` vào Dockerfile

### 2. Memos Access Token

**Vấn đề:** Backend không có quyền gọi Memos API

**Giải pháp:**

- Thêm `MEMOS_ACCESS_TOKEN` vào `.env.example`
- Update `config.yaml` với field `access_token`
- Update `config.go` để load từ environment variable

**Cách lấy token:**

1. Truy cập http://localhost:5230
2. Đăng nhập/Tạo tài khoản admin
3. Vào Settings > Access Tokens
4. Click "Create Token"
5. Copy token và paste vào `.env`

### 3. Google Service Account (thay OAuth Desktop App)

**Vấn đề:** OAuth Desktop App không chạy được trong Docker container (headless environment)

**Giải pháp:**

- Đổi từ `GOOGLE_CALENDAR_CREDENTIALS` → `GOOGLE_SERVICE_ACCOUNT_JSON`
- Update config struct từ `GoogleCalendarConfig` → `GoogleConfig`
- Update `google-calendar-setup.md` với hướng dẫn Service Account

**Tại sao Service Account tốt hơn:**

- ✅ Chạy headless (không cần browser)
- ✅ Không cần user interaction
- ✅ Phù hợp cho backend service
- ✅ Dễ rotate credentials

### 4. Live Reload với Air (Development)

**Vấn đề:** Mỗi lần sửa code phải rebuild Docker image → chậm, giảm DX

**Giải pháp:**

- Tạo `cmd/api/Dockerfile.dev` với Air pre-installed
- Tạo `docker-compose.override.yml` cho dev mode
- Tạo `.air.toml` configuration
- Mount source code vào container

**Usage:**

```bash
# Development mode (auto-reload)
docker compose up

# Production mode (no override)
docker compose -f docker-compose.yml up
```

---

## 🎯 Verification Steps (Sau khi setup)

### 1. Verify Services Health

```bash
# Check all services
bash scripts/verify-setup.sh

# Check individual services
docker compose ps
docker compose logs memos
docker compose logs qdrant
docker compose logs backend
```

### 2. Test Memos API

```bash
# Get Memos info
curl -H "Authorization: Bearer $MEMOS_ACCESS_TOKEN" \
     http://localhost:5230/api/v1/user/me

# List memos
curl -H "Authorization: Bearer $MEMOS_ACCESS_TOKEN" \
     http://localhost:5230/api/v1/memos
```

### 3. Test Qdrant

```bash
# Check health
curl http://localhost:6333/health

# List collections
curl http://localhost:6333/collections
```

### 4. Test Backend

```bash
# Health check
curl http://localhost:8080/health

# Root endpoint
curl http://localhost:8080/
```

### 5. Test Live Reload (Dev Mode)

```bash
# Start in dev mode
docker compose up

# In another terminal, edit a file
echo "// test change" >> cmd/api/main.go

# Watch logs - should see rebuild and restart
docker compose logs -f backend
```

---

## 🔒 Security Checklist

- [ ] `.env` trong `.gitignore`
- [ ] `google-service-account.json` trong `.gitignore`
- [ ] `token.json` trong `.gitignore`
- [ ] Không commit sensitive data
- [ ] Review `.env.example` không chứa real credentials
- [ ] Memos Access Token được rotate định kỳ
- [ ] Google Service Account key được bảo mật

---

## 💡 Troubleshooting

### Backend crash loop

**Triệu chứng:** Backend restart liên tục

**Nguyên nhân:** Memos/Qdrant chưa sẵn sàng

**Giải pháp:** Kiểm tra healthcheck đã được apply đúng

```bash
docker compose config | grep -A 5 healthcheck
```

### Memos API 401 Unauthorized

**Triệu chứng:** Backend log "unauthorized" khi gọi Memos

**Nguyên nhân:** Thiếu hoặc sai Access Token

**Giải pháp:**

1. Kiểm tra `.env` có `MEMOS_ACCESS_TOKEN`
2. Verify token còn valid
3. Restart backend: `docker compose restart backend`

### Google Calendar authentication failed

**Triệu chứng:** Backend log "invalid credentials"

**Nguyên nhân:** Service Account JSON sai hoặc chưa share calendar

**Giải pháp:**

1. Verify JSON format đúng
2. Check Service Account email
3. Share Google Calendar với Service Account email
4. Verify permissions (Make changes to events)

### Live reload không hoạt động

**Triệu chứng:** Sửa code nhưng không thấy rebuild

**Nguyên nhân:** Không dùng `docker-compose.override.yml`

**Giải pháp:**

```bash
# Ensure override file exists
ls docker-compose.override.yml

# Restart with override
docker compose down
docker compose up
```

---

## 📚 References

- [Memos API Documentation](https://www.usememos.com/docs/api)
- [Qdrant Documentation](https://qdrant.tech/documentation/)
- [Docker Compose Healthcheck](https://docs.docker.com/compose/compose-file/compose-file-v3/#healthcheck)
- [Air - Live Reload for Go](https://github.com/cosmtrek/air)
- [Google Service Account](https://cloud.google.com/iam/docs/service-accounts)
