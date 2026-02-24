package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"autonomous-task-management/internal/agent"
	"autonomous-task-management/internal/checklist"
	"autonomous-task-management/internal/task/repository"
	pkgLog "autonomous-task-management/pkg/log"
)

type UpdateChecklistItemTool struct {
	memosRepo    repository.MemosRepository
	vectorRepo   repository.VectorRepository
	checklistSvc checklist.Service
	l            pkgLog.Logger
}

func NewUpdateChecklistItemTool(
	memosRepo repository.MemosRepository,
	vectorRepo repository.VectorRepository,
	checklistSvc checklist.Service,
	l pkgLog.Logger,
) *UpdateChecklistItemTool {
	return &UpdateChecklistItemTool{
		memosRepo:    memosRepo,
		vectorRepo:   vectorRepo,
		checklistSvc: checklistSvc,
		l:            l,
	}
}

func (t *UpdateChecklistItemTool) Name() string {
	return "update_checklist_item"
}

func (t *UpdateChecklistItemTool) Description() string {
	return "Update a checklist item in a task. Can mark items as checked or unchecked by matching text."
}

func (t *UpdateChecklistItemTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"task_id": map[string]interface{}{
				"type":        "string",
				"description": "Memos task ID (UID)",
			},
			"item_text": map[string]interface{}{
				"type":        "string",
				"description": "Text of the checklist item to update (partial match OK)",
			},
			"checked": map[string]interface{}{
				"type":        "boolean",
				"description": "New checked state (true = checked, false = unchecked)",
			},
		},
		"required": []string{"task_id", "item_text", "checked"},
	}
}

type UpdateChecklistItemInput struct {
	TaskID   string `json:"task_id"`
	ItemText string `json:"item_text"`
	Checked  bool   `json:"checked"`
}

type UpdateChecklistItemOutput struct {
	TaskID  string `json:"task_id"`
	Updated bool   `json:"updated"`
	Count   int    `json:"count"`
	Summary string `json:"summary"`
}

func (t *UpdateChecklistItemTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	// Parse input
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}

	var params UpdateChecklistItemInput
	if err := json.Unmarshal(inputBytes, &params); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	t.l.Infof(ctx, "update_checklist_item: task_id=%s item=%q checked=%v", params.TaskID, params.ItemText, params.Checked)

	// Fetch task
	task, err := t.memosRepo.GetTask(ctx, params.TaskID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch task: %w", err)
	}

	// Update checkbox
	output, err := t.checklistSvc.UpdateCheckbox(ctx, checklist.UpdateCheckboxInput{
		Content:      task.Content,
		CheckboxText: params.ItemText,
		Checked:      params.Checked,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update checkbox: %w", err)
	}

	if !output.Updated {
		return UpdateChecklistItemOutput{
			TaskID:  params.TaskID,
			Updated: false,
			Count:   0,
			Summary: fmt.Sprintf("Không tìm thấy checkbox với text: %q", params.ItemText),
		}, nil
	}

	// Update Memos
	if err := t.memosRepo.UpdateTask(ctx, params.TaskID, output.Content); err != nil {
		return nil, fmt.Errorf("failed to update Memos: %w", err)
	}

	// Phase 3 webhook handles re-embedding
	t.l.Infof(ctx, "Updated checklist for task %s", params.TaskID)

	// Generate summary
	action := "unchecked"
	if params.Checked {
		action = "checked"
	}
	summary := fmt.Sprintf("✅ Đã %s %d checkbox(es) matching %q", action, output.Count, params.ItemText)

	return UpdateChecklistItemOutput{
		TaskID:  params.TaskID,
		Updated: true,
		Count:   output.Count,
		Summary: summary,
	}, nil
}

var _ agent.Tool = (*UpdateChecklistItemTool)(nil)
