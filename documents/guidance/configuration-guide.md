# Hướng Dẫn Cấu Hình Chi Tiết

*Read this in [English](configuration-guide.en.md).*

> **Mục tiêu**: Lấy đầy đủ API keys và credentials để hệ thống hoạt động 100%

---

## Checklist tổng quan

Bạn cần chuẩn bị các thông tin sau (đánh dấu [x] khi hoàn thành):

- [ ] **Telegram Bot Token** - Bắt buộc (giao diện chat)
- [ ] **LLM Provider API Key** - Bắt buộc (AI brain - DeepSeek primary, Gemini secondary, Qwen tertiary)
- [ ] **Voyage AI API Key** - Bắt buộc (embeddings)
- [ ] **Memos Access Token** - Bắt buộc (storage)
- [ ] **Ngrok Auth Token** - Bắt buộc (webhooks)
- [ ] **Google Calendar Credentials** - Tùy chọn (scheduling)
- [ ] **Webhook Secret** - Tùy chọn (Git automation)
- [ ] **LLM Provider Config** - Tùy chọn (multi-provider, fallback)

---

## 1. Telegram Bot Token

**Mục đích**: Giao diện chat duy nhất của hệ thống

**Thời gian**: ~2 phút

### Các bước thực hiện

1. **Mở Telegram** và tìm kiếm bot `@BotFather`

2. **Tạo bot mới**:

   ```
   Bạn: /newbot
   BotFather: Alright, a new bot. How are we going to call it?
   
   Bạn: My Task Manager Bot
   BotFather: Good. Now let's choose a username for your bot.
   
   Bạn: my_task_manager_bot
   BotFather: Done! Here's your token: 123456789:ABCdefGHIjklMNOpqrsTUVwxyz
   ```

3. **Copy token** và paste vào file `.env`:

   ```bash
   TELEGRAM_BOT_TOKEN=123456789:ABCdefGHIjklMNOpqrsTUVwxyz
   ```

### Lưu ý bảo mật

- **KHÔNG** commit token vào Git
- **KHÔNG** share token công khai
- Nếu lộ token, dùng `/revoke` với BotFather để tạo token mới

### Kiểm tra

```bash
# Test bot token
curl https://api.telegram.org/bot<YOUR_TOKEN>/getMe

# Response thành công:
# {"ok":true,"result":{"id":123456789,"is_bot":true,"first_name":"My Task Manager Bot",...}}
```

---

## 2. Memos Access Token

**Mục đích**: Đọc/ghi notes và tasks vào Memos (self-hosted)

**Thời gian**: ~3 phút

### Các bước thực hiện

1. **Khởi động hệ thống**:

   ```bash
   make up
   # Hoặc: docker compose up -d
   ```

2. **Mở Memos** tại <http://localhost:5230>

3. **Tạo tài khoản admin**:
   - Click "Sign up"
   - Username: `admin` (hoặc tùy ý)
   - Password: Chọn password mạnh
   - Click "Sign up"

4. **Tạo Access Token**:
   - Click vào **Avatar** (góc dưới bên trái)
   - Chọn **Settings**
   - Tab **Tokens**
   - Click **Create** (hoặc **New Token**)
   - Description: `Backend API`
   - Expiration: **Never** (không hết hạn)
   - Click **Create**

5. **Copy token** và paste vào `.env`:

   ```bash
   MEMOS_ACCESS_TOKEN=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
   ```

6. **Restart backend**:

   ```bash
   make restart
   ```

### Kiểm tra

```bash
# Test Memos API
curl -H "Authorization: Bearer <YOUR_TOKEN>" \
     http://localhost:5230/api/v1/memo

# Response thành công: {"memos":[...]}
```

---

## 3. LLM Provider API Key (Gemini)

**Mục đích**: "Bộ não AI" - Agent orchestration, NLU, reasoning

**Thời gian**: ~2 phút

**Chi phí**: FREE tier - 15 requests/minute, 1M tokens/day

### Các bước thực hiện

1. **Truy cập** [Google AI Studio](https://aistudio.google.com/app/apikey)

2. **Đăng nhập** bằng tài khoản Google

3. **Tạo API key**:
   - Click **Create API key**
   - Chọn project (hoặc tạo mới)
   - Click **Create API key in new project**
   - Copy key: `AIzaSyA...`

4. **Paste vào `.env`**:

   ```bash
   GEMINI_API_KEY="AIzaSyA..."
   ```

### Mẹo nhanh

- **Model sử dụng**: `gemini-2.0-flash-exp` (nhanh, rẻ, thông minh)
- **Rate limit**: 15 req/min (FREE), 1000 req/min (Paid)
- **Context window**: 1M tokens input, 8K tokens output

### Kiểm tra

```bash
# Test Gemini API
curl "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash-exp:generateContent?key=<YOUR_KEY>" \
  -H 'Content-Type: application/json' \
  -d '{"contents":[{"parts":[{"text":"Hello"}]}]}'

# Response thành công: {"candidates":[{"content":{"parts":[{"text":"Hello! ..."}]}}]}
```

---

## 4. Voyage AI API Key

**Mục đích**: Generate embeddings cho semantic search (RAG)

**Thời gian**: ~3 phút

**Chi phí**: FREE tier - 100M tokens/month

### Các bước thực hiện

1. **Đăng ký** tại [Voyage AI Dashboard](https://dash.voyageai.com/)

2. **Verify email** và đăng nhập

3. **Tạo API key**:
   - Sidebar -> **API Keys**
   - Click **Create new secret key**
   - Name: `Task Management`
   - Click **Create**
   - Copy key: `pa-...`

4. **Paste vào `.env`**:

   ```bash
   VOYAGE_API_KEY="pa-..."
   ```

### Model info

- **Model**: `voyage-3` (1024 dimensions)
- **Đặc điểm**: Multilingual, SOTA performance
- **Use case**: Semantic search, RAG, clustering

### Kiểm tra

```bash
# Test Voyage API
curl https://api.voyageai.com/v1/embeddings \
  -H "Authorization: Bearer <YOUR_KEY>" \
  -H "Content-Type: application/json" \
  -d '{"input":["Hello world"],"model":"voyage-3"}'

# Response thành công: {"object":"list","data":[{"object":"embedding","embedding":[0.123,...],"index":0}],...}
```

---

## 5. Ngrok Auth Token

**Mục đích**: Expose localhost ra internet để nhận webhooks (Telegram, GitHub, GitLab)

**Thời gian**: ~2 phút

**Chi phí**: FREE tier - 1 tunnel, 40 connections/minute

### Tại sao cần Ngrok?

Telegram và GitHub cần gửi webhooks đến server của bạn, nhưng:

- Localhost không thể truy cập từ internet
- Ngrok tạo một tunnel: `https://abc123.ngrok.io` -> `localhost:8080`

### Các bước thực hiện

1. **Đăng ký** tại [Ngrok](https://dashboard.ngrok.com/signup)

2. **Lấy auth token**:
   - Dashboard -> **Getting Started** -> **Your Authtoken**
   - Copy token: `2a...`

3. **Paste vào `.env`**:

   ```bash
   NGROK_AUTHTOKEN=2a...
   ```

4. **Restart services**:

   ```bash
   make down
   make up
   ```

### Kiểm tra

1. **Mở Ngrok dashboard**: <http://localhost:4040>

2. **Xem public URL**:

   ```bash
   curl http://localhost:4040/api/tunnels | jq '.tunnels[0].public_url'
   # Output: "https://abc123.ngrok.io"
   ```

3. **Test webhook**:

   ```bash
   curl https://abc123.ngrok.io/health
   # Response: {"status":"ok"}
   ```

### Mẹo nhanh

- Ngrok FREE: URL thay đổi mỗi lần restart
- Ngrok PAID: Static domain (recommended cho production)
- Alternative: Cloudflare Tunnel, Tailscale Funnel

---

## 6. Google Calendar Credentials (Optional)

**Mục đích**: Tự động tạo events, check lịch trống, conflict detection

**Thời gian**: ~10 phút

**Chi phí**: FREE

### Tại sao optional?

Hệ thống vẫn hoạt động 100% không có Calendar, nhưng:

- Có Calendar: Auto-schedule, conflict detection, reminders
- Không có: Tasks chỉ lưu trong Memos

### Bước A: Lấy `google-credentials.json`

1. **Truy cập** [Google Cloud Console](https://console.cloud.google.com/)

2. **Tạo project mới**:
   - Click **Select a project** -> **New Project**
   - Name: `Task Management`
   - Click **Create**

3. **Enable Google Calendar API**:
   - Sidebar -> **APIs & Services** -> **Library**
   - Tìm "Google Calendar API"
   - Click **Enable**

4. **Tạo OAuth credentials**:
   - Sidebar -> **APIs & Services** -> **Credentials**
   - Click **Create Credentials** -> **OAuth client ID**
   - Application type: **Desktop app**
   - Name: `Task Management Desktop`
   - Click **Create**

5. **Download JSON**:
   - Click **Download** (icon mũi tên xuống)
   - Save file as `secrets/google-credentials.json`

### Bước B: Generate `token.json`

1. **Chạy auth script**:

   ```bash
   go run scripts/gcal-auth/main.go secrets/google-credentials.json
   ```

2. **Follow hướng dẫn**:

   ```
   Go to the following link in your browser:
   https://accounts.google.com/o/oauth2/auth?...
   
   Enter authorization code:
   ```

3. **Authorize**:
   - Click vào link
   - Chọn tài khoản Google
   - Click **Allow**
   - Copy code từ URL: `?code=4/0A...`
   - Paste vào terminal

4. **Token được tạo**:

   ```
   Token saved to: secrets/token.json
   ```

5. **Update `.env`**:

   ```bash
   GOOGLE_CALENDAR_CREDENTIALS=secrets/google-credentials.json
   ```

### Kiểm tra

```bash
# Test calendar access
go run scripts/gcal-auth/main.go secrets/google-credentials.json

# Nếu thành công, sẽ list ra các events gần đây
```

### Refresh token

Token tự động refresh, không cần làm gì thêm. Nếu gặp lỗi:

```bash
# Xóa token cũ và auth lại
rm secrets/token.json
go run scripts/gcal-auth/main.go secrets/google-credentials.json
```

---

## 7. Webhook Secret (Optional)

**Mục đích**: Bảo mật webhooks từ GitHub/GitLab (HMAC verification)

**Thời gian**: ~1 phút

### Tại sao cần?

Webhooks là public endpoints, bất kỳ ai cũng có thể gửi fake requests. Webhook secret đảm bảo:

- Request đến từ GitHub/GitLab chính thức
- Payload không bị tamper
- Chống replay attacks

### Các bước thực hiện

1. **Generate secret mạnh**:

   ```bash
   # Option 1: OpenSSL
   openssl rand -hex 32
   
   # Option 2: Python
   python3 -c "import secrets; print(secrets.token_hex(32))"
   
   # Output: a1b2c3d4e5f6...
   ```

2. **Paste vào `.env`**:

   ```bash
   WEBHOOK_SECRET=a1b2c3d4e5f6...
   WEBHOOK_ENABLED=true
   ```

3. **Configure GitHub webhook**:
   - Repo -> **Settings** -> **Webhooks** -> **Add webhook**
   - Payload URL: `https://your-ngrok-url.ngrok.io/webhook/github`
   - Content type: `application/json`
   - Secret: `a1b2c3d4e5f6...` (same as .env)
   - Events: **Pull requests**, **Pushes**
   - Click **Add webhook**

4. **Configure GitLab webhook**:
   - Project -> **Settings** -> **Webhooks** -> **Add webhook**
   - URL: `https://your-ngrok-url.ngrok.io/webhook/gitlab`
   - Secret token: `a1b2c3d4e5f6...`
   - Trigger: **Push events**, **Merge request events**
   - Click **Add webhook**

### Kiểm tra

1. **Test GitHub webhook**:
   - Repo -> Settings -> Webhooks -> Click webhook
   - Tab **Recent Deliveries**
   - Click **Redeliver** -> Check response

2. **Check logs**:

   ```bash
   make logs
   # Nên thấy: "GitHub signature verification passed"
   ```

### Security best practices

- **Secret length**: Tối thiểu 32 bytes (64 hex chars)
- **Rotation**: Đổi secret mỗi 3-6 tháng
- **Storage**: Chỉ lưu trong `.env`, không commit vào Git
- **Rate limiting**: Đã enable sẵn (60 req/min)

---

## 8. LLM Provider Configuration (Optional - Advanced)

**Mục đích**: Cấu hình nhiều LLM providers với fallback tự động

**Thời gian**: ~5 phút

**Chi phí**: Tùy provider

### Tại sao cần?

Mặc định hệ thống dùng DeepSeek (nếu chỉ config `DEEPSEEK_API_KEY`). Nhưng bạn có thể:

- **Multi-provider**: Dùng DeepSeek làm PRIMARY (rẻ hơn, nhanh hơn), Gemini làm SECONDARY, Qwen làm TERTIARY
- **Fallback**: DeepSeek fail → tự động chuyển sang Gemini → nếu vẫn fail → chuyển sang Qwen
- **Cost optimization**: DeepSeek rẻ nhất cho simple tasks, Gemini backup khi cần, Qwen là fallback cuối
- **Redundancy**: Không bị downtime khi 1-2 providers lỗi

### Cấu hình trong `config/config.yaml`

```yaml
llm:
  fallback_enabled: true      # Bật fallback tự động
  retry_attempts: 3           # Số lần retry mỗi provider
  retry_delay: "1s"           # Delay giữa các retry
  
  providers:
    # Provider 1: DeepSeek (PRIMARY - ưu tiên cao nhất)
    - name: deepseek
      enabled: true
      priority: 1             # Thử TRƯỚC (primary)
      api_key: "${DEEPSEEK_API_KEY}"
      model: "deepseek-chat"
      timeout: "30s"
    
    # Provider 2: Gemini (SECONDARY)
    - name: gemini
      enabled: true
      priority: 2             # Thử SAU nếu DeepSeek fail (secondary)
      api_key: "${GEMINI_API_KEY}"
      model: "gemini-2.5-flash"
      timeout: "30s"
    
    # Provider 3: Qwen (TERTIARY)
    - name: qwen
      enabled: true
      priority: 3             # Thử CUỐI nếu cả DeepSeek và Gemini fail (tertiary)
      api_key: "${QWEN_API_KEY}"
      model: "qwen-turbo"
      timeout: "30s"
```

### Lấy Qwen API Key

1. **Đăng ký** tại [Alibaba Cloud DashScope](https://dashscope.console.aliyun.com/)

2. **Tạo API key**:
   - Dashboard -> **API Keys**
   - Click **Create API Key**
   - Copy key: `sk-...`

3. **Thêm vào `.env`**:

   ```bash
   QWEN_API_KEY="sk-..."
   ```

### Các mode cấu hình

#### Mode 1: Single provider (mặc định)

```yaml
# Không cần config gì, chỉ cần DEEPSEEK_API_KEY trong .env
# Hệ thống tự tạo config cho DeepSeek
```

#### Mode 2: Multi-provider với fallback

```yaml
llm:
  fallback_enabled: true
  providers:
    - name: deepseek
      priority: 1           # PRIMARY
      api_key: "${DEEPSEEK_API_KEY}"
      model: "deepseek-chat"
    - name: gemini
      priority: 2           # SECONDARY
      api_key: "${GEMINI_API_KEY}"
      model: "gemini-2.5-flash"
    - name: qwen
      priority: 3           # TERTIARY
      api_key: "${QWEN_API_KEY}"
      model: "qwen-turbo"
```

#### Mode 3: Disable một provider

```yaml
llm:
  providers:
    - name: deepseek
      enabled: false          # Tắt DeepSeek (không dùng PRIMARY)
      priority: 1
      api_key: "${DEEPSEEK_API_KEY}"
      model: "deepseek-chat"
    - name: gemini
      enabled: true           # Chỉ dùng Gemini
      priority: 2
      api_key: "${GEMINI_API_KEY}"
      model: "gemini-2.5-flash"
    - name: qwen
      enabled: true           # Và Qwen
      priority: 3
      api_key: "${QWEN_API_KEY}"
      model: "qwen-turbo"
```

### Priority và Fallback

- **Priority**: Số nhỏ = ưu tiên cao (1 > 2 > 3)
- **Fallback flow**:
  1. Thử provider priority 1 (DeepSeek - PRIMARY)
  2. Nếu fail → retry 3 lần với delay 1s
  3. Nếu vẫn fail → chuyển sang priority 2 (Gemini - SECONDARY)
  4. Nếu vẫn fail → chuyển sang priority 3 (Qwen - TERTIARY)
  5. Nếu tất cả fail → trả lỗi

### Kiểm tra

```bash
# Restart để load config mới
make restart

# Check logs
make logs

# Nên thấy:
# "LLM Provider Manager initialized" providers=3 fallback_enabled=true
# "Provider 1: deepseek (model: deepseek-chat)"
# "Provider 2: gemini (model: gemini-2.5-flash)"
# "Provider 3: qwen (model: qwen-turbo)"
```

### Test fallback

```bash
# Tắt DeepSeek (PRIMARY) bằng cách set sai API key
DEEPSEEK_API_KEY=invalid make restart

# Gửi message qua Telegram
# Logs sẽ show:
# "LLM generation failed" provider=deepseek error="API error 401"
# "LLM generation successful" provider=gemini fallback=true
```

### Best practices

- **Development**: Dùng single provider (đơn giản, chỉ cần DEEPSEEK_API_KEY)
- **Production**: Dùng multi-provider với fallback (reliable)
- **Cost optimization**: DeepSeek priority 1 (PRIMARY, rẻ nhất), Gemini priority 2 (SECONDARY), Qwen priority 3 (TERTIARY)
- **Monitoring**: Check logs để biết provider nào đang được dùng

---

## Tổng kết

### File `.env` hoàn chỉnh

```bash
# ===== BẮT BUỘC =====
TELEGRAM_BOT_TOKEN=123456789:ABCdefGHIjklMNOpqrsTUVwxyz
MEMOS_ACCESS_TOKEN=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
DEEPSEEK_API_KEY="sk-..."  # Primary LLM provider (khuyến nghị)
VOYAGE_API_KEY="pa-..."
NGROK_AUTHTOKEN=2a...

# ===== TÙY CHỌN =====
GOOGLE_CALENDAR_CREDENTIALS=secrets/google-credentials.json
WEBHOOK_ENABLED=true
WEBHOOK_SECRET=a1b2c3d4e5f6...

# ===== CẤU HÌNH NÂNG CAO =====
GEMINI_TIMEZONE="Asia/Ho_Chi_Minh"
WEBHOOK_RATE_LIMIT_PER_MIN=60
WEBHOOK_ALLOWED_IPS=  # Để trống = allow all

# ===== LLM MULTI-PROVIDER (OPTIONAL) =====
# Nếu muốn dùng fallback, thêm vào config.yaml
# Xem section 8 trong guide
GEMINI_API_KEY="AIzaSyA..."  # Optional: Google Gemini cho secondary fallback
QWEN_API_KEY="sk-..."  # Optional: Alibaba Qwen cho tertiary fallback
```

### Checklist cuối cùng

- [ ] Tất cả API keys đã paste vào `.env`
- [ ] `make up` chạy thành công
- [ ] Memos accessible tại <http://localhost:5230>
- [ ] Ngrok tunnel active tại <http://localhost:4040>
- [ ] Telegram bot phản hồi `/start`
- [ ] Test tạo task: "Họp team lúc 2pm ngày mai"

### Next steps

1. **Test basic flow**:

   ```
   Telegram: "Deadline dự án ABC vào 15/3"
   Bot: Đã tạo task!
   ```

2. **Test search**:

   ```
   Telegram: /search deadline
   Bot: Tìm thấy 1 task: Deadline dự án ABC...
   ```

3. **Test agent**:

   ```
   Telegram: /ask Tôi có deadline nào tuần này?
   Bot: Bạn có 1 deadline: Dự án ABC vào 15/3
   ```

4. **Setup webhooks** (nếu cần automation):
   - GitHub/GitLab webhooks
   - Test với PR merge

---

## Troubleshooting

### Bot không phản hồi

```bash
# Check logs
make logs

# Verify token
curl https://api.telegram.org/bot<TOKEN>/getMe

# Check webhook
curl http://localhost:4040/api/tunnels
```

### Memos 401 Unauthorized

```bash
# Token sai hoặc expired
# Tạo token mới trong Memos UI
# Update .env và restart: make restart
```

### LLM rate limit

```bash
# FREE tier: 15 req/min (Gemini), varies by provider
# Giải pháp: Upgrade to paid tier, dùng multi-provider fallback, hoặc đợi 1 phút
```

### Qdrant không tìm thấy tasks

```bash
# Re-embed all tasks
go run scripts/backfill-embeddings/main.go
```

### Ngrok tunnel không hoạt động

```bash
# Check auth token
docker compose logs ngrok

# Restart ngrok
docker compose restart ngrok
```

---

## Tài liệu liên quan

- [README](../../README.md) - Tổng quan hệ thống
- [Architecture Overview](../architecture-overview.md) - Kiến trúc chi tiết
- [Master Plan](../version-1.0/master-plan.md) - Kiến trúc tổng thể (v1.0)
- [Google Calendar Setup](../version-1.0/google-calendar-setup.md) - OAuth2 deep dive
- [Troubleshooting Guide](../../README.md#troubleshooting) - Common issues

---

**Chúc bạn setup thành công!**

*Có vấn đề? Mở issue trên GitHub hoặc check logs với `make logs`*
