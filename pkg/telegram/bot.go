package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

// Bot is the Telegram Bot API client.
type Bot struct {
	token      string
	apiURL     string
	httpClient *http.Client
}

// NewBot creates a new Telegram Bot client with the given token.
func NewBot(token string) *Bot {
	return &Bot{
		token:      token,
		apiURL:     fmt.Sprintf("https://api.telegram.org/bot%s", token),
		httpClient: &http.Client{},
	}
}

// SetAPIURL overrides the default Telegram API URL for testing purposes.
func (b *Bot) SetAPIURL(url string) {
	b.apiURL = url
}

// SetWebhook registers the webhook URL with Telegram.
func (b *Bot) SetWebhook(webhookURL string) error {
	url := fmt.Sprintf("%s/setWebhook", b.apiURL)
	payload := map[string]string{"url": webhookURL}

	body, _ := json.Marshal(payload)
	resp, err := b.httpClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to set webhook: %w", err)
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("failed to decode webhook response: %w", err)
	}
	if !apiResp.OK {
		return fmt.Errorf("telegram setWebhook failed: %s", apiResp.Description)
	}
	return nil
}

// SendMessage sends a text message to the specified chat (uses HTML mode by default)
func (b *Bot) SendMessage(chatID int64, text string) error {
	return b.SendMessageHTML(chatID, text)
}

// SendMessageWithMode sends a message with optional parse mode (e.g. "Markdown").
func (b *Bot) SendMessageWithMode(chatID int64, text string, parseMode string) error {
	// HOTFIX 3: Sanitize markdown before sending to prevent Telegram API 400 errors
	if parseMode == "Markdown" || parseMode == "MarkdownV2" {
		text = removeInvalidMarkdown(text)
	}

	url := fmt.Sprintf("%s/sendMessage", b.apiURL)
	payload := SendMessageRequest{
		ChatID:    chatID,
		Text:      text,
		ParseMode: parseMode,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	resp, err := b.httpClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram sendMessage API error %d: %s", resp.StatusCode, string(raw))
	}

	return nil
}

// removeInvalidMarkdown removes unclosed markdown tags to prevent Telegram API errors
func removeInvalidMarkdown(text string) string {
	// Remove unclosed bold
	boldCount := strings.Count(text, "**")
	if boldCount%2 != 0 {
		text = strings.ReplaceAll(text, "**", "")
	}

	// Remove unclosed italic
	italicCount := strings.Count(text, "*")
	if italicCount%2 != 0 {
		text = strings.ReplaceAll(text, "*", "")
	}

	// Remove unclosed links [text](url)
	openBracket := strings.Count(text, "[")
	closeBracket := strings.Count(text, "]")
	openParen := strings.Count(text, "(")
	closeParen := strings.Count(text, ")")

	if openBracket != closeBracket || openParen != closeParen {
		// Remove all markdown links if unbalanced
		text = regexp.MustCompile(`\[([^\]]*)\]\(([^\)]*)\)`).ReplaceAllString(text, "$1")
	}

	return text
}

// SendMessageHTML sends message with HTML formatting (safer than MarkdownV2)
func (b *Bot) SendMessageHTML(chatID int64, text string) error {
	url := fmt.Sprintf("%s/sendMessage", b.apiURL)

	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "HTML",
	}

	body, _ := json.Marshal(payload)
	resp, err := b.httpClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Fallback to plain text if HTML parsing fails
		return b.SendMessagePlain(chatID, text)
	}

	return nil
}

// SendMessagePlain sends message without any formatting
func (b *Bot) SendMessagePlain(chatID int64, text string) error {
	url := fmt.Sprintf("%s/sendMessage", b.apiURL)

	payload := map[string]interface{}{
		"chat_id": chatID,
		"text":    text,
		// No parse_mode = plain text
	}

	body, _ := json.Marshal(payload)
	resp, err := b.httpClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram sendMessage API error %d: %s", resp.StatusCode, string(raw))
	}

	return nil
}
