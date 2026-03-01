package checklist

import (
	"context"

	"autonomous-task-management/internal/agent"
	"autonomous-task-management/internal/task/repository"
	pkgLog "autonomous-task-management/pkg/log"
)

// UseCase defines the business logic interface for the checklist domain.
type UseCase interface {
	// ParseCheckboxes extracts all checkboxes from markdown content
	ParseCheckboxes(content string) []Checkbox

	// GetStats calculates checklist statistics
	GetStats(content string) ChecklistStats

	// UpdateCheckbox updates checkbox state by text match
	UpdateCheckbox(ctx context.Context, input UpdateCheckboxInput) (UpdateCheckboxOutput, error)

	// UpdateAllCheckboxes sets all checkboxes to specified state
	UpdateAllCheckboxes(content string, checked bool) string

	// IsFullyCompleted checks if all checkboxes are checked
	IsFullyCompleted(content string) bool

	// RegisterAgentTools registers this domain's agent tools into the registry.
	// l is provided by the caller since checklist usecase is stateless (no logger field).
	RegisterAgentTools(registry *agent.ToolRegistry, memosRepo repository.MemosRepository, vectorRepo repository.VectorRepository, l pkgLog.Logger)
}
