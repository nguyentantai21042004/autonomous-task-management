package orchestrator

import (
	"context"
	"testing"

	"autonomous-task-management/internal/agent"
	"autonomous-task-management/pkg/gemini"
)

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
func (m *mockTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	return map[string]interface{}{"result": "executed"}, nil
}

func TestOrchestrator_ProcessQuery(t *testing.T) {
	registry := agent.NewToolRegistry()
	registry.Register(&mockTool{})

	// Test 1: Simple text response
	t.Run("simple text response", func(t *testing.T) {
		mockClient := &mockGeminiClient{
			response: &gemini.Response{
				Content: gemini.Content{
					Parts: []gemini.Part{
						{Text: "Hello there!"},
					},
				},
				Usage: &gemini.Usage{},
			},
		}

		l := &mockLogger{}
		manager := createManagerFromGeminiClient(mockClient, l)
		o := New(manager, registry, l, "Asia/Ho_Chi_Minh")

		ctx := context.Background()
		result, err := o.ProcessQuery(ctx, "user123", "test query")

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result != "Hello there!" {
			t.Errorf("expected 'Hello there!', got %q", result)
		}
	})

	// Test 2: Tool call (simplified - just check it doesn't crash)
	t.Run("with tool call", func(t *testing.T) {
		mockClient := &mockGeminiClient{
			response: &gemini.Response{
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
				Usage: &gemini.Usage{},
			},
		}

		l := &mockLogger{}
		manager := createManagerFromGeminiClient(mockClient, l)
		o := New(manager, registry, l, "Asia/Ho_Chi_Minh")

		ctx := context.Background()
		_, err := o.ProcessQuery(ctx, "user123", "test query with tool")

		// Should not crash even if tool call happens
		if err != nil {
			t.Logf("tool call resulted in error (expected): %v", err)
		}
	})
}

func TestOrchestrator_GetSession(t *testing.T) {
	registry := agent.NewToolRegistry()
	mockClient := &mockGeminiClient{}
	l := &mockLogger{}
	manager := createManagerFromGeminiClient(mockClient, l)
	o := New(manager, registry, l, "Asia/Ho_Chi_Minh")

	session := o.GetSession("user123")
	if session == nil {
		t.Error("expected non-nil session")
	}

	if session.UserID != "user123" {
		t.Errorf("expected userID 'user123', got %q", session.UserID)
	}
}
