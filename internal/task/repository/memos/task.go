package memos

import (
	"context"
	"fmt"
	"strings"

	"autonomous-task-management/internal/model"
	"autonomous-task-management/internal/task/repository"
	pkgLog "autonomous-task-management/pkg/log"
)

type implRepository struct {
	client      *Client
	memoBaseURL string // e.g. "http://localhost:5230" for deep link generation
	l           pkgLog.Logger
}

// New creates a new Memos repository.
func New(client *Client, memoBaseURL string, l pkgLog.Logger) repository.MemosRepository {
	return &implRepository{
		client:      client,
		memoBaseURL: memoBaseURL,
		l:           l,
	}
}

func (r *implRepository) CreateTask(ctx context.Context, opt repository.CreateTaskOptions) (model.Task, error) {
	content := r.buildMarkdownContent(opt)

	visibility := opt.Visibility
	if visibility == "" {
		visibility = "PRIVATE"
	}

	req := CreateMemoRequest{
		Content:    content,
		Visibility: visibility,
	}

	memo, err := r.client.CreateMemo(ctx, req)
	if err != nil {
		r.l.Errorf(ctx, "memos repository: failed to create memo: %v", err)
		return model.Task{}, err
	}

	return r.memoToTask(memo), nil
}

func (r *implRepository) CreateTasksBatch(ctx context.Context, opts []repository.CreateTaskOptions) ([]model.Task, error) {
	tasks := make([]model.Task, 0, len(opts))
	for i, opt := range opts {
		t, err := r.CreateTask(ctx, opt)
		if err != nil {
			r.l.Errorf(ctx, "memos repository: batch item %d failed: %v", i, err)
			continue // partial success: skip failed items
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (r *implRepository) GetTask(ctx context.Context, id string) (model.Task, error) {
	memo, err := r.client.GetMemo(ctx, id)
	if err != nil {
		return model.Task{}, err
	}
	return r.memoToTask(memo), nil
}

func (r *implRepository) ListTasks(ctx context.Context, opt repository.ListTasksOptions) ([]model.Task, error) {
	limit := opt.Limit
	if limit == 0 {
		limit = 20
	}

	memos, err := r.client.ListMemos(ctx, opt.Tag, limit, opt.Offset)
	if err != nil {
		return nil, err
	}

	tasks := make([]model.Task, 0, len(memos))
	for _, m := range memos {
		tasks = append(tasks, r.memoToTask(&m))
	}
	return tasks, nil
}

// buildMarkdownContent builds the Markdown body for a Memos memo from options.
func (r *implRepository) buildMarkdownContent(opt repository.CreateTaskOptions) string {
	var sb strings.Builder
	sb.WriteString(opt.Content)
	if len(opt.Tags) > 0 {
		sb.WriteString("\n\n")
		sb.WriteString(strings.Join(opt.Tags, " "))
	}
	return sb.String()
}

// memoToTask converts a Memos API Memo object to the internal model.Task.
func (r *implRepository) memoToTask(m *Memo) model.Task {
	uid := m.UID
	// Name format is "memos/{uid}" from the Memos v1 API
	if uid == "" && m.Name != "" {
		parts := strings.SplitN(m.Name, "/", 2)
		if len(parts) == 2 {
			uid = parts[1]
		}
	}

	memoURL := ""
	if uid != "" && r.memoBaseURL != "" {
		memoURL = fmt.Sprintf("%s/m/%s", r.memoBaseURL, uid)
	}

	return model.Task{
		ID:         m.Name,
		UID:        uid,
		Content:    m.Content,
		MemoURL:    memoURL,
		Visibility: m.Visibility,
		CreateTime: m.CreateTime,
		UpdateTime: m.UpdateTime,
	}
}
