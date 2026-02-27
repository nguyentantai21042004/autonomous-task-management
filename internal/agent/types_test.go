package agent_test

import (
	"context"
	"testing"

	"autonomous-task-management/internal/agent"
)

type mockTool struct {
	name        string
	description string
	params      map[string]interface{}
}

func (m *mockTool) Name() string                       { return m.name }
func (m *mockTool) Description() string                { return m.description }
func (m *mockTool) Parameters() map[string]interface{} { return m.params }
func (m *mockTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	return nil, nil
}

func TestToolRegistry(t *testing.T) {
	registry := agent.NewToolRegistry()

	tool1 := &mockTool{name: "tool1", description: "desc1", params: nil}
	tool2 := &mockTool{name: "tool2", description: "desc2"}

	registry.Register(tool1)
	registry.Register(tool2)

	t.Run("Get existing tool", func(t *testing.T) {
		got, ok := registry.Get("tool1")
		if !ok || got.Name() != "tool1" {
			t.Errorf("expected tool1 to be found")
		}
	})

	t.Run("Get non-existing tool", func(t *testing.T) {
		_, ok := registry.Get("missing")
		if ok {
			t.Errorf("expected 'missing' tool to not be found")
		}
	})

	t.Run("List tools", func(t *testing.T) {
		tools := registry.List()
		if len(tools) != 2 {
			t.Errorf("expected 2 tools, got %d", len(tools))
		}
	})

	t.Run("ToFunctionDefinitions", func(t *testing.T) {
		defs := registry.ToFunctionDefinitions()
		if len(defs) != 2 {
			t.Fatalf("expected 2 tools, got %d", len(defs))
		}

		foundTool1 := false
		for _, tool := range defs {
			if tool.Name == "tool1" {
				foundTool1 = true
			}
		}

		if !foundTool1 {
			t.Errorf("expected tool1 to be in function definitions")
		}
	})
}
