package telegram

import (
	"context"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"autonomous-task-management/internal/agent/orchestrator"
	"autonomous-task-management/internal/model"
	"autonomous-task-management/internal/task"
	pkgLog "autonomous-task-management/pkg/log"
	pkgResponse "autonomous-task-management/pkg/response"
	pkgTelegram "autonomous-task-management/pkg/telegram"
)

type handler struct {
	l            pkgLog.Logger
	uc           task.UseCase
	bot          *pkgTelegram.Bot
	orchestrator *orchestrator.Orchestrator
}

// HandleWebhook is the Gin handler for incoming Telegram webhook updates.
// It responds with HTTP 200 immediately and processes the message in a background goroutine
// to avoid Telegram webhook timeout (Telegram expects a response within a few seconds,
// but our pipeline: LLM + Memos + Calendar can take 5-10s).
func (h *handler) HandleWebhook(c *gin.Context) {
	ctx := c.Request.Context()

	var update pkgTelegram.Update
	if err := c.ShouldBindJSON(&update); err != nil {
		h.l.Errorf(ctx, "telegram handler: failed to parse update: %v", err)
		pkgResponse.Error(c, err, nil)
		return
	}

	// Ignore non-message updates (polls, channel_post, etc.)
	if update.Message == nil {
		pkgResponse.OK(c, map[string]string{"status": "ignored"})
		return
	}

	// Snapshot the message before spawning goroutine to avoid data races on gin context
	msg := update.Message

	// Critical: process in goroutine, return 200 immediately to Telegram
	go func() {
		// Detach from HTTP request context (which gets cancelled after response)
		bgCtx := context.Background()
		if err := h.processMessage(bgCtx, msg); err != nil {
			h.l.Errorf(bgCtx, "telegram handler: background processMessage failed: %v", err)
			// Best-effort error notification to user
			_ = h.bot.SendMessage(msg.Chat.ID, "Có lỗi xảy ra khi xử lý yêu cầu của bạn. Vui lòng thử lại.")
		}
	}()

	// Telegram acknowledged immediately
	pkgResponse.OK(c, map[string]string{"status": "accepted"})
}

// processMessage handles a single Telegram message.
func (h *handler) processMessage(ctx context.Context, msg *pkgTelegram.Message) error {
	sc := model.Scope{UserID: fmt.Sprintf("telegram_%d", msg.From.ID)}

	// Handle commands
	switch {
	case msg.Text == "/start":
		return h.handleStart(ctx, msg.Chat.ID)

	case msg.Text == "/help":
		return h.handleHelp(ctx, msg.Chat.ID)

	case strings.HasPrefix(msg.Text, "/search "):
		// Fast semantic search (Phase 3 Basic)
		query := strings.TrimSpace(strings.TrimPrefix(msg.Text, "/search"))
		return h.handleSearch(ctx, sc, query, msg.Chat.ID)

	case strings.HasPrefix(msg.Text, "/ask "):
		// Intelligent agent mode (Phase 3 Advanced)
		query := strings.TrimSpace(strings.TrimPrefix(msg.Text, "/ask"))
		return h.handleAgentOrchestrator(ctx, sc, query, msg.Chat.ID)

	default:
		// Default: Create task
		return h.handleCreateTask(ctx, sc, msg)
	}
}

// handleCreateTask processes requests to create tasks.
func (h *handler) handleCreateTask(ctx context.Context, sc model.Scope, msg *pkgTelegram.Message) error {
	// Notify user that processing has started
	if err := h.bot.SendMessage(msg.Chat.ID, "⏳ Đang xử lý..."); err != nil {
		h.l.Warnf(ctx, "telegram handler: failed to send ack message: %v", err)
	}

	input := task.CreateBulkInput{
		RawText:        msg.Text,
		TelegramChatID: msg.Chat.ID,
	}

	output, err := h.uc.CreateBulk(ctx, sc, input)
	if err != nil {
		h.l.Errorf(ctx, "telegram handler: CreateBulk failed: %v", err)
		return h.bot.SendMessage(msg.Chat.ID, fmt.Sprintf("Không thể xử lý yêu cầu: %v", err))
	}

	if output.TaskCount == 0 {
		return h.bot.SendMessage(msg.Chat.ID, "⚠️ Không tìm thấy tasks nào trong tin nhắn của bạn. Vui lòng thử lại với mô tả rõ ràng hơn.")
	}

	// Build success reply
	reply := fmt.Sprintf("Đã tạo *%d task(s)* thành công!\n\n", output.TaskCount)
	for i, t := range output.Tasks {
		reply += fmt.Sprintf("%d. *%s*", i+1, t.Title)
		if t.MemoURL != "" {
			reply += fmt.Sprintf("\n   📝 [Xem Memo](%s)", t.MemoURL)
		}
		if t.CalendarLink != "" {
			reply += fmt.Sprintf("\n   📅 [Xem Calendar](%s)", t.CalendarLink)
		}
		reply += "\n\n"
	}

	return h.bot.SendMessageWithMode(msg.Chat.ID, reply, "Markdown")
}

// handleSearch performs fast semantic search (existing functionality).
func (h *handler) handleSearch(ctx context.Context, sc model.Scope, query string, chatID int64) error {
	if query == "" {
		return h.bot.SendMessage(chatID, "❌ Vui lòng nhập từ khóa tìm kiếm.\n\nVí dụ: `/search meeting tomorrow`")
	}

	h.bot.SendMessage(chatID, "🔍 Đang tìm kiếm...")

	// Use existing search functionality
	searchInput := task.SearchInput{Query: query, Limit: 5}
	result, err := h.uc.Search(ctx, sc, searchInput)
	if err != nil {
		h.l.Errorf(ctx, "Search failed: %v", err)
		return h.bot.SendMessage(chatID, "❌ Lỗi tìm kiếm. Vui lòng thử lại.")
	}

	if len(result.Results) == 0 {
		return h.bot.SendMessage(chatID, "🤷‍♂️ Không tìm thấy task nào phù hợp.")
	}

	// Format results
	var response strings.Builder
	response.WriteString(fmt.Sprintf("🎯 Tìm thấy %d task:\n\n", len(result.Results)))

	for i, taskResult := range result.Results {
		title := extractTitle(taskResult.Content)
		response.WriteString(fmt.Sprintf("**%d. [%s](%s)**\n", i+1, title, taskResult.MemoURL))
		response.WriteString(fmt.Sprintf("🎯 %.0f%%\n", taskResult.Score*100))

		// Show preview (first 100 chars)
		preview := taskResult.Content
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}
		response.WriteString(fmt.Sprintf("💭 %s\n\n", preview))
	}

	return h.bot.SendMessageWithMode(chatID, response.String(), "Markdown")
}

// handleAgentOrchestrator uses intelligent agent with tools.
func (h *handler) handleAgentOrchestrator(ctx context.Context, sc model.Scope, query string, chatID int64) error {
	if query == "" {
		return h.bot.SendMessage(chatID, "❌ Vui lòng nhập câu hỏi.\n\nVí dụ: `/ask Tôi có meeting nào vào thứ 2 không?`")
	}

	h.bot.SendMessage(chatID, "🧠 Agent đang suy nghĩ...")

	// Call orchestrator (agent will decide which tools to use)
	answer, err := h.orchestrator.ProcessQuery(ctx, query)
	if err != nil {
		h.l.Errorf(ctx, "Agent failed: %v", err)
		return h.bot.SendMessage(chatID, "❌ Lỗi hệ thống Agent. Vui lòng thử lại.")
	}

	return h.bot.SendMessageWithMode(chatID, answer, "Markdown")
}

// handleStart shows welcome message with all modes.
func (h *handler) handleStart(ctx context.Context, chatID int64) error {
	message := `👋 **Chào mừng đến với Task Management Bot!**

🎯 **3 chế độ sử dụng:**

**1. Tạo Task (Mặc định)**
Gửi tin nhắn bình thường để tạo task mới.
*Ví dụ: "Meeting với team lúc 2pm ngày mai"*

**2. Tìm kiếm nhanh**
/search [từ khóa] - Tìm task theo từ khóa
*Ví dụ: /search meeting tomorrow*

**3. Trợ lý thông minh**
/ask [câu hỏi] - Agent tự động tìm kiếm và phân tích
*Ví dụ: /ask Tôi có meeting nào vào thứ 2 không?*

Gõ /help để xem hướng dẫn chi tiết.`

	return h.bot.SendMessageWithMode(chatID, message, "Markdown")
}

// handleHelp shows detailed usage instructions.
func (h *handler) handleHelp(ctx context.Context, chatID int64) error {
	message := `📖 **Hướng dẫn sử dụng**

**🆕 Tạo Task**
Gửi tin nhắn bình thường:
• "Họp team lúc 10am ngày mai"
• "Deadline dự án ABC vào 15/3"
• "Gọi điện cho khách hàng XYZ"

**🔍 Tìm kiếm nhanh**
/search [từ khóa]
• /search meeting - Tìm tất cả meeting
• /search deadline march - Tìm deadline tháng 3
• /search client call - Tìm cuộc gọi khách hàng

**🧠 Trợ lý thông minh**
/ask [câu hỏi]
• /ask Tôi có meeting nào tuần này?
• /ask Deadline nào gần nhất?
• /ask Tóm tắt công việc hôm nay

**💡 Mẹo:**
• Agent mode (/ask) thông minh hơn nhưng chậm hơn
• Search mode (/search) nhanh hơn cho truy vấn đơn giản
• Tạo task trực tiếp bằng tin nhắn thường`

	return h.bot.SendMessageWithMode(chatID, message, "Markdown")
}

// extractTitle extracts the first line from markdown content.
func extractTitle(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			// Remove markdown formatting
			line = strings.ReplaceAll(line, "**", "")
			line = strings.ReplaceAll(line, "*", "")
			if len(line) > 100 {
				return line[:100] + "..."
			}
			return line
		}
	}
	return "Untitled"
}
