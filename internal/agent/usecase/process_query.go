package usecase

import (
	"context"
	"strings"

	"autonomous-task-management/internal/agent/graph"
	"autonomous-task-management/internal/model"
	"autonomous-task-management/pkg/llmprovider"
)

// ProcessQuery xu ly natural language query bang Graph Engine.
//
// So voi V1.2 (for loop bi reset sau moi tin nhan), V2.0:
//   - Load GraphState tu LRU cache → co the resume tu giua chung
//   - Neu State = WAITING_FOR_HUMAN → xu ly confirm / cancel / resume
//   - Goi engine.Run() → engine co the PAUSE lai neu can them user input
//   - Luu state vao cache (ke ca khi WAITING, de resume sau)
func (uc *implUseCase) ProcessQuery(ctx context.Context, sc model.Scope, query string) (string, error) {
	// Inject time context de agent hieu "tuan nay", "ngay mai"
	timeContext := buildTimeContext(uc.timezone)
	enhancedQuery := query + timeContext

	// Load hoac tao moi GraphState
	state, ok := uc.stateCache.Get(sc.UserID)
	if !ok || state.IsExpired() {
		state = graph.NewGraphState(sc.UserID)
	}

	// Append user message vao state
	state.AppendMessage(llmprovider.Message{
		Role:  "user",
		Parts: []llmprovider.Part{{Text: enhancedQuery}},
	})

	// Xu ly theo trang thai hien tai cua graph
	switch state.Status {
	case graph.StatusWaitingForHuman:
		if state.PendingTool != nil {
			// Dangerous operation dang cho confirm
			if isUserConfirmed(query) {
				// User dong y → chay tiep tool
				state.Status = graph.StatusRunning
			} else {
				// User tu choi → huy bo
				state.Status = graph.StatusFinished
				state.PendingTool = nil
				state.Touch()
				uc.stateCache.Add(sc.UserID, state)
				return "Da huy thao tac.", nil
			}
		} else {
			// LLM da hoi user → gio co answer → tiep tuc reason
			state.Status = graph.StatusRunning
		}
	default:
		// Tin nhan moi hoac FINISHED/ERROR → bat dau tu dau
		state.Status = graph.StatusRunning
		state.CurrentStep = 0
	}

	// Chay Graph Engine
	if err := uc.engine.Run(ctx, state); err != nil {
		return "", err
	}

	// Context compression: giam token cost khi history dai
	state.CompressIfNeeded()
	state.TrimHistory()
	state.Touch()

	// Luu state lai (ke ca khi WAITING_FOR_HUMAN de resume sau)
	uc.stateCache.Add(sc.UserID, state)

	response := uc.engine.GetLastResponse(state)
	if response == "" && state.Status == graph.StatusWaitingForHuman {
		// Engine dung de hoi user, lay message assistant cuoi trong messages
		response = getLastAssistantMessage(state)
	}

	return response, nil
}

// isUserConfirmed: kiem tra user co dong y voi dangerous operation khong.
func isUserConfirmed(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	confirmWords := []string{"ok", "yes", "dong y", "xac nhan", "co", "duoc", "chac chan"}
	for _, word := range confirmWords {
		if lower == word || strings.HasPrefix(lower, word+" ") {
			return true
		}
	}
	return false
}

// getLastAssistantMessage lay text tu message assistant cuoi trong state.Messages.
func getLastAssistantMessage(state *graph.GraphState) string {
	for i := len(state.Messages) - 1; i >= 0; i-- {
		msg := state.Messages[i]
		if msg.Role == "assistant" && len(msg.Parts) > 0 {
			if msg.Parts[0].Text != "" {
				return msg.Parts[0].Text
			}
		}
	}
	return ErrMsgMaxStepsExceeded
}
