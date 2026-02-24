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

type GetChecklistProgressTool struct {
	memosRepo    repository.MemosRepository
	checklistSvc checklist.Service
	l            pkgLog.Logger
}

func NewGetChecklistProgressTool(memosRepo repository.MemosRepository, checklistSvc checklist.Service, l pkgLog.Logger) *GetChecklistProgressTool {
	return &GetChecklistProgressTool{
		memosRepo:    memosRepo,
		checklistSvc: checklistSvc,
		l:            l,
	}
}

func (t *GetChecklistProgressTool) Name() string {
	return "get_checklist_progress"
}

func (t *GetChecklistProgressTool) Description() string {
	return "Get checklist progress for a specific task. Returns total, completed, and pending checkboxes."
}

func (t *GetChecklistProgressTool) Parameters() map[string]interface{} {
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

type GetChecklistProgressInput struct {
	TaskID string `json:"task_id"`
}

type GetChecklistProgressOutput struct {
	TaskID     string                   `json:"task_id"`
	Stats      checklist.ChecklistStats `json:"stats"`
	Checkboxes []checklist.Checkbox     `json:"checkboxes"`
	Summary    string                   `json:"summary"`
}

func (t *GetChecklistProgressTool) Execute(ctx context.Context, input map[string]interface{}) (interface{}, error) {
	// Parse input
	inputBytes, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal input: %w", err)
	}

	var params GetChecklistProgressInput
	if err := json.Unmarshal(inputBytes, &params); err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	t.l.Infof(ctx, "get_checklist_progress: task_id=%s", params.TaskID)

	// Fetch task
	task, err := t.memosRepo.GetTask(ctx, params.TaskID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch task: %w", err)
	}

	// Get stats and checkboxes
	stats := t.checklistSvc.GetStats(task.Content)
	checkboxes := t.checklistSvc.ParseCheckboxes(task.Content)

	// Generate summary
	summary := fmt.Sprintf("📊 Tiến độ: %d/%d hoàn thành (%.0f%%)", stats.Completed, stats.Total, stats.Progress)
	if stats.Total == 0 {
		summary = "Task này không có checklist"
	} else if stats.Progress == 100 {
		summary += " ✅ Hoàn thành!"
	}

	return GetChecklistProgressOutput{
		TaskID:     params.TaskID,
		Stats:      stats,
		Checkboxes: checkboxes,
		Summary:    summary,
	}, nil
}

var _ agent.Tool = (*GetChecklistProgressTool)(nil)
