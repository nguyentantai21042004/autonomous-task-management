package graph

import (
	"context"
	"strings"

	"autonomous-task-management/pkg/llmprovider"
)

// NodeAgent goi LLM, phan tich ket qua, va cap nhat GraphStatus.
// Day la buoc "Reason" trong ReAct, nhung co kha nang PAUSE khi can user input.
//
// Sau khi chay:
//   - FunctionCall safe    → Status=RUNNING, PendingTool set
//   - FunctionCall nguy hiem → Status=WAITING_FOR_HUMAN, PendingTool set
//   - Text la cau hoi      → Status=WAITING_FOR_HUMAN
//   - Text la ket luan     → Status=FINISHED
//   - LLM error / empty   → Status=ERROR
func NodeAgent(
	ctx context.Context,
	state *GraphState,
	llm llmprovider.IManager,
	tools []llmprovider.Tool,
	systemPrompt string,
) error {
	req := &llmprovider.Request{
		SystemInstruction: &llmprovider.Message{
			Parts: []llmprovider.Part{{Text: systemPrompt}},
		},
		Messages: state.Messages,
		Tools:    tools,
	}

	resp, err := llm.GenerateContent(ctx, req)
	if err != nil {
		state.Status = StatusError
		return err
	}

	if len(resp.Content.Parts) == 0 {
		state.Status = StatusError
		return ErrEmptyResponse
	}

	part := resp.Content.Parts[0]

	// Append LLM response vao history
	state.AppendMessage(resp.Content)
	state.CurrentStep++

	if part.FunctionCall != nil {
		state.PendingTool = part.FunctionCall

		if isDangerousOperation(part.FunctionCall.Name) {
			// Yeu cau xac nhan truoc khi thuc thi
			state.Status = StatusWaitingForHuman
		} else {
			// An toan, cho phep chay ngay
			state.Status = StatusRunning
		}
		return nil
	}

	// LLM tra ve text
	if isAskingUser(part.Text) {
		state.Status = StatusWaitingForHuman
	} else {
		state.Status = StatusFinished
	}
	return nil
}

// isDangerousOperation: cac operation can xac nhan truoc khi chay.
// Co the mo rong danh sach nay theo nhu cau.
func isDangerousOperation(toolName string) bool {
	switch toolName {
	case "delete_task", "delete_all_tasks", "complete_all", "bulk_delete":
		return true
	}
	return false
}

// isAskingUser: phat hien LLM dang hoi nguoc lai user thay vi co cau tra loi.
// Dung ket hop dau cau hoi "?" va cac tu chi van de can them thong tin.
func isAskingUser(text string) bool {
	if !strings.Contains(text, "?") {
		return false
	}

	lower := strings.ToLower(text)
	questionIndicators := []string{
		"ban muon", "ban co the", "vui long cho biet",
		"ngay nao", "may gio", "khi nao", "o dau", "the nao",
		"ban muon", "you want", "which", "when", "what time",
	}
	for _, indicator := range questionIndicators {
		if strings.Contains(lower, indicator) {
			return true
		}
	}
	return false
}
