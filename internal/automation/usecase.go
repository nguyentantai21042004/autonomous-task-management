package automation

import (
	"context"
	"fmt"

	"autonomous-task-management/internal/checklist"
	"autonomous-task-management/internal/model"
	"autonomous-task-management/internal/task/repository"
	pkgLog "autonomous-task-management/pkg/log"
)

type usecase struct {
	memosRepo    repository.MemosRepository
	vectorRepo   repository.VectorRepository
	checklistSvc checklist.Service
	matcher      *TaskMatcher
	l            pkgLog.Logger
}

// ProcessWebhook processes a webhook event and updates tasks
func (uc *usecase) ProcessWebhook(ctx context.Context, sc model.Scope, input ProcessWebhookInput) (ProcessWebhookOutput, error) {
	event := input.Event

	uc.l.Infof(ctx, "Processing webhook: %s/%s from %s", event.EventType, event.Action, event.Repository)

	// CRITICAL: For PR/MR events, only process "merged" action
	// "closed" without merge means PR was REJECTED/CANCELLED - should NOT auto-complete tasks!
	// For "push" events, Action is empty - allow them through
	if (event.EventType == "pull_request" || event.EventType == "merge_request") && event.Action != "merged" {
		uc.l.Infof(ctx, "Skipping PR/MR event with action: %s (only 'merged' triggers auto-completion)", event.Action)
		return ProcessWebhookOutput{
			TasksUpdated: 0,
			Message:      fmt.Sprintf("PR/MR action '%s' not processed", event.Action),
		}, nil
	}

	// For push events (Action is empty), continue processing
	if event.EventType == "push" {
		uc.l.Infof(ctx, "Processing push event for branch: %s", event.Branch)
	}

	// Find matching tasks
	matches, err := uc.matcher.FindMatchingTasks(ctx, event)
	if err != nil {
		return ProcessWebhookOutput{}, fmt.Errorf("failed to find matching tasks: %w", err)
	}

	if len(matches) == 0 {
		uc.l.Infof(ctx, "No matching tasks found for event")
		return ProcessWebhookOutput{
			TasksUpdated: 0,
			Message:      "No matching tasks found",
		}, nil
	}

	// Update each matched task
	updatedIDs := make([]string, 0)
	for _, match := range matches {
		if err := uc.updateTaskChecklist(ctx, match.TaskID, match.Content); err != nil {
			uc.l.Errorf(ctx, "Failed to update task %s: %v", match.TaskID, err)
			continue
		}
		updatedIDs = append(updatedIDs, match.TaskID)
	}

	return ProcessWebhookOutput{
		TasksUpdated: len(updatedIDs),
		TaskIDs:      updatedIDs,
		Message:      fmt.Sprintf("Updated %d task(s)", len(updatedIDs)),
	}, nil
}

// updateTaskChecklist updates all checkboxes in a task to checked
func (uc *usecase) updateTaskChecklist(ctx context.Context, taskID string, content string) error {
	// Check if task has checkboxes
	stats := uc.checklistSvc.GetStats(content)
	if stats.Total == 0 {
		uc.l.Infof(ctx, "Task %s has no checkboxes, skipping", taskID)
		return nil
	}

	// Update all checkboxes to checked
	updatedContent := uc.checklistSvc.UpdateAllCheckboxes(content, true)

	// OPTIMIZATION: Check if the content actually changed before submitting
	if updatedContent == content {
		uc.l.Infof(ctx, "Task %s already completed or has no changes, skipping update", taskID)
		return nil
	}

	// Update Memos
	if err := uc.memosRepo.UpdateTask(ctx, taskID, updatedContent); err != nil {
		return fmt.Errorf("failed to update Memos: %w", err)
	}

	// DO NOTHING - Phase 3 webhook will handle re-embedding automatically
	// This prevents double embedding and race conditions
	uc.l.Infof(ctx, "Updated task %s (%d/%d checkboxes), Phase 3 webhook will re-embed",
		taskID, stats.Total, stats.Total)
	return nil
}

// CompleteTask manually marks a task as complete
func (uc *usecase) CompleteTask(ctx context.Context, sc model.Scope, taskID string) error {
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

// ArchiveCompletedTasks archives all fully completed tasks
func (uc *usecase) ArchiveCompletedTasks(ctx context.Context, sc model.Scope) (int, error) {
	// This would require listing all tasks and checking completion
	// For now, return not implemented
	// In production, you'd want to:
	// 1. List all tasks from Memos
	// 2. Check each for full completion
	// 3. Archive completed ones (add #archived tag or move to archive)

	uc.l.Infof(ctx, "Archive completed tasks not yet implemented")
	return 0, nil
}
