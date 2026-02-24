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

	// HOTFIX 2: Calculate week boundaries (Monday-Sunday)
	weekday := int(currentTime.Weekday())
	if weekday == 0 { // Sunday
		weekday = 7
	}
	weekStart := currentTime.AddDate(0, 0, -(weekday - 1)) // Monday
	weekEnd := weekStart.AddDate(0, 0, 6)                  // Sunday
	tomorrow := currentTime.AddDate(0, 0, 1)

	// HOTFIX 2: Hard inject temporal context into user query (don't rely on SystemInstruction)
	timeContext := fmt.Sprintf(
		"\n\n[SYSTEM CONTEXT - Thông tin thời gian hiện tại:"+
			"\n- Hôm nay: %s (%s)"+
			"\n- Tuần này: từ %s đến %s"+
			"\n- Ngày mai: %s"+
			"\n\nQUY TẮC QUAN TRỌNG:"+
			"\n1. Nếu user hỏi về 'tuần này', hãy TỰ ĐỘNG gọi tool với start_date='%s' và end_date='%s'"+
			"\n2. Nếu user hỏi về 'ngày mai', dùng date='%s'"+
			"\n3. KHÔNG BAO GIỜ hỏi ngược lại user về ngày tháng cụ thể"+
			"\n4. Format ngày LUÔN LUÔN là YYYY-MM-DD]",
		currentTime.Format("2006-01-02"),
		currentTime.Weekday().String(),
		weekStart.Format("2006-01-02"),
		weekEnd.Format("2006-01-02"),
		tomorrow.Format("2006-01-02"),
		weekStart.Format("2006-01-02"),
		weekEnd.Format("2006-01-02"),
		tomorrow.Format("2006-01-02"),
	)

	// Inject time context at end of query
	enhancedQuery := query + timeContext

	systemPrompt := `Bạn là một trợ lý quản lý công việc thiết kế bởi Agentic.
Nhiệm vụ của bạn là tư vấn, giải đáp lịch trình và hỗ trợ người dùng tạo task.

Nếu người dùng hỏi về khả năng hoặc chức năng của bạn, hãy giải thích ngắn gọn rằng bạn có thể:
- Lên lịch và tạo công việc (cả hàng loạt)
- Quản lý Checklist (thêm, xóa, đánh dấu hoàn thành)
- Tìm kiếm ngữ nghĩa cực nhanh (dựa trên Qdrant)
- Cảnh báo và đồng bộ với Google Calendar

Hãy luôn thân thiện, xưng hô là "mình" hoặc "trợ lý".`

	session := o.getSession(userID)

	// Build current user message with enhanced query
	userMessage := gemini.Content{Role: "user", Parts: []gemini.Part{{Text: enhancedQuery}}}

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
