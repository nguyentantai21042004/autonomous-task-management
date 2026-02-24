package gemini_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"autonomous-task-management/pkg/gemini"
)

func TestBuildTaskParsingPrompt(t *testing.T) {
	nowStr := time.Now().Format(time.RFC3339)
	rawText := "Buy milk tomorrow"

	prompt := gemini.BuildTaskParsingPrompt(rawText, nowStr)

	if !strings.Contains(prompt, "You are a task parsing assistant") {
		t.Errorf("prompt missing system context")
	}
	if !strings.Contains(prompt, nowStr) {
		t.Errorf("prompt missing current time string")
	}
	if !strings.Contains(prompt, rawText) {
		t.Errorf("prompt missing source user text")
	}
}

func TestClient_GenerateContent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock LLM generation check
		if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if r.URL.Query().Get("key") != "test-api-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		var req gemini.GenerateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Read mock command
		text := req.Contents[0].Parts[0].Text
		if text == "cause_500" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"candidates": [
				{
					"content": {
						"parts": [
							{ "text": "mocked response string" }
						],
						"role": "model"
					}
				}
			]
		}`))
	}))
	defer ts.Close()

	client := gemini.NewClient("test-api-key")
	client.SetAPIURL(ts.URL)

	t.Run("Success Flow", func(t *testing.T) {
		req := gemini.GenerateRequest{
			Contents: []gemini.Content{
				{Parts: []gemini.Part{{Text: "Hello world"}}},
			},
		}

		resp, err := client.GenerateContent(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(resp.Candidates) != 1 {
			t.Fatalf("expected 1 candidate")
		}
		if resp.Candidates[0].Content.Parts[0].Text != "mocked response string" {
			t.Errorf("unexpected content response: %s", resp.Candidates[0].Content.Parts[0].Text)
		}
	})

	t.Run("Server Error Flow", func(t *testing.T) {
		req := gemini.GenerateRequest{
			Contents: []gemini.Content{
				{Parts: []gemini.Part{{Text: "cause_500"}}},
			},
		}

		_, err := client.GenerateContent(context.Background(), req)
		if err == nil {
			t.Fatalf("expected error from 500 response")
		}
	})

	t.Run("SetAPIURL test", func(t *testing.T) {
		c2 := gemini.NewClient("test-api-key")
		c2.SetAPIURL(ts.URL)

		req := gemini.GenerateRequest{
			Contents: []gemini.Content{
				{Parts: []gemini.Part{{Text: "Set URL Flow"}}},
			},
		}

		resp, err := c2.GenerateContent(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Candidates[0].Content.Parts[0].Text != "mocked response string" {
			t.Errorf("unexpected string")
		}
	})
}
