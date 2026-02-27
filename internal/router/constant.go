package router

// Log prefixes
const (
	LogPrefixClassify = "internal.router.Classify"
)

// Router prompts
const (
	PromptRouterSystem = `Bạn là Semantic Router. Phân tích tin nhắn sau và xác định ý định (intent) của người dùng.

Tin nhắn hiện tại: "%s"

Các intent có thể:
1. CREATE_TASK: Tạo task mới, thêm công việc, nhắc nhở, deadline
2. SEARCH_TASK: Tìm kiếm, tra cứu, xem task cũ
3. MANAGE_CHECKLIST: Đánh dấu hoàn thành, check/uncheck, xem tiến độ
4. CONVERSATION: Chào hỏi, hỏi về tính năng, chat thông thường

Trả về JSON với format:
{
  "intent": "CREATE_TASK|SEARCH_TASK|MANAGE_CHECKLIST|CONVERSATION",
  "confidence": 0-100,
  "reasoning": "Giải thích ngắn gọn"
}`

	PromptHistoryPrefix = "Lịch sử hội thoại gần đây:\n"
)

// Router configuration
const (
	RouterTemperature        = 0.1
	RouterFallbackIntent     = IntentConversation
	RouterFallbackConfidence = 50
)

// Error messages
const (
	ErrMsgLLMCallFailed   = "LLM call failed"
	ErrMsgJSONParseFailed = "Failed to parse JSON, falling back to CONVERSATION"
	ErrMsgEmptyResponse   = "Empty LLM response, falling back to CONVERSATION"
)

// Fallback reasons
const (
	ReasonParsingError  = "Fallback due to parsing error - route to conversational agent"
	ReasonEmptyResponse = "Fallback due to empty response"
)
