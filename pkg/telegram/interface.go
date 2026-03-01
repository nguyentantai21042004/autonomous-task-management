package telegram

// IBot defines the interface for Telegram Bot operations.
// Implementations are safe for concurrent use.
type IBot interface {
	SetWebhook(webhookURL string) error
	SendMessage(chatID int64, text string) error
	SendMessageHTML(chatID int64, text string) error
	SendMessagePlain(chatID int64, text string) error
	SendMessageWithMode(chatID int64, text string, parseMode string) error
}

// New creates a new IBot instance.
func New(token string) IBot {
	return NewBot(token)
}
