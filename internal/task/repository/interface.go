package repository

import (
	"context"

	"autonomous-task-management/internal/model"
)

// MemosRepository is the interface for Memos data access operations.
type MemosRepository interface {
	CreateTask(ctx context.Context, opt CreateTaskOptions) (model.Task, error)
	CreateTasksBatch(ctx context.Context, opts []CreateTaskOptions) ([]model.Task, error)
	GetTask(ctx context.Context, id string) (model.Task, error)
	ListTasks(ctx context.Context, opt ListTasksOptions) ([]model.Task, error)
	UpdateTask(ctx context.Context, id string, content string) error
}

// VectorRepository handles vector operations (Qdrant).
type VectorRepository interface {
	EmbedTask(ctx context.Context, task model.Task) error
	SearchTasks(ctx context.Context, opt SearchTasksOptions) ([]SearchResult, error)
	SearchTasksWithFilter(ctx context.Context, opt SearchTasksOptions) ([]SearchResult, error)
	DeleteTask(ctx context.Context, taskID string) error
}
