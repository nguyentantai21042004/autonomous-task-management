package usecase

import (
	"autonomous-task-management/internal/agent"
	"autonomous-task-management/internal/task/tools"
)

// RegisterAgentTools registers the task domain's agent tools into the registry.
// Implements agent.ToolRegistrar.
func (uc *implUseCase) RegisterAgentTools(registry *agent.ToolRegistry) {
	registry.Register(tools.NewSearchTasksTool(uc, uc.l))

	if uc.calendar != nil {
		registry.Register(tools.NewCheckCalendarTool(uc.calendar, uc.l))
	}
}
