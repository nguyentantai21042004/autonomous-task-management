package usecase

import (
	"context"
	"fmt"
	"time"

	"autonomous-task-management/internal/model"
	"autonomous-task-management/pkg/llmprovider"
)

func (uc *implUseCase) ProcessQuery(ctx context.Context, sc model.Scope, query string) (string, error) {
	// Inject time context into query
	timeContext := buildTimeContext(uc.timezone)
	enhancedQuery := query + timeContext

	session := uc.getSession(sc.UserID)

	// Build current user message with enhanced query
	userMessage := llmprovider.Message{
		Role:  "user",
		Parts: []llmprovider.Part{{Text: enhancedQuery}},
	}

	// Create request with history
	messages := make([]llmprovider.Message, 0, len(session.Messages)+1)
	messages = append(messages, session.Messages...)
	messages = append(messages, userMessage)

	req := llmprovider.Request{
		SystemInstruction: &llmprovider.Message{
			Parts: []llmprovider.Part{{Text: SystemPromptAgent}},
		},
		Messages: messages,
		Tools:    uc.convertToolsToNormalized(),
	}

	for step := 0; step < MaxAgentSteps; step++ {
		uc.l.Infof(ctx, "%s: "+LogMsgAgentStep, LogPrefixProcessQuery, step+1, MaxAgentSteps)

		// 1. Reason: Ask LLM what to do
		resp, err := uc.llm.GenerateContent(ctx, &req)
		if err != nil {
			return "", fmt.Errorf("%s: "+ErrMsgAgentLLMError+": %w", LogPrefixProcessQuery, step, err)
		}

		if len(resp.Content.Parts) == 0 {
			return "", fmt.Errorf("%s: %s", LogPrefixProcessQuery, ErrMsgEmptyLLMResponse)
		}

		part := resp.Content.Parts[0]

		// 2. Check if LLM wants to call a tool
		if part.FunctionCall == nil {
			// LLM has final answer
			uc.l.Infof(ctx, "%s: "+LogMsgAgentFinished, LogPrefixProcessQuery, step+1)

			// Save to session history
			uc.cacheMutex.Lock()
			session.Messages = append(session.Messages, userMessage)
			session.Messages = append(session.Messages, llmprovider.Message{
				Role:  "assistant",
				Parts: []llmprovider.Part{{Text: part.Text}},
			})

			// Limit history to last N messages
			if len(session.Messages) > MaxSessionHistory {
				session.Messages = session.Messages[len(session.Messages)-MaxSessionHistory:]
			}
			session.LastUpdated = time.Now()
			uc.cacheMutex.Unlock()

			return part.Text, nil
		}

		// 3. Act: Execute the tool
		toolName := part.FunctionCall.Name
		uc.l.Infof(ctx, "%s: "+LogMsgAgentCallingTool, LogPrefixProcessQuery, toolName, part.FunctionCall.Args)

		tool, ok := uc.registry.Get(toolName)
		var toolResult interface{}

		if !ok {
			uc.l.Errorf(ctx, "%s: Tool %s not found", LogPrefixProcessQuery, toolName)
			toolResult = map[string]string{"error": ErrMsgToolNotFound}
		} else {
			// Execute tool
			res, err := tool.Execute(ctx, part.FunctionCall.Args)
			if err != nil {
				uc.l.Errorf(ctx, "%s: "+LogMsgToolExecutionError, LogPrefixProcessQuery, toolName, err)
				toolResult = map[string]string{"error": err.Error()}
			} else {
				toolResult = res
			}
		}

		// 4. Observe: Add tool result to current ReAct session memory
		req.Messages = append(req.Messages, llmprovider.Message{
			Role:  "assistant",
			Parts: []llmprovider.Part{{FunctionCall: part.FunctionCall}},
		})
		req.Messages = append(req.Messages, llmprovider.Message{
			Role: "function",
			Parts: []llmprovider.Part{{
				FunctionResponse: &llmprovider.FunctionResponse{
					Name:     toolName,
					Response: toolResult,
				},
			}},
		})
	}

	uc.l.Warnf(ctx, "%s: "+LogMsgAgentMaxSteps, LogPrefixProcessQuery, MaxAgentSteps)
	return ErrMsgMaxStepsExceeded, nil
}
