package orchestrator

import (
	"autonomous-task-management/internal/agent"
	"autonomous-task-management/pkg/gemini"
	pkgLog "autonomous-task-management/pkg/log"
)

type Orchestrator struct {
	llm      *gemini.Client
	registry *agent.ToolRegistry
	l        pkgLog.Logger
}

func New(llm *gemini.Client, registry *agent.ToolRegistry, l pkgLog.Logger) *Orchestrator {
	return &Orchestrator{
		llm:      llm,
		registry: registry,
		l:        l,
	}
}
