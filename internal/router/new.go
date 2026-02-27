package router

import (
	"context"

	"autonomous-task-management/pkg/gemini"
	"autonomous-task-management/pkg/log"
)

// Router is the interface for semantic routing
type Router interface {
	Classify(ctx context.Context, message string, conversationHistory []string) (RouterOutput, error)
}

// SemanticRouter classifies user intent using LLM
type SemanticRouter struct {
	llm *gemini.Client
	l   log.Logger
}

// Ensure SemanticRouter implements Router interface
var _ Router = (*SemanticRouter)(nil)

// New creates a new SemanticRouter
// Convention: Factory function returns concrete type (not interface) for internal packages
func New(llm *gemini.Client, l log.Logger) *SemanticRouter {
	return &SemanticRouter{
		llm: llm,
		l:   l,
	}
}
