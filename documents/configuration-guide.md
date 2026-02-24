# Hướng Dẫn Cấu Hình Chi Tiết

*Read this in [English](configuration-guide.en.md).*

> **Mục tiêu**: Lấy đầy đủ API keys và credentials để hệ thống hoạt động 100%

---

## Checklist tổng quan

Bạn cần chuẩn bị các thông tin sau (đánh dấu [x] khi hoàn thành):

- [ ] **Telegram Bot Token** - Bắt buộc (giao diện chat)
- [ ] **Gemini API Key** - Bắt buộc (AI brain)
- [ ] **Voyage AI API Key** - Bắt buộc (embeddings)
- [ ] **Memos Access Token** - Bắt buộc (storage)
- [ ] **Ngrok Auth Token** - Bắt buộc (webhooks)
- [ ] **Google Calendar Credentials** - Tùy chọn (scheduling)
- [ ] **Webhook Secret** - Tùy chọn (Git automation)

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

## 3. Gemini API Key

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

## Tổng kết

### File `.env` hoàn chỉnh

```bash
# ===== BẮT BUỘC =====
TELEGRAM_BOT_TOKEN=123456789:ABCdefGHIjklMNOpqrsTUVwxyz
MEMOS_ACCESS_TOKEN=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
GEMINI_API_KEY="AIzaSyA..."
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

### Gemini rate limit

```bash
# FREE tier: 15 req/min
# Giải pháp: Upgrade to paid tier hoặc đợi 1 phút
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

- [README](../README.md) - Tổng quan hệ thống
- [Master Plan](master-plan.md) - Kiến trúc chi tiết
- [Google Calendar Setup](google-calendar-setup.md) - OAuth2 deep dive
- [Troubleshooting Guide](../README.md#troubleshooting) - Common issues

---

**Chúc bạn setup thành công!**

*Có vấn đề? Mở issue trên GitHub hoặc check logs với `make logs`*
