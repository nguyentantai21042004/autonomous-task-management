package telegram

import (
	"context"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"autonomous-task-management/internal/model"
	"autonomous-task-management/internal/task"
	pkgLog "autonomous-task-management/pkg/log"
	pkgResponse "autonomous-task-management/pkg/response"
	pkgTelegram "autonomous-task-management/pkg/telegram"
)

type handler struct {
	l   pkgLog.Logger
	uc  task.UseCase
	bot *pkgTelegram.Bot
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
	if msg.Text == "" {
		return nil
	}

	// Handle built-in commands
	if msg.Text == "/start" {
		return h.bot.SendMessage(msg.Chat.ID, "Chào mừng đến với Autonomous Task Management!\n\n"+
			"Bạn có thể:\n"+
			"- Tạo task: gửi mô tả công việc (mặc định)\n"+
			"- Tìm task: dùng lệnh /search <query>\n"+
			"- Ví dụ: /search task SMAP đang block")
	}

	if msg.Text == "/help" {
		return h.bot.SendMessage(msg.Chat.ID, "📖 Hướng dẫn sử dụng:\n\n"+
			"**Tạo task:**\n"+
			"Finish SMAP report by tomorrow\n"+
			"Review code today p1\n"+
			"Tìm hiểu cách tích hợp VNPay (sẽ tạo task, KHÔNG search)\n\n"+
			"**Tìm task:**\n"+
			"/search task SMAP đang block\n"+
			"/search ahamove high priority\n"+
			"/search tasks due this week")
	}

	// Build scope from Telegram user
	sc := model.Scope{
		UserID: fmt.Sprintf("telegram_%d", msg.From.ID),
	}

	// CRITICAL FIX: Use explicit command instead of regex intent detection
	// Problem: "Tìm hiểu cách tích hợp VNPay" would be detected as search intent
	// Solution: Use /search command for explicit search, default to create task
	if strings.HasPrefix(msg.Text, "/search ") {
		return h.handleSearch(ctx, sc, msg)
	}

	// Default: create task (safer than regex intent detection)
	return h.handleCreateTask(ctx, sc, msg)
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

// handleSearch processes search requests.
func (h *handler) handleSearch(ctx context.Context, sc model.Scope, msg *pkgTelegram.Message) error {
	// Extract query (remove /search command)
	query := strings.TrimSpace(strings.TrimPrefix(msg.Text, "/search"))

	input := task.SearchInput{
		Query: query,
		Limit: 5, // Top 5 results
	}

	output, err := h.uc.Search(ctx, sc, input)
	if err != nil {
		h.l.Errorf(ctx, "Search failed: %v", err)
		return h.bot.SendMessage(msg.Chat.ID, "Có lỗi khi tìm kiếm. Vui lòng thử lại.")
	}

	if output.Count == 0 {
		return h.bot.SendMessage(msg.Chat.ID, "Không tìm thấy task nào phù hợp.")
	}

	// Format results
	response := fmt.Sprintf("🔍 Tìm thấy %d task:\n\n", output.Count)
	for i, result := range output.Results {
		// Extract title from content (first line)
		title := extractTitle(result.Content)
		score := int(result.Score * 100)

		response += fmt.Sprintf("%d. %s\n", i+1, title)
		response += fmt.Sprintf("   📊 Độ phù hợp: %d%%\n", score)
		response += fmt.Sprintf("   🔗 [Xem chi tiết](%s)\n\n", result.MemoURL)
	}

	return h.bot.SendMessageWithMode(msg.Chat.ID, response, "Markdown")
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
