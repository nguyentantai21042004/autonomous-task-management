package graph

import (
	"context"

	"autonomous-task-management/internal/agent"
	"autonomous-task-management/pkg/llmprovider"
)

// NodeExecuteTool doc PendingTool tu state, thuc thi qua ToolRegistry,
// va luu ket qua vao Messages.
// Day la buoc "Act + Observe" trong ReAct.
//
// Sau khi chay:
//   - Thanh cong → PendingTool=nil, Status=RUNNING (de NodeAgent reason tiep)
//   - Tool khong ton tai → append error result, Status=RUNNING
//   - Tool thuc thi loi → append error string, Status=RUNNING
//   - PendingTool nil → Status=ERROR, ErrNoPendingTool
func NodeExecuteTool(
	ctx context.Context,
	state *GraphState,
	registry *agent.ToolRegistry,
) error {
	if state.PendingTool == nil {
		state.Status = StatusError
		return ErrNoPendingTool
	}

	toolName := state.PendingTool.Name
	tool, ok := registry.Get(toolName)

	var toolResult interface{}
	if !ok {
		toolResult = map[string]string{"error": "tool not found: " + toolName}
	} else {
		result, err := tool.Execute(ctx, state.PendingTool.Args)
		if err != nil {
			toolResult = map[string]string{"error": err.Error()}
		} else {
			toolResult = result
		}
	}

	// Append tool result vao Messages
	state.AppendMessage(llmprovider.Message{
		Role: "function",
		Parts: []llmprovider.Part{{
			FunctionResponse: &llmprovider.FunctionResponse{
				Name:     toolName,
				Response: toolResult,
			},
		}},
	})

	// Reset pending tool, tiep tuc reasoning
	state.PendingTool = nil
	state.Status = StatusRunning
	return nil
}
