package orchestrator

import (
	"context"
	"fmt"

	"autonomous-task-management/pkg/gemini"
)

const MaxAgentSteps = 5

// ProcessQuery runs ReAct loop: Reason → Act → Observe
func (o *Orchestrator) ProcessQuery(ctx context.Context, query string) (string, error) {
	req := gemini.GenerateRequest{
		Contents: []gemini.Content{
			{Role: "user", Parts: []gemini.Part{{Text: query}}},
		},
		Tools: o.registry.ToFunctionDefinitions(),
	}

	for step := 0; step < MaxAgentSteps; step++ {
		o.l.Infof(ctx, "Agent step %d/%d", step+1, MaxAgentSteps)

		// 1. Reason: Ask LLM what to do
		resp, err := o.llm.GenerateContent(ctx, req)
		if err != nil {
			return "", fmt.Errorf("agent LLM error at step %d: %w", step, err)
		}

		if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
			return "", fmt.Errorf("empty LLM response")
		}

		part := resp.Candidates[0].Content.Parts[0]

		// 2. Check if LLM wants to call a tool
		if part.FunctionCall == nil {
			// LLM has final answer
			o.l.Infof(ctx, "Agent finished at step %d", step+1)
			return part.Text, nil
		}

		// 3. Act: Execute the tool
		toolName := part.FunctionCall.Name
		o.l.Infof(ctx, "Agent calling tool: %s with args: %+v", toolName, part.FunctionCall.Args)

		tool, ok := o.registry.Get(toolName)
		var toolResult interface{}

		if !ok {
			o.l.Errorf(ctx, "Tool %s not found", toolName)
			toolResult = map[string]string{"error": "tool not found"}
		} else {
			// Execute tool
			res, err := tool.Execute(ctx, part.FunctionCall.Args)
			if err != nil {
				o.l.Errorf(ctx, "Tool %s failed: %v", toolName, err)
				toolResult = map[string]string{"error": err.Error()}
			} else {
				toolResult = res
			}
		}

		// 4. Observe: Add tool result to conversation history
		req.Contents = append(req.Contents, gemini.Content{
			Role:  "model",
			Parts: []gemini.Part{{FunctionCall: part.FunctionCall}},
		})
		req.Contents = append(req.Contents, gemini.Content{
			Role: "function",
			Parts: []gemini.Part{{
				FunctionResponse: &gemini.FunctionResponse{
					Name:     toolName,
					Response: toolResult,
				},
			}},
		})
	}

	// Max steps exceeded
	o.l.Warnf(ctx, "Agent exceeded max steps (%d)", MaxAgentSteps)
	return "Trợ lý đã suy nghĩ quá lâu (vượt quá số bước cho phép). Vui lòng thử chia nhỏ câu hỏi.", nil
}
