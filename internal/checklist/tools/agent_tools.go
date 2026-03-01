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

// getChecklistProgressTool fetches checklist stats for a task.
type getChecklistProgressTool struct {
	memosRepo   repository.MemosRepository
	checklistUC checklist.UseCase
	l           pkgLog.Logger
}

type getChecklistProgressInput struct {
	TaskID string `json:"task_id"`
}

type getChecklistProgressOutput struct {
	TaskID     string                   `json:"task_id"`
	Stats      checklist.ChecklistStats `json:"stats"`
	Checkboxes []checklist.Checkbox     `json:"checkboxes"`
	Summary    string                   `json:"summary"`
}

func (t *getChecklistProgressTool) Name() string { return "get_checklist_progress" }

func (t *getChecklistProgressTool) Description() string {
	return "Get checklist progress for a specific task. Returns total, completed, and pending checkboxes."
}

func (t *getChecklistProgressTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"task_id": map[string]interface{}{
				"type":        "string",
				"description": "Memos task ID (UID)",
			},
		},
		"required": []string{"task_id"},
	}
}

func (t *getChecklistProgressTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}
	var params getChecklistProgressInput
	if err := json.Unmarshal(inputBytes, &params); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	t.l.Infof(ctx, "get_checklist_progress: task_id=%s", params.TaskID)

	task, err := t.memosRepo.GetTask(ctx, params.TaskID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch task: %w", err)
	}

	stats := t.checklistUC.GetStats(task.Content)
	checkboxes := t.checklistUC.ParseCheckboxes(task.Content)

	summary := fmt.Sprintf("📊 Tiến độ: %d/%d hoàn thành (%.0f%%)", stats.Completed, stats.Total, stats.Progress)
	if stats.Total == 0 {
		summary = "Task này không có checklist"
	} else if stats.Progress == 100 {
		summary += " ✅ Hoàn thành!"
	}

	return getChecklistProgressOutput{
		TaskID:     params.TaskID,
		Stats:      stats,
		Checkboxes: checkboxes,
		Summary:    summary,
	}, nil
}

var _ agent.Tool = (*getChecklistProgressTool)(nil)

// updateChecklistItemTool updates a checkbox by text match.
type updateChecklistItemTool struct {
	memosRepo   repository.MemosRepository
	vectorRepo  repository.VectorRepository
	checklistUC checklist.UseCase
	l           pkgLog.Logger
}

type updateChecklistItemInput struct {
	TaskID   string `json:"task_id"`
	ItemText string `json:"item_text"`
	Checked  bool   `json:"checked"`
}

type updateChecklistItemOutput struct {
	TaskID  string `json:"task_id"`
	Updated bool   `json:"updated"`
	Count   int    `json:"count"`
	Summary string `json:"summary"`
}

func (t *updateChecklistItemTool) Name() string { return "update_checklist_item" }

func (t *updateChecklistItemTool) Description() string {
	return "Update a checklist item in a task. Can mark items as checked or unchecked by matching text."
}

func (t *updateChecklistItemTool) Parameters() map[string]interface{} {
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

func (t *updateChecklistItemTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}
	var params updateChecklistItemInput
	if err := json.Unmarshal(inputBytes, &params); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	t.l.Infof(ctx, "update_checklist_item: task_id=%s item=%q checked=%v", params.TaskID, params.ItemText, params.Checked)

	task, err := t.memosRepo.GetTask(ctx, params.TaskID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch task: %w", err)
	}

	output, err := t.checklistUC.UpdateCheckbox(ctx, checklist.UpdateCheckboxInput{
		Content:      task.Content,
		CheckboxText: params.ItemText,
		Checked:      params.Checked,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update checkbox: %w", err)
	}

	if !output.Updated {
		return updateChecklistItemOutput{
			TaskID:  params.TaskID,
			Updated: false,
			Count:   0,
			Summary: fmt.Sprintf("Không tìm thấy checkbox với text: %q", params.ItemText),
		}, nil
	}

	if err := t.memosRepo.UpdateTask(ctx, params.TaskID, output.Content); err != nil {
		return nil, fmt.Errorf("failed to update Memos: %w", err)
	}

	t.l.Infof(ctx, "Updated checklist for task %s", params.TaskID)

	action := "unchecked"
	if params.Checked {
		action = "checked"
	}

	return updateChecklistItemOutput{
		TaskID:  params.TaskID,
		Updated: true,
		Count:   output.Count,
		Summary: fmt.Sprintf("✅ Đã %s %d checkbox(es) matching %q", action, output.Count, params.ItemText),
	}, nil
}

var _ agent.Tool = (*updateChecklistItemTool)(nil)

// NewChecklistTools creates all checklist-related agent tools.
// Called by checklist.UseCase.RegisterAgentTools.
func NewGetChecklistProgressTool(memosRepo repository.MemosRepository, checklistUC checklist.UseCase, l pkgLog.Logger) agent.Tool {
	return &getChecklistProgressTool{memosRepo: memosRepo, checklistUC: checklistUC, l: l}
}

func NewUpdateChecklistItemTool(memosRepo repository.MemosRepository, vectorRepo repository.VectorRepository, checklistUC checklist.UseCase, l pkgLog.Logger) agent.Tool {
	return &updateChecklistItemTool{memosRepo: memosRepo, vectorRepo: vectorRepo, checklistUC: checklistUC, l: l}
}
