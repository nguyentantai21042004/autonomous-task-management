package agent

import (
	"autonomous-task-management/internal/model"
	"autonomous-task-management/pkg/llmprovider"
	"context"
)

// UseCase defines the business logic interface for the agent domain.
type UseCase interface {
	// ProcessQuery handles a natural language query using a ReAct agent loop.
	ProcessQuery(ctx context.Context, sc model.Scope, query string) (string, error)

	// ClearSession removes conversation history for a user
	ClearSession(userID string)

	// GetSessionMessages retrieves the conversation history for a user
	GetSessionMessages(userID string) []llmprovider.Message
}

// ToolRegistrar is implemented by domains that expose tools to the agent.
// Each domain that has agent tools should implement this on its UseCase.
type ToolRegistrar interface {
	RegisterAgentTools(registry *ToolRegistry)
}
