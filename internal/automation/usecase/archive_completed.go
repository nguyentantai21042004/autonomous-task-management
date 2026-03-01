package usecase

import (
	"context"

	"autonomous-task-management/internal/model"
)

// ArchiveCompletedTasks archives all fully completed tasks
func (uc *implUseCase) ArchiveCompletedTasks(ctx context.Context, sc model.Scope) (int, error) {
	uc.l.Infof(ctx, "Archive completed tasks not yet implemented")
	return 0, nil
}
