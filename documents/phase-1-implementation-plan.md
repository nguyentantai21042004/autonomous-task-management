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

- Service `memos`: Image official của Memos, expose port 5230, mount volume cho data persistence
- Service `qdrant`: Image official của Qdrant, expose port 6333 (HTTP) và 6334 (gRPC), mount volume cho vector storage
- Service `backend`: Build từ `cmd/api/Dockerfile`, expose port 8080, depends_on memos và qdrant
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
      - QDRANT_URL=http://qdrant:6333
      - TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN}
      - GOOGLE_CALENDAR_CREDENTIALS=${GOOGLE_CALENDAR_CREDENTIALS}
    volumes:
      - ./config/config.yaml:/app/config/config.yaml:ro
    depends_on:
      - memos
      - qdrant
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

---

#### Task 1.2: Cập nhật Dockerfile

**File:** `cmd/api/Dockerfile`

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

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/main .

# Copy config directory
COPY --from=builder /app/config ./config

EXPOSE 8080

CMD ["./main"]
```

---

#### Task 1.3: Cập nhật File Environment Template

**File:** `.env.example`

```bash
# Telegram Bot Configuration
TELEGRAM_BOT_TOKEN=your_telegram_bot_token_here

# Google Calendar OAuth (JSON string hoặc file path)
GOOGLE_CALENDAR_CREDENTIALS=path/to/credentials.json

# Memos Configuration
MEMOS_URL=http://localhost:5230

# Qdrant Configuration
QDRANT_URL=http://localhost:6333

# Optional: Custom ports (nếu muốn override)
# MEMOS_PORT=5230
# QDRANT_HTTP_PORT=6333
# BACKEND_PORT=8080
```

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

- [x] Phase 1: Infrastructure Setup
- [ ] Phase 2: Core Engine (Telegram + LLM + Bulk Processing)
- [ ] Phase 3: RAG & Agent Tools
- [ ] Phase 4: Automation & Webhooks

```

---

### Checklist Hoàn thành Phase 1

- [ ] `docker-compose.yml` với 3 services (Memos, Qdrant, Backend)
- [ ] `cmd/api/Dockerfile` updated
- [ ] `.env.example` đầy đủ
- [ ] `manifests/tags-schema.json` định nghĩa hệ thống tags
- [ ] `config/config.yaml` với cấu hình Memos, Qdrant, Telegram, Google Calendar
- [ ] `config/config.go` updated để load config mới
- [ ] `cmd/api/main.go` updated với graceful shutdown
- [ ] Scripts `verify-setup.sh` và `init-memos.sh` với execute permission
- [ ] Tài liệu `google-calendar-setup.md`
- [ ] `.gitignore` updated
- [ ] `README.md` với hướng dẫn Quick Start
- [ ] Test: `docker compose up` chạy thành công
- [ ] Test: Truy cập được cả 3 services qua browser/curl
- [ ] Test: Backend health check trả về 200 OK

---

### Deliverables

Sau khi hoàn thành Phase 1, developer sẽ có:

1. Môi trường local hoàn chỉnh chạy bằng 1 lệnh (`docker compose up`)
2. Backend Golang đã sẵn sàng để mở rộng (Phase 2)
3. Memos và Qdrant đã được cấu hình và persistent data
4. Tài liệu đầy đủ để onboard người mới
5. Foundation vững chắc để implement business logic
6. Cấu trúc code tuân thủ convention đã định nghĩa

---

### Thời gian Ước tính

- Setup Docker Compose: 1-2 giờ
- Update Dockerfile và config: 2-3 giờ
- Update main.go và wiring: 2-3 giờ
- Scripts và documentation: 2-3 giờ
- Testing và debugging: 2-3 giờ

**Tổng: 9-14 giờ** (1-2 ngày làm việc)

---

### Lưu ý Quan trọng

1. **Không xóa code hiện tại**: Giữ nguyên cấu trúc `internal/example`, `pkg/*` đã có. Chỉ thêm config và wiring mới.

2. **Convention**: Tuân thủ convention trong `documents/convention/` khi implement Phase 2.

3. **Testing**: Sau khi setup xong, test kỹ:
   - Health check endpoint
   - Kết nối đến Memos
   - Kết nối đến Qdrant
   - Graceful shutdown

4. **Security**:
   - Không commit `.env` hoặc `google-credentials.json`
   - Sử dụng environment variables cho sensitive data
   - Review `.gitignore` trước khi commit

5. **Documentation**:
   - Cập nhật README.md nếu có thay đổi
   - Document các API endpoints mới (nếu có)
   - Giữ convention docs updated
```
