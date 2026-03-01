package usecase

import (
	"autonomous-task-management/internal/agent"
	"autonomous-task-management/internal/checklist/tools"
	"autonomous-task-management/internal/task/repository"
	pkgLog "autonomous-task-management/pkg/log"
)

// RegisterAgentTools registers the checklist domain's agent tools into the registry.
// Implements checklist.UseCase. The logger is provided by the caller since
// checklist usecase is stateless (no logger field).
func (uc *implUseCase) RegisterAgentTools(registry *agent.ToolRegistry, memosRepo repository.MemosRepository, vectorRepo repository.VectorRepository, l pkgLog.Logger) {
	registry.Register(tools.NewGetChecklistProgressTool(memosRepo, uc, l))
	if vectorRepo != nil {
		registry.Register(tools.NewUpdateChecklistItemTool(memosRepo, vectorRepo, uc, l))
	}
}

// noopLogger is a no-op logger used when checklist usecase has no logger dependency.
type noopLogger struct{}

func (noopLogger) Debug(ctx interface{}, args ...interface{})             {}
func (noopLogger) Debugf(ctx interface{}, f string, args ...interface{})  {}
func (noopLogger) Info(ctx interface{}, args ...interface{})              {}
func (noopLogger) Infof(ctx interface{}, f string, args ...interface{})   {}
func (noopLogger) Warn(ctx interface{}, args ...interface{})              {}
func (noopLogger) Warnf(ctx interface{}, f string, args ...interface{})   {}
func (noopLogger) Error(ctx interface{}, args ...interface{})             {}
func (noopLogger) Errorf(ctx interface{}, f string, args ...interface{})  {}
func (noopLogger) Fatal(ctx interface{}, args ...interface{})             {}
func (noopLogger) Fatalf(ctx interface{}, f string, args ...interface{})  {}
func (noopLogger) DPanic(ctx interface{}, args ...interface{})            {}
func (noopLogger) DPanicf(ctx interface{}, f string, args ...interface{}) {}
func (noopLogger) Panic(ctx interface{}, args ...interface{})             {}
func (noopLogger) Panicf(ctx interface{}, f string, args ...interface{})  {}
