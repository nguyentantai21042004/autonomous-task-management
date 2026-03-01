package usecase

import (
	"context"
	"fmt"

	"autonomous-task-management/internal/model"
)

// CompleteTask manually marks a task as complete
func (uc *implUseCase) CompleteTask(ctx context.Context, sc model.Scope, taskID string) error {
	// Fetch task
	task, err := uc.memosRepo.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to fetch task: %w", err)
	}

	// Update all checkboxes
	updatedContent := uc.checklistSvc.UpdateAllCheckboxes(task.Content, true)

	// OPTIMIZATION: Skip update if nothing changed
	if updatedContent == task.Content {
		uc.l.Infof(ctx, "Task %s already completed or has no checkboxes, skipping update", taskID)
		return nil
	}

	// Update Memos
	if err := uc.memosRepo.UpdateTask(ctx, taskID, updatedContent); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	// Phase 3 webhook handles re-embedding
	uc.l.Infof(ctx, "Manually completed task %s", taskID)
	return nil
}
