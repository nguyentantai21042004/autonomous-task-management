package telegram_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"autonomous-task-management/pkg/telegram"
)

func TestBot(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if strings.HasSuffix(path, "/setWebhook") {
			var req map[string]string
			json.NewDecoder(r.Body).Decode(&req)
			if req["url"] == "cause_error" {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"ok": false, "description": "invalid url"}`))
				return
			}
			if req["url"] == "cause_500" {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"ok": true, "description": "webhook set"}`))
			return
		}

		if strings.HasSuffix(path, "/sendMessage") {
			var req map[string]interface{}
			json.NewDecoder(r.Body).Decode(&req)
			text := req["text"].(string)

			if text == "cause_error" {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"ok": false, "description": "invalid text"}`))
				return
			}
			if text == "cause_500" {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"ok": true}`))
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	bot := telegram.NewBot("test-token")
	bot.SetAPIURL(ts.URL) // Route commands to test server instead of api.telegram.org

	t.Run("SetWebhook Success", func(t *testing.T) {
		err := bot.SetWebhook("https://example.com/webhook")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("SetWebhook API Failed", func(t *testing.T) {
		err := bot.SetWebhook("cause_error")
		if err == nil || !strings.Contains(err.Error(), "invalid url") {
			t.Fatalf("expected api failure error, got: %v", err)
		}
	})

	t.Run("SetWebhook HTTP Failed", func(t *testing.T) {
		err := bot.SetWebhook("cause_500")
		if err == nil {
			t.Fatalf("expected http decoding error")
		}
	})

	t.Run("SendMessage Success", func(t *testing.T) {
		err := bot.SendMessage(12345, "Hello")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("SendMessageWithMode Success", func(t *testing.T) {
		err := bot.SendMessageWithMode(12345, "Hello", "Markdown")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("SendMessage API Failed", func(t *testing.T) {
		err := bot.SendMessage(12345, "cause_error")
		if err == nil || !strings.Contains(err.Error(), "invalid text") {
			t.Fatalf("expected api failure error, got: %v", err)
		}
	})

	t.Run("SendMessage HTTP Failed", func(t *testing.T) {
		err := bot.SendMessage(12345, "cause_500")
		if err == nil {
			t.Fatalf("expected http decoding error")
		}
	})

	t.Run("Invalid API URL logic", func(t *testing.T) {
		badBot := telegram.NewBot("test")
		badBot.SetAPIURL("http://invalid-url.local:1234")
		err := badBot.SendMessage(12345, "fail")
		if err == nil {
			t.Errorf("expected network failure on invalid domain")
		}
	})
}
