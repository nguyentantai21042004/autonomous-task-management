package orchestrator

import (
	"context"
	"fmt"
	"time"

	"autonomous-task-management/pkg/gemini"
)

const MaxAgentSteps = 5

// getSession retrieves or creates session for user
func (o *Orchestrator) getSession(userID string) *SessionMemory {
	o.cacheMutex.Lock()
	defer o.cacheMutex.Unlock()

	session, exists := o.sessionCache[userID]
	if !exists || time.Since(session.LastUpdated) > o.cacheTTL {
		session = &SessionMemory{
			UserID:      userID,
			Messages:    []gemini.Content{},
			LastUpdated: time.Now(),
		}
		o.sessionCache[userID] = session
	}

	return session
}

// cleanupExpiredSessions runs every 5 minutes to remove expired sessions
func (o *Orchestrator) cleanupExpiredSessions() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		o.cacheMutex.Lock()

		now := time.Now()
		expiredKeys := make([]string, 0)

		for userID, session := range o.sessionCache {
			if now.Sub(session.LastUpdated) > o.cacheTTL {
				expiredKeys = append(expiredKeys, userID)
			}
		}

		for _, userID := range expiredKeys {
			delete(o.sessionCache, userID)
		}

		o.cacheMutex.Unlock()

		if len(expiredKeys) > 0 {
			o.l.Infof(context.Background(),
				"Cleaned up %d expired sessions", len(expiredKeys))
		}
	}
}

// ClearSession removes the session memory for a specific user.
func (o *Orchestrator) ClearSession(userID string) {
	o.cacheMutex.Lock()
	defer o.cacheMutex.Unlock()
	delete(o.sessionCache, userID)
}

// ProcessQuery runs ReAct loop: Reason → Act → Observe
func (o *Orchestrator) ProcessQuery(ctx context.Context, userID string, query string) (string, error) {
	loc, err := time.LoadLocation(o.timezone)
	if err != nil {
		loc = time.UTC
	}
	currentTime := time.Now().In(loc)
	dateContext := fmt.Sprintf(
		"Hôm nay là %s, ngày %s. Timezone: %s.",
		currentTime.Weekday().String(),
		currentTime.Format("02/01/2006 15:04:05"),
		currentTime.Location().String(),
	)

	systemPrompt := `Bạn là một trợ lý quản lý công việc thiết kế bởi Agentic.
Nhiệm vụ của bạn là tư vấn, giải đáp lịch trình và hỗ trợ người dùng tạo task.

LUÔN LUÔN ghi nhớ thông tin thời gian sau để nội suy các mốc thời gian tương đối:
` + dateContext + `

QUAN TRỌNG - Xử lý thời gian tương đối:
- Khi người dùng hỏi về "tuần này", "ngày mai", "tháng sau", hãy TỰ ĐỘNG TÍNH TOÁN ngày cụ thể dựa trên thông tin trên.
- KHÔNG BAO GIỜ hỏi ngược lại người dùng về ngày tháng cụ thể.
- Khi gọi tool check_calendar, LUÔN LUÔN truyền start_date và end_date đã tính toán sẵn theo format YYYY-MM-DD.
- Ví dụ: Nếu hôm nay là Monday 24/02/2026 và user hỏi "lịch tuần này", hãy gọi check_calendar với start_date="2026-02-24" và end_date="2026-03-02".

Nếu người dùng hỏi về khả năng hoặc chức năng của bạn, hãy giải thích ngắn gọn rằng bạn có thể:
- Lên lịch và tạo công việc (cả hàng loạt)
- Quản lý Checklist (thêm, xóa, đánh dấu hoàn thành)
- Tìm kiếm ngữ nghĩa cực nhanh (dựa trên Qdrant)
- Cảnh báo và đồng bộ với Google Calendar

Hãy luôn thân thiện, xưng hô là "mình" hoặc "trợ lý".`

	session := o.getSession(userID)

	// Build current user message
	userMessage := gemini.Content{Role: "user", Parts: []gemini.Part{{Text: query}}}

	// Create request with history
	contents := make([]gemini.Content, 0, len(session.Messages)+1)
	contents = append(contents, session.Messages...)
	contents = append(contents, userMessage)

	req := gemini.GenerateRequest{
		SystemInstruction: &gemini.Content{
			Parts: []gemini.Part{{Text: systemPrompt}},
		},
		Contents: contents,
		Tools:    o.registry.ToFunctionDefinitions(),
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

			// Save to session history
			o.cacheMutex.Lock()
			session.Messages = append(session.Messages, userMessage)
			session.Messages = append(session.Messages, gemini.Content{Role: "model", Parts: []gemini.Part{{Text: part.Text}}})

			// Limit history to last 5 turns (10 messages)
			if len(session.Messages) > 10 {
				session.Messages = session.Messages[len(session.Messages)-10:]
			}
			session.LastUpdated = time.Now()
			o.cacheMutex.Unlock()

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

		// 4. Observe: Add tool result to current ReAct session memory (not saved to long term cache until final answer)
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
