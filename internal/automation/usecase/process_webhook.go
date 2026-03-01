package usecase

import (
	"context"
	"fmt"

	"autonomous-task-management/internal/automation"
	"autonomous-task-management/internal/model"
)

// ProcessWebhook processes a webhook event and updates tasks
func (uc *implUseCase) ProcessWebhook(ctx context.Context, sc model.Scope, input automation.ProcessWebhookInput) (automation.ProcessWebhookOutput, error) {
	event := input.Event

	uc.l.Infof(ctx, "Processing webhook: %s/%s from %s", event.EventType, event.Action, event.Repository)

	// CRITICAL: For PR/MR events, only process "merged" action
	if (event.EventType == "pull_request" || event.EventType == "merge_request") && event.Action != "merged" {
		uc.l.Infof(ctx, "Skipping PR/MR event with action: %s (only 'merged' triggers auto-completion)", event.Action)
		return automation.ProcessWebhookOutput{
			TasksUpdated: 0,
			Message:      fmt.Sprintf("PR/MR action '%s' not processed", event.Action),
		}, nil
	}

	// For push events (Action is empty), continue processing
	if event.EventType == "push" {
		uc.l.Infof(ctx, "Processing push event for branch: %s", event.Branch)
	}

	// Find matching tasks
	matches, err := uc.matcher.findMatchingTasks(ctx, event)
	if err != nil {
		return automation.ProcessWebhookOutput{}, fmt.Errorf("failed to find matching tasks: %w", err)
	}

	if len(matches) == 0 {
		uc.l.Infof(ctx, "No matching tasks found for event")
		return automation.ProcessWebhookOutput{
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

	return automation.ProcessWebhookOutput{
		TasksUpdated: len(updatedIDs),
		TaskIDs:      updatedIDs,
		Message:      fmt.Sprintf("Updated %d task(s)", len(updatedIDs)),
	}, nil
}
