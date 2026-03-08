package graph

import (
	"context"
	"errors"
	"testing"

	"autonomous-task-management/internal/agent"
	"autonomous-task-management/pkg/llmprovider"

	"github.com/stretchr/testify/assert"
)

// mockAgentTool implements agent.Tool for testing
type mockAgentTool struct {
	name    string
	result  interface{}
	execErr error
}

func (m *mockAgentTool) Name() string                    { return m.name }
func (m *mockAgentTool) Description() string             { return "mock tool" }
func (m *mockAgentTool) Parameters() map[string]interface{} { return map[string]interface{}{} }
func (m *mockAgentTool) Execute(_ context.Context, _ map[string]interface{}) (interface{}, error) {
	return m.result, m.execErr
}

func TestNodeExecuteTool_NoPendingTool(t *testing.T) {
	state := NewGraphState("user")
	state.PendingTool = nil
	registry := agent.NewToolRegistry()

	err := NodeExecuteTool(context.Background(), state, registry)

	assert.ErrorIs(t, err, ErrNoPendingTool)
	assert.Equal(t, StatusError, state.Status)
}

func TestNodeExecuteTool_ToolNotFound(t *testing.T) {
	state := NewGraphState("user")
	state.PendingTool = &llmprovider.FunctionCall{Name: "nonexistent_tool", Args: map[string]interface{}{}}
	registry := agent.NewToolRegistry()

	err := NodeExecuteTool(context.Background(), state, registry)

	assert.NoError(t, err) // loi tool khong phai loi fatal
	assert.Equal(t, StatusRunning, state.Status)
	assert.Nil(t, state.PendingTool)

	// Kiem tra error message duoc append vao messages
	lastMsg := state.Messages[len(state.Messages)-1]
	assert.Equal(t, "function", lastMsg.Role)
	resp := lastMsg.Parts[0].FunctionResponse.Response.(map[string]string)
	assert.Contains(t, resp["error"], "not found")
}

func TestNodeExecuteTool_ToolSuccess(t *testing.T) {
	state := NewGraphState("user")
	state.PendingTool = &llmprovider.FunctionCall{
		Name: "search_tasks",
		Args: map[string]interface{}{"query": "meeting"},
	}

	registry := agent.NewToolRegistry()
	registry.Register(&mockAgentTool{
		name:   "search_tasks",
		result: map[string]interface{}{"tasks": []string{"task1", "task2"}},
	})

	err := NodeExecuteTool(context.Background(), state, registry)

	assert.NoError(t, err)
	assert.Equal(t, StatusRunning, state.Status)
	assert.Nil(t, state.PendingTool)

	// Tool result duoc append
	assert.Len(t, state.Messages, 1)
	lastMsg := state.Messages[0]
	assert.Equal(t, "function", lastMsg.Role)
	assert.Equal(t, "search_tasks", lastMsg.Parts[0].FunctionResponse.Name)
}

func TestNodeExecuteTool_ToolExecutionError(t *testing.T) {
	state := NewGraphState("user")
	state.PendingTool = &llmprovider.FunctionCall{Name: "fail_tool", Args: map[string]interface{}{}}

	registry := agent.NewToolRegistry()
	registry.Register(&mockAgentTool{
		name:    "fail_tool",
		execErr: errors.New("database connection failed"),
	})

	err := NodeExecuteTool(context.Background(), state, registry)

	// Tool loi khong panic, tiep tuc running
	assert.NoError(t, err)
	assert.Equal(t, StatusRunning, state.Status)
	assert.Nil(t, state.PendingTool)

	lastMsg := state.Messages[0]
	resp := lastMsg.Parts[0].FunctionResponse.Response.(map[string]string)
	assert.Contains(t, resp["error"], "database connection failed")
}

func TestNodeExecuteTool_ResetsPendingTool(t *testing.T) {
	state := NewGraphState("user")
	state.PendingTool = &llmprovider.FunctionCall{Name: "my_tool", Args: map[string]interface{}{}}

	registry := agent.NewToolRegistry()
	registry.Register(&mockAgentTool{name: "my_tool", result: "ok"})

	NodeExecuteTool(context.Background(), state, registry)

	assert.Nil(t, state.PendingTool)
}
