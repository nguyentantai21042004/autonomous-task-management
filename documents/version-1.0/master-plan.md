## MASTER PROPOSAL: AUTONOMOUS TASK MANAGEMENT WORKSPACE

### 1. Kiến trúc Tổng thể (System Architecture)

Hệ thống chuyển dịch từ mô hình lưu trữ có cấu trúc (Relational) sang mô hình văn bản phẳng phân cấp (Tag-driven Flat Markdown), vận hành dưới sự điều phối của một AI Agent tự trị.

- **User Interface (Kênh Tương tác):** Telegram Bot nhận lệnh văn bản/giọng nói tự nhiên hoặc những khối văn bản lớn (bulk text).
- **Orchestrator Agent (Bộ não Điều phối):** Backend tự phát triển bằng Golang. Chịu trách nhiệm quản lý vòng đời request, xử lý logic ngày tháng (Date Math), parse Markdown/Regex, và điều tiết tốc độ gọi API (Rate Limiting/Batching).
- **Source of Truth - SoT (Nguồn Sự thật):** **Memos** (Chạy local theo Docker Compose). Lưu trữ toàn bộ payload công việc dưới dạng Markdown. Zero API Cost, Zero Rate Limit.
- **Semantic Memory (Bộ nhớ Ngữ nghĩa):** **Qdrant** (Vector Database, chạy bằng Docker compose). Lưu trữ embeddings của các thẻ Memos để LLM có khả năng truy xuất ngữ cảnh siêu tốc các đầu việc cũ, đang block hoặc liên quan.
- **Time Scheduler (Bộ định tuyến Thời gian):** **Google Calendar**. Đóng vai trò làm Timebox, push notification ép buộc chuyển ngữ cảnh, và là nơi chứa **Deep Links** trỏ ngược về Memos.

> **Lưu ý:** Toàn bộ hạ tầng được đóng gói, triển khai và run local thông qua cấu hình **docker-compose** ngay trong repo này (không yêu cầu chuẩn bị environment bên ngoài hoặc homelab phức tạp).

### 2. Danh sách Tính năng Cốt lõi (Core Features)

- **Bulk Semantic Parsing (Phân tách Ngữ nghĩa Hàng loạt):** Khả năng tiếp nhận một "nề" text lộn xộn (ví dụ: plan ôn thi 4 tuần, requirement đồ án dài), dùng LLM bóc tách thành một mảng (Array) JSON các sub-tasks chi tiết.
- **Tag-Driven Context (Ngữ cảnh qua Thẻ):** Loại bỏ các trường dữ liệu tĩnh. Quản lý phân luồng công việc bằng hệ sinh thái Tags cấp bậc (ví dụ: `#domain/ahamove`, `#domain/hcmut`, `#project/smap`, `#priority/p0`).
- **Zero-Friction Context Switching (Chuyển ngữ cảnh không rào cản):** Tự động đính kèm URL nội bộ của thẻ Memos (VD: `memos.local/m/123`) vào sự kiện Google Calendar. Tới giờ làm việc, chỉ 1 click là mở ra toàn bộ spec, link tài liệu, checklist.
- **Agentic Conflict Resolution (Giải quyết Xung đột Lịch trình):** Agent tự động gọi tool `check_calendar` để đối chiếu quỹ thời gian trống trước khi quyết định push một block thời gian mới.
- **Webhook-Driven Automation (Tự động hóa qua Webhook):** Khả năng bắt các event từ ngoại vi (ví dụ: merge code trên GitLab/GitHub) để Agent dùng Regex tự động đánh dấu `- [x]` vào checklist bên trong Memos và close task.

### 3. Luồng Dữ liệu Sự kiện (Event-Driven Data Flow)

Hệ thống hoạt động với 3 luồng độc lập, đảm bảo Golang Agent luôn làm chủ state và kiểm soát được tải.

- **Write Flow (Luồng Khởi tạo - Hỗ trợ Bulk):**

1. Lệnh từ Telegram (1 task hoặc plan dài) -> LLM parse ra mảng JSON (chỉ chứa ngày tương đối).
2. Golang Agent nhận mảng, dùng thư viện tính toán ra chuỗi `DateTime` tuyệt đối.
3. Golang Agent mở vòng lặp (For Loop): Gọi API local của Memos (chạy trong docker compose) tạo hàng loạt các cards (không lo rate limit) -> Thu thập mảng các `Memos_URL`.
4. Golang Agent gói các `Memos_URL` và thời gian vào một **Batch Request** bắn sang Google Calendar API để tạo nhiều sự kiện cùng lúc.

- **Read Flow (Luồng Truy vấn):**

1. Hỏi Telegram ("Tìm các task SMAP đang vướng") -> Agent tạo embedding -> Query Qdrant (local docker compose) lấy `Memos_ID`.
2. Gọi Memos API lấy nội dung Markdown mới nhất -> LLM đọc, suy luận và trả lời.

- **Sync Flow (Luồng Đồng bộ Trạng thái):**

1. Khi thao tác trực tiếp trên giao diện web Memos hoặc có Webhook từ Git trả về.
2. Memos bắn Webhook nội bộ sang Golang Agent -> Cập nhật lại vector trong Qdrant để đảm bảo "trí nhớ" của AI luôn chuẩn xác với thực tế.

### 4. Lộ trình Triển khai (Execution Roadmap)

- **Giai đoạn 1: Chuẩn hóa Hạ tầng Local (Local Setup)**
- Việc deploy toàn bộ hệ thống (bao gồm: Memos, Qdrant, Golang backend, các cấu phần phụ trợ) sẽ thực hiện bằng **docker compose** có sẵn trong repo. 
- Chỉ cần clone repo, sửa file `.env` nếu muốn và chạy `docker compose up` là đủ.
- Định nghĩa schema Tags và các config khác cũng để trong repo, không yêu cầu setup ngoài luồng.
- Đăng ký API Key/OAuth cho Google Calendar (được hướng dẫn riêng).

- **Giai đoạn 2: Xây dựng Pipeline & Bulk Processing (Core Engine)**
- Viết logic Golang kết nối Telegram với Gemini.
- Thiết kế System Prompt ép LLM trả về JSON Array.
- Code logic xử lý ngày tháng (Date Math) và vòng lặp gọi local API Memos + Batch API Calendar.

- **Giai đoạn 3: Nâng cấp Trí tuệ (RAG & Agent Tools)**
- Tích hợp Qdrant vào luồng. Viết trigger để cứ tạo/sửa Memo là tự động embed lại vào Qdrant.
- Khai báo Function Calling cho LLM (Tool check lịch, Tool tìm kiếm).

- **Giai đoạn 4: Tự động hóa Checklist (Regex & Webhooks)**
- Viết các bộ Regex parser trong Golang để nhận diện đoạn `- [ ]` trong chuỗi Markdown.
- Mở API endpoint trên Golang để hứng Webhook từ các hệ thống quản lý source code, tự động tick checklist.