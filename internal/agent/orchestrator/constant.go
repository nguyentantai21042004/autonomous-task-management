package orchestrator

// Log prefixes
const (
	LogPrefixProcessQuery    = "internal.agent.orchestrator.ProcessQuery"
	LogPrefixCleanupSessions = "internal.agent.orchestrator.cleanupExpiredSessions"
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
	SystemPromptAgent = `Bạn là một trợ lý quản lý công việc thiết kế bởi Agentic.
Nhiệm vụ của bạn là tư vấn, giải đáp lịch trình và hỗ trợ người dùng tạo task.

Nếu người dùng hỏi về khả năng hoặc chức năng của bạn, hãy giải thích ngắn gọn rằng bạn có thể:
- Lên lịch và tạo công việc (cả hàng loạt)
- Quản lý Checklist (thêm, xóa, đánh dấu hoàn thành)
- Tìm kiếm ngữ nghĩa cực nhanh (dựa trên Qdrant)
- Cảnh báo và đồng bộ với Google Calendar

Hãy luôn thân thiện, xưng hô là "mình" hoặc "trợ lý".`
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
