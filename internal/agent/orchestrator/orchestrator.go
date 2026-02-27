package orchestrator

import (
	"context"
	"fmt"
	"time"

	"autonomous-task-management/pkg/gemini"
)

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

// cleanupExpiredSessions runs periodically to remove expired sessions
func (o *Orchestrator) cleanupExpiredSessions() {
	ticker := time.NewTicker(SessionCleanupInterval * time.Minute)
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
				"%s: "+LogMsgSessionsCleanedUp, LogPrefixCleanupSessions, len(expiredKeys))
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
	// Inject time context into query using extracted utility
	timeContext := buildTimeContext(o.timezone)
	enhancedQuery := query + timeContext

	session := o.getSession(userID)

	// Build current user message with enhanced query
	userMessage := gemini.Content{Role: "user", Parts: []gemini.Part{{Text: enhancedQuery}}}

	// Create request with history
	contents := make([]gemini.Content, 0, len(session.Messages)+1)
	contents = append(contents, session.Messages...)
	contents = append(contents, userMessage)

	req := gemini.GenerateRequest{
		SystemInstruction: &gemini.Content{
			Parts: []gemini.Part{{Text: SystemPromptAgent}},
		},
		Contents: contents,
		Tools:    o.registry.ToFunctionDefinitions(),
	}

	for step := 0; step < MaxAgentSteps; step++ {
		o.l.Infof(ctx, "%s: "+LogMsgAgentStep, LogPrefixProcessQuery, step+1, MaxAgentSteps)

		// 1. Reason: Ask LLM what to do
		resp, err := o.llm.GenerateContent(ctx, req)
		if err != nil {
			return "", fmt.Errorf("%s: "+ErrMsgAgentLLMError+": %w", LogPrefixProcessQuery, step, err)
		}

		if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
			return "", fmt.Errorf("%s: %s", LogPrefixProcessQuery, ErrMsgEmptyLLMResponse)
		}

		part := resp.Candidates[0].Content.Parts[0]

		// 2. Check if LLM wants to call a tool
		if part.FunctionCall == nil {
			// LLM has final answer
			o.l.Infof(ctx, "%s: "+LogMsgAgentFinished, LogPrefixProcessQuery, step+1)

			// Save to session history
			o.cacheMutex.Lock()
			session.Messages = append(session.Messages, userMessage)
			session.Messages = append(session.Messages, gemini.Content{Role: "model", Parts: []gemini.Part{{Text: part.Text}}})

			// Limit history to last N messages
			if len(session.Messages) > MaxSessionHistory {
				session.Messages = session.Messages[len(session.Messages)-MaxSessionHistory:]
			}
			session.LastUpdated = time.Now()
			o.cacheMutex.Unlock()

			return part.Text, nil
		}

		// 3. Act: Execute the tool
		toolName := part.FunctionCall.Name
		o.l.Infof(ctx, "%s: "+LogMsgAgentCallingTool, LogPrefixProcessQuery, toolName, part.FunctionCall.Args)

		tool, ok := o.registry.Get(toolName)
		var toolResult interface{}

		if !ok {
			o.l.Errorf(ctx, "%s: Tool %s not found", LogPrefixProcessQuery, toolName)
			toolResult = map[string]string{"error": ErrMsgToolNotFound}
		} else {
			// Execute tool
			res, err := tool.Execute(ctx, part.FunctionCall.Args)
			if err != nil {
				o.l.Errorf(ctx, "%s: "+LogMsgToolExecutionError, LogPrefixProcessQuery, toolName, err)
				toolResult = map[string]string{"error": err.Error()}
			} else {
				toolResult = res
			}
		}

		// 4. Observe: Add tool result to current ReAct session memory
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
	o.l.Warnf(ctx, "%s: "+LogMsgAgentMaxSteps, LogPrefixProcessQuery, MaxAgentSteps)
	return ErrMsgMaxStepsExceeded, nil
}

// GetSession exposes session for router to access conversation history
func (o *Orchestrator) GetSession(userID string) *SessionMemory {
	return o.getSession(userID)
}
