## Hướng dẫn Cấu hình Google Calendar API

### Bước 1: Tạo Project trên Google Cloud Console

1. Truy cập <https://console.cloud.google.com>
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
```

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
```

### Troubleshooting

**Error: "redirect_uri_mismatch"**

- Thêm `http://localhost` vào "Authorized redirect URIs" trong OAuth client settings

**Error: "invalid_grant"**

- Xóa file `token.json` và authorize lại

**Error: "access_denied"**

- Kiểm tra OAuth consent screen có đúng scope không
- Đảm bảo user account có quyền truy cập Calendar
