package graph

import (
	"context"
	"fmt"

	"autonomous-task-management/internal/agent"
	"autonomous-task-management/pkg/llmprovider"
	pkgLog "autonomous-task-management/pkg/log"
)

// MaxGraphSteps: gioi han buoc de tranh infinite loop.
// Tang tu 5 (V1.2) len 10 vi pause/resume giam token waste.
const MaxGraphSteps = 10

// Engine dieu phoi viec thuc thi Graph:
//   - Goi NodeAgent de reason
//   - Goi NodeExecuteTool de act
//   - Dung lai khi WAITING_FOR_HUMAN
//   - Ket thuc khi FINISHED hoac ERROR
type Engine struct {
	llm          llmprovider.IManager
	registry     *agent.ToolRegistry
	l            pkgLog.Logger
	systemPrompt string
	tools        []llmprovider.Tool
}

// NewEngine tao mot Engine moi.
func NewEngine(
	llm llmprovider.IManager,
	registry *agent.ToolRegistry,
	l pkgLog.Logger,
	systemPrompt string,
) *Engine {
	return &Engine{
		llm:          llm,
		registry:     registry,
		l:            l,
		systemPrompt: systemPrompt,
		tools:        registry.ToFunctionDefinitions(),
	}
}

// Run thuc thi do thi tu trang thai hien tai cua state.
// Se dung lai khi: FINISHED, WAITING_FOR_HUMAN, ERROR, hoac MaxGraphSteps.
// Caller co trach nhiem luu state vao cache truoc va sau khi goi Run.
func (e *Engine) Run(ctx context.Context, state *GraphState) error {
	for state.CurrentStep < MaxGraphSteps {
		e.l.Infof(ctx, "graph.engine: step=%d status=%s pending_tool=%v",
			state.CurrentStep, state.Status, state.PendingTool != nil)

		switch state.Status {
		case StatusRunning:
			if state.PendingTool != nil {
				// Co tool pending → chay tool truoc, sau do NodeAgent se reason tiep
				if err := NodeExecuteTool(ctx, state, e.registry); err != nil {
					return fmt.Errorf("NodeExecuteTool: %w", err)
				}
			} else {
				// Khong co tool pending → goi NodeAgent de reason
				if err := NodeAgent(ctx, state, e.llm, e.tools, e.systemPrompt); err != nil {
					return fmt.Errorf("NodeAgent: %w", err)
				}
			}

		case StatusWaitingForHuman:
			// Dung lai, caller se luu state vao cache
			e.l.Infof(ctx, "graph.engine: pausing — waiting for human input at step %d", state.CurrentStep)
			return nil

		case StatusFinished:
			e.l.Infof(ctx, "graph.engine: finished at step %d", state.CurrentStep)
			return nil

		case StatusError:
			e.l.Warnf(ctx, "graph.engine: error state at step %d", state.CurrentStep)
			return nil
		}
	}

	e.l.Warnf(ctx, "graph.engine: exceeded MaxGraphSteps (%d), forcing finish", MaxGraphSteps)
	state.Status = StatusFinished
	return nil
}

// GetLastResponse tra ve noi dung text cua tin nhan assistant cuoi cung.
// Tra ve "" neu khong tim thay.
func (e *Engine) GetLastResponse(state *GraphState) string {
	for i := len(state.Messages) - 1; i >= 0; i-- {
		msg := state.Messages[i]
		if msg.Role == "assistant" && len(msg.Parts) > 0 && msg.Parts[0].Text != "" {
			return msg.Parts[0].Text
		}
	}
	return ""
}
