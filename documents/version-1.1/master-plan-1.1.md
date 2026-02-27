# 🚀 MASTER PLAN: VERSION 1.1 - THE SMART GATEWAY & FIRST AID

## 1. HIỆN TRẠNG HỆ THỐNG (CURRENT STATUS)

Dựa trên quá trình vận hành thực tế và các log hệ thống (`real-chat.log`, `system.log`), hệ thống ATM hiện tại đang có kiến trúc Core Backend rất vững chắc, nhưng mắc phải các "điểm nghẽn" nghiêm trọng về UX và Logic giao tiếp:

- **Trải nghiệm người dùng (UX) bị phân mảnh:** Bắt buộc user phải nhớ và sử dụng đúng các Slash Commands (`/ask`, `/search`). Nếu chat tự nhiên (VD: "trong tuần này"), hệ thống tự ép vào luồng tạo Task và văng lỗi `no tasks parsed from input`.
- **Bệnh "Mù thời gian" (Temporal Blindness):** Agent không tự suy luận được thời gian thực tế để truyền vào Tool `check_calendar`, liên tục hỏi ngược lại user ngày bắt đầu/kết thúc gây ức chế.
- **Lỗi sập API ngầm (Markdown Parse Error):** Telegram API chặn đứng tin nhắn của Agent và văng lỗi `400 Bad Request` nếu LLM sinh ra các ký tự đặc biệt không được đóng/escape đúng chuẩn MarkdownV2.
- **Bóng ma dữ liệu (Data Drift trong RAG):** Vector trong Qdrant vẫn tồn tại trong khi Task gốc ở Memos đã bị xóa. Khi query, hệ thống quét trúng vector rác, gọi API Memos bị lỗi `404 Not Found`, dẫn đến báo cáo sai là "Không tìm thấy task".

---

## 2. CHI TIẾT ĐỀ XUẤT CẢI TIẾN (THE SOLUTIONS)

Để giải quyết triệt để các vấn đề trên, Version 1.1 sẽ bao gồm 5 mũi nhọn nâng cấp, tập trung vào tầng **Delivery (Telegram)** và **UseCase**.

### 2.1. Omni-Router (Cửa ngõ Phân loại Ý định)

- **Mô tả:** Đập bỏ các check `strings.HasPrefix` cứng nhắc. Mọi tin nhắn Telegram sẽ đi qua một bộ định tuyến ngữ nghĩa (Semantic Router) sử dụng **Gemini 2.5 Flash** (rất rẻ, độ trễ thấp).
- **Cách hoạt động:** LLM sẽ đọc tin nhắn và phân loại vào 1 trong 4 Intents: `CREATE_TASK`, `SEARCH_TASK`, `MANAGE_CHECKLIST`, `CONVERSATION`. Dựa vào Intent này, `handler.go` sẽ tự động gọi UseCase tương ứng. Bot có thể chat tự nhiên 100%.

### 2.2. Hard Time Injection (Ép bối cảnh thời gian)

- **Mô tả:** Chữa bệnh Agent hỏi lại ngày tháng. Thay vì truyền thời gian qua `SystemInstruction`, ta sẽ đính kèm trực tiếp thông tin thời gian thực vào phía sau tin nhắn của user trước khi đưa vào Orchestrator.
- **Cách hoạt động:** Bổ sung block `[System Note: Hôm nay là Thứ X, ngày YYYY-MM-DD. Hãy tự suy luận ngày tháng, tuyệt đối không hỏi lại]` vào đuôi câu query.

### 2.3. Markdown Sanitizer (Làm sạch Text Telegram)

- **Mô tả:** Chống lỗi `400 Bad Request` của Telegram.
- **Cách hoạt động:** Viết một hàm tiện ích `SanitizeMarkdownV2(text string) string` để tự động thêm dấu backslash `\` escape các ký tự nguy hiểm (`_`, `*`, `[`, `]`, `(`, `)`, `~`, `>`, `#`, `+`, `-`, `=`, `|`, `{`, `}`, `.`, `!`), TRỪ KHI chúng đang được dùng để format đúng chuẩn. Hoặc đơn giản hóa: Chuyển `ParseMode` của Telegram sang `HTML` để LLM ít sinh lỗi format hơn.

### 2.4. Self-Healing RAG (Bộ nhớ Tự chữa lành)

- **Mô tả:** Dọn dẹp rác VectorDB tự động.
- **Cách hoạt động:** Trong hàm `Search` của RAG UseCase, khi duyệt qua các kết quả của Qdrant: nếu gọi `GetTask` xuống Memos mà nhận về mã lỗi `404 Not Found`, lập tức trigger hàm ngầm `vectorRepo.DeleteTask(ctx, memoID)` để xóa vector rác đó đi.

### 2.5. Nối Session Memory vào Omni-Router

- **Mô tả:** Đảm bảo Router hiểu ngữ cảnh của câu chuyện đang diễn ra.
- **Cách hoạt động:** Tận dụng `map[string]*SessionMemory` từ Phase 5. Khi gọi Omni-Router, truyền 3 tin nhắn gần nhất vào prompt để Gemini Flash biết user đang tiếp nối câu chuyện (Ví dụ: Từ `CREATE_TASK` ở câu 1, sang câu 2 user bảo "Đổi lại lúc 9h nhé" -> Router vẫn hiểu là `CREATE_TASK`).

---

## 3. KẾ HOẠCH TRIỂN KHAI CODE (IMPLEMENTATION STEPS)

**Bước 1: Core Utilities (Sanitizer)**

- File: `pkg/telegram/bot.go`
- Action: Thêm hàm `EscapeMarkdownV2` hoặc update hàm `SendMessageWithMode` để fallback sang chế độ gửi text thô nếu gửi format bị lỗi.

**Bước 2: Self-Healing Logic**

- File: `internal/task/usecase/search.go` (Hoặc file xử lý RAG `answer_query.go`)
- Action: Bắt lỗi 404 từ Memos Repo và gọi `uc.vectorRepo.DeleteTask()`.

**Bước 3: Xây dựng Omni-Router**

- File mới: `internal/router/router.go`
- Action: Định nghĩa struct `SemanticRouter`, viết prompt cho Gemini Flash phân loại 4 Intents, trả về JSON.

**Bước 4: Nâng cấp Telegram Handler**

- File: `internal/task/delivery/telegram/handler.go`
- Action: Gỡ bỏ check `/ask`, `/search`. Inject `SemanticRouter` vào. Setup `switch-case` điều hướng dựa trên Intent trả về từ Router.

**Bước 5: Hard Time Injection & Session Memory**

- File: `internal/agent/orchestrator/orchestrator.go`
- Action: Load `time.Now()` theo timezone, ghép vào `rawQuery`. Nối lịch sử chat từ Cache vào request gửi lên Gemini.

---

## 4. MILESTONES & TIÊU CHÍ NGHIỆM THU (DEFINITION OF DONE)

Đây là các bài test khắc nghiệt để chứng minh Version 1.1 thành công rực rỡ:

### 🏆 Milestone 1: "Smooth Talker" (Giao tiếp không rào cản)

- **Hành động:** Gửi tin nhắn _"Chào bạn, bạn có thể giúp tôi những gì?"_ (không có `/ask`).
- **Kỳ vọng:** Hệ thống KHÔNG văng lỗi `no tasks parsed`. Trả về câu trả lời thân thiện mô tả các tính năng. Lịch sử log ghi nhận Intent là `CONVERSATION`.

### 🏆 Milestone 2: "Time Master" (Bậc thầy thời gian)

- **Hành động:** Gửi tin nhắn _"Kiểm tra lịch tuần này xem có vướng gì không?"_.
- **Kỳ vọng:** Agent KHÔNG HỎI LẠI ngày tháng. Tự động tính ra `start_date` (Thứ 2) và `end_date` (Chủ nhật), gọi Tool `check_calendar` và báo cáo kết quả.

### 🏆 Milestone 3: "Self-Healing RAG" (Không còn bóng ma)

- **Hành động:** 1. Tạo 1 task: _"Mua sữa lúc 5h chiều"_.

2. Vào web Memos xóa task đó đi.
3. Chat với Bot: _"Tìm task về việc mua sữa"_.

- **Kỳ vọng:** Bot báo _"Không tìm thấy task"_. (Nhưng khi kiểm tra backend log: Phải ghi nhận được log báo _Task deleted in Memos, triggering Qdrant self-healing cleanup..._).

### 🏆 Milestone 4: "Bulletproof Messaging" (Chống đạn API)

- **Hành động:** Ép bot sinh ra câu có ký tự đặc biệt: _"Tạo cho tôi một task: Code hàm func()\_test[]!"_
- **Kỳ vọng:** Telegram nhận được tin nhắn bình thường, các ký tự ngoặc và gạch dưới hiển thị đúng, hệ thống KHÔNG báo lỗi `400 Bad Request: can't parse entities`.
