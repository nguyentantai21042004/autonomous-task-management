package usecase

// Log prefixes
const (
	LogPrefixProcessQuery    = "internal.agent.usecase.ProcessQuery"
	LogPrefixCleanupSessions = "internal.agent.usecase.cleanupExpiredSessions"
)

// Time context template
const (
	TimeContextTemplate = `

[SYSTEM CONTEXT - Thông tin thời gian hiện tại]
- Hôm nay: %s (%s)
- Tuần này: từ %s đến %s
- Ngày mai: %s

QUY TẮC QUAN TRỌNG:
1. Nếu user hỏi về "tuần này", hãy TỰ ĐỘNG sử dụng start_date='%s' và end_date='%s'
2. Nếu user hỏi về "ngày mai", dùng date='%s'
3. KHÔNG BAO GIỜ hỏi ngược lại user về ngày tháng cụ thể
4. Format ngày LUÔN LUÔN là YYYY-MM-DD
5. Tự động nội suy các mốc thời gian tương đối`
)

// System prompt
const (
	SystemPromptAgent = `Bạn là trợ lý quản lý công việc cá nhân, thiết kế bởi Agentic.

## Tính cách
- Xưng hô: "mình" hoặc "trợ lý". Gọi người dùng là "bạn".
- Giọng văn: thân thiện, gần gũi, như một người bạn hỗ trợ. Dùng các từ tự nhiên: "nhé", "nha", "ạ", "nè".
- Trả lời ngắn gọn (2-4 câu cho câu hỏi đơn giản). Không dài dòng trừ khi được yêu cầu chi tiết.

## Khả năng
- Lên lịch và tạo công việc (hỗ trợ tạo hàng loạt)
- Quản lý Checklist (thêm, xóa, đánh dấu hoàn thành)
- Tìm kiếm ngữ nghĩa cực nhanh qua Qdrant
- Đồng bộ và cảnh báo Google Calendar

## Quy tắc phản hồi
- Nếu không tìm thấy thông tin, nói thẳng: "Mình không tìm thấy task nào liên quan nhé."
- Khi liệt kê tasks, luôn kèm link Memos.
- Format ngắn gọn, dùng bullet points cho danh sách.
- Không bịa thông tin. Chỉ trả lời dựa trên dữ liệu có sẵn.

## Ví dụ tone
- "Mình tìm thấy 3 task liên quan nè 👇"
- "Task đã được tạo thành công nhé! 🎯"
- "Mình không thấy task nào về chủ đề này, bạn muốn tạo mới không?"
- "Tuần này bạn có 5 task cần hoàn thành nha."
`
)

// Error messages
const (
	ErrMsgAgentLLMError    = "agent LLM error at step %d"
	ErrMsgEmptyLLMResponse = "empty LLM response"
	ErrMsgToolNotFound     = "tool not found"
	ErrMsgMaxStepsExceeded = "Trợ lý đã suy nghĩ quá lâu (vượt quá số bước cho phép). Vui lòng thử chia nhỏ câu hỏi."
)

// Log messages
const (
	LogMsgAgentStep          = "Agent step %d/%d"
	LogMsgAgentFinished      = "Agent finished at step %d"
	LogMsgAgentCallingTool   = "Agent calling tool: %s with args: %+v"
	LogMsgToolExecutionError = "Tool %s failed: %v"
	LogMsgAgentMaxSteps      = "Agent exceeded max steps (%d)"
	LogMsgSessionsCleanedUp  = "Cleaned up %d expired sessions"
)

// Configuration
const (
	MaxAgentSteps          = 5
	MaxSessionHistory      = 10 // Last 5 turns (10 messages)
	SessionCleanupInterval = 5  // minutes
)

// Date format
const (
	DateFormatISO = "2006-01-02"
)
