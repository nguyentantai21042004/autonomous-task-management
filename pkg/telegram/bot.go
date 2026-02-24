package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

// SendMessage sends a plain text message to a Telegram chat.
func (b *Bot) SendMessage(chatID int64, text string) error {
	return b.SendMessageWithMode(chatID, text, "")
}

// SendMessageWithMode sends a message with optional parse mode (e.g. "Markdown").
func (b *Bot) SendMessageWithMode(chatID int64, text string, parseMode string) error {
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
