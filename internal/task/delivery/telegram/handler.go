package telegram

import (
	"context"
	"fmt"

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

	// ---- Built-in commands ----
	switch msg.Text {
	case "/start":
		return h.bot.SendMessageWithMode(msg.Chat.ID,
			"👋 Chào mừng đến với *Autonomous Task Management*!\n\nGửi cho tôi danh sách công việc của bạn và tôi sẽ tự động:\n• 📝 Tạo tasks trong Memos\n• 📅 Đặt lịch trong Google Calendar\n\n_Ví dụ: \"Hoàn thành báo cáo SMAP ngày mai, review code cho Ahamove hôm nay\"_",
			"Markdown",
		)
	case "/help":
		return h.bot.SendMessageWithMode(msg.Chat.ID,
			"*Cách sử dụng:*\n\nNhập vào danh sách công việc tự nhiên, ví dụ:\n`Họp với team vào thứ Hai, viết unit test hôm nay ưu tiên p1, nghiên cứu Qdrant trong 2 ngày tới`\n\nBot sẽ phân tích và tạo tasks tương ứng.",
			"Markdown",
		)
	}

	// Build scope from Telegram user context
	sc := model.Scope{
		UserID:   fmt.Sprintf("telegram_%d", msg.From.ID),
		Username: msg.From.Username,
	}

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
