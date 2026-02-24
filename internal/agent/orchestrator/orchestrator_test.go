package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"autonomous-task-management/internal/agent"
	"autonomous-task-management/pkg/gemini"
)

type mockLogger struct{}

func (m *mockLogger) Debug(ctx context.Context, args ...any)                  {}
func (m *mockLogger) Debugf(ctx context.Context, format string, args ...any)  {}
func (m *mockLogger) Info(ctx context.Context, args ...any)                   {}
func (m *mockLogger) Infof(ctx context.Context, format string, args ...any)   {}
func (m *mockLogger) Warn(ctx context.Context, args ...any)                   {}
func (m *mockLogger) Warnf(ctx context.Context, format string, args ...any)   {}
func (m *mockLogger) Error(ctx context.Context, args ...any)                  {}
func (m *mockLogger) Errorf(ctx context.Context, format string, args ...any)  {}
func (m *mockLogger) DPanic(ctx context.Context, args ...any)                 {}
func (m *mockLogger) DPanicf(ctx context.Context, format string, args ...any) {}
func (m *mockLogger) Panic(ctx context.Context, args ...any)                  {}
func (m *mockLogger) Panicf(ctx context.Context, format string, args ...any)  {}
func (m *mockLogger) Fatal(ctx context.Context, args ...any)                  {}
func (m *mockLogger) Fatalf(ctx context.Context, format string, args ...any)  {}

type mockTool struct{}

func (m *mockTool) Name() string        { return "mock_tool" }
func (m *mockTool) Description() string { return "A mock tool" }
func (m *mockTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"foo": map[string]interface{}{"type": "string"},
		},
	}
}
func (m *mockTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	return "mock result", nil
}

func TestOrchestrator_ProcessQuery(t *testing.T) {
	registry := agent.NewToolRegistry()
	registry.Register(&mockTool{})

	// Create a mock LLM server that simulates returning text then returning a tool call
	callCount := 0
	llmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var resp gemini.GenerateResponse

		if callCount == 1 {
			// standard text reply
			resp = gemini.GenerateResponse{
				Candidates: []gemini.Candidate{
					{
						Content: gemini.Content{
							Parts: []gemini.Part{
								{Text: "Hello there!"},
							},
						},
					},
				},
			}
		} else if callCount == 2 {
			// tool call request
			resp = gemini.GenerateResponse{
				Candidates: []gemini.Candidate{
					{
						Content: gemini.Content{
							Parts: []gemini.Part{
								{
									FunctionCall: &gemini.FunctionCall{
										Name: "mock_tool",
										Args: map[string]interface{}{"foo": "bar"},
									},
								},
							},
						},
					},
				},
			}
		} else if callCount == 3 {
			// tool result ingestion -> final text
			resp = gemini.GenerateResponse{
				Candidates: []gemini.Candidate{
					{
						Content: gemini.Content{
							Parts: []gemini.Part{
								{Text: "Mock tool finished execution"},
							},
						},
					},
				},
			}
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer llmServer.Close()

	llm := gemini.NewClient("test-key")
	llm.SetAPIURL(llmServer.URL)

	l := &mockLogger{}
	o := New(llm, registry, l, "Asia/Ho_Chi_Minh")

	ctx := context.Background()

	t.Run("Standard Text Reply", func(t *testing.T) {
		reply, err := o.ProcessQuery(ctx, "user1", "Say hi")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if reply != "Hello there!" {
			t.Errorf("unexpected reply: %s", reply)
		}

		// check history length
		session := o.getSession("user1")
		if len(session.Messages) != 2 { // 1 user + 1 model
			t.Errorf("expected 2 messages in history, got %d", len(session.Messages))
		}
	})

	t.Run("Tool Calling Workflow", func(t *testing.T) {
		reply, err := o.ProcessQuery(ctx, "user2", "Use tool")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if !strings.Contains(reply, "Mock tool finished") {
			t.Errorf("unexpected reply: %s", reply)
		}

		// check history length
		session := o.getSession("user2")
		// Intermediate ReAct states are dropped. Only user query + final answer are saved.
		if len(session.Messages) != 2 {
			t.Errorf("expected 2 messages in history, got %d", len(session.Messages))
		}
	})

	t.Run("Session Cleanup", func(t *testing.T) {
		oldTTL := o.cacheTTL
		// Force tiny TTL
		o.cacheTTL = 10 * time.Millisecond

		s := o.getSession("user3")
		o.cacheMutex.Lock()
		s.Messages = append(s.Messages, gemini.Content{Role: "user", Parts: []gemini.Part{{Text: "dummy"}}})
		o.cacheMutex.Unlock()

		// Wait for expiry
		time.Sleep(20 * time.Millisecond)

		s2 := o.getSession("user3")

		if len(s2.Messages) != 0 {
			t.Errorf("expected user3 session to be evicted and reset, got %d messages", len(s2.Messages))
		}

		o.cacheTTL = oldTTL // restore
	})

	t.Run("LLM Failure Path", func(t *testing.T) {
		llmServer2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer llmServer2.Close()

		llm2 := gemini.NewClient("test-key")
		llm2.SetAPIURL(llmServer2.URL)

		o2 := New(llm2, registry, l, "")
		_, err := o2.ProcessQuery(ctx, "user", "hi")
		if err == nil {
			t.Errorf("expected LLM error but got nil")
		}
	})

	t.Run("Tool Not Found Workflow", func(t *testing.T) {
		missingToolServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := gemini.GenerateResponse{
				Candidates: []gemini.Candidate{
					{
						Content: gemini.Content{
							Parts: []gemini.Part{
								{
									FunctionCall: &gemini.FunctionCall{
										Name: "missing_tool",
									},
								},
							},
						},
					},
				},
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(resp)
		}))
		defer missingToolServer.Close()

		llmMiss := gemini.NewClient("test-key")
		llmMiss.SetAPIURL(missingToolServer.URL)

		oMiss := New(llmMiss, registry, l, "")
		// We expect the LLM to get an error map, then call LLM again... which loops.
		// To avoid infinite loop since the mock returns the same thing repeatedly,
		// we can limit Steps or rely on GenerateResponse dropping after the context is cancelled.
		// Let's just create a context with timeout to force it to return an error when HTTP fails.
		ctxTimeout, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		defer cancel()

		reply, err := oMiss.ProcessQuery(ctxTimeout, "user", "hi")
		if err != nil {
			t.Errorf("expected nil error for max steps graceful fallback, got %v", err)
		}
		if !strings.Contains(reply, "Trợ lý đã suy nghĩ quá lâu") {
			t.Errorf("expected max steps fallback message, got %q", reply)
		}
	})
}

func TestSessionMemory_BasicOps(t *testing.T) {
	l := &mockLogger{}
	o := New(nil, agent.NewToolRegistry(), l, "Asia/Ho_Chi_Minh")

	t.Run("NewUser", func(t *testing.T) {
		s := o.getSession("user_new")
		if s == nil {
			t.Fatal("expected session to not be nil")
		}
		if s.UserID != "user_new" {
			t.Errorf("expected UserID user_new, got %s", s.UserID)
		}
		if len(s.Messages) != 0 {
			t.Errorf("expected 0 messages, got %d", len(s.Messages))
		}
	})

	t.Run("ExistingUser", func(t *testing.T) {
		s1 := o.getSession("user_exist")
		s1.Messages = append(s1.Messages, gemini.Content{Role: "user", Parts: []gemini.Part{{Text: "hi"}}})

		s2 := o.getSession("user_exist")
		if len(s2.Messages) != 1 {
			t.Errorf("expected same session with 1 message, got %d", len(s2.Messages))
		}
	})

	t.Run("ExpiredSession", func(t *testing.T) {
		o.cacheTTL = 5 * time.Millisecond // very short TTL
		s1 := o.getSession("user_expire")
		s1.Messages = append(s1.Messages, gemini.Content{Role: "user", Parts: []gemini.Part{{Text: "hi"}}})

		time.Sleep(10 * time.Millisecond) // wait for expiration

		s2 := o.getSession("user_expire")
		if len(s2.Messages) != 0 {
			t.Errorf("expected new blank session after expiry, got %d messages", len(s2.Messages))
		}
		o.cacheTTL = 10 * time.Minute // restore
	})

	t.Run("ClearSession", func(t *testing.T) {
		s1 := o.getSession("user_clear")
		s1.Messages = append(s1.Messages, gemini.Content{Role: "user", Parts: []gemini.Part{{Text: "hi"}}})

		o.ClearSession("user_clear")

		s2 := o.getSession("user_clear")
		if len(s2.Messages) != 0 {
			t.Errorf("expected blank session after clear, got %d messages", len(s2.Messages))
		}
	})
}

func TestOrchestrator_ProcessQuery_HistoryLimit(t *testing.T) {
	registry := agent.NewToolRegistry()
	llmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := gemini.GenerateResponse{
			Candidates: []gemini.Candidate{
				{Content: gemini.Content{Parts: []gemini.Part{{Text: "Reply"}}}},
			},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer llmServer.Close()

	llm := gemini.NewClient("test-key")
	llm.SetAPIURL(llmServer.URL)

	l := &mockLogger{}
	o := New(llm, registry, l, "Asia/Ho_Chi_Minh")
	ctx := context.Background()

	t.Run("HistoryLimit", func(t *testing.T) {
		// Send 6 queries (12 messages total)
		for i := 0; i < 6; i++ {
			_, err := o.ProcessQuery(ctx, "user_limit", fmt.Sprintf("Query %d", i+1))
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
		}

		session := o.getSession("user_limit")
		if len(session.Messages) != 10 {
			t.Errorf("expected exactly 10 messages (5 user, 5 model) max, got %d", len(session.Messages))
		}
		// The first user query "Query 1" should be evicted
		if len(session.Messages) > 0 && strings.Contains(session.Messages[0].Parts[0].Text, "Query 1") {
			t.Errorf("expected first message to be evicted, but it's still there")
		}
	})
}

func TestOrchestrator_ProcessQuery_TemporalContext(t *testing.T) {
	registry := agent.NewToolRegistry()
	var capturedSystemPrompt string

	llmServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody gemini.GenerateRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err == nil {
			if reqBody.SystemInstruction != nil && len(reqBody.SystemInstruction.Parts) > 0 {
				capturedSystemPrompt = reqBody.SystemInstruction.Parts[0].Text
			}
		}

		resp := gemini.GenerateResponse{
			Candidates: []gemini.Candidate{
				{Content: gemini.Content{Parts: []gemini.Part{{Text: "OK"}}}},
			},
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer llmServer.Close()

	llm := gemini.NewClient("test-key")
	llm.SetAPIURL(llmServer.URL)
	l := &mockLogger{}

	t.Run("ValidTimezone", func(t *testing.T) {
		capturedSystemPrompt = ""
		o := New(llm, registry, l, "Asia/Ho_Chi_Minh")
		_, _ = o.ProcessQuery(context.Background(), "user_tz1", "hi")
		if !strings.Contains(capturedSystemPrompt, "Asia/Ho_Chi_Minh") {
			t.Errorf("expected system prompt to contain timezone string, got: %s", capturedSystemPrompt)
		}
		if !strings.Contains(capturedSystemPrompt, "Hôm nay là ") {
			t.Errorf("expected system prompt to contain current date, got: %s", capturedSystemPrompt)
		}
	})

	t.Run("InvalidTimezone_FallbackToUTC", func(t *testing.T) {
		capturedSystemPrompt = ""
		o := New(llm, registry, l, "Invalid/Timezone")
		_, _ = o.ProcessQuery(context.Background(), "user_tz2", "hi")
		if !strings.Contains(capturedSystemPrompt, "UTC") {
			t.Errorf("expected system prompt to fallback to UTC timezone string, got: %s", capturedSystemPrompt)
		}
	})
}
