package usecase

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"autonomous-task-management/internal/model"
	"autonomous-task-management/internal/task/repository"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// Minimal mocks
// ---------------------------------------------------------------------------

type mockLogger struct{}

func (m *mockLogger) Debug(ctx context.Context, args ...any)                  {}
func (m *mockLogger) Debugf(ctx context.Context, fmt string, args ...any)    {}
func (m *mockLogger) Info(ctx context.Context, args ...any)                   {}
func (m *mockLogger) Infof(ctx context.Context, fmt string, args ...any)     {}
func (m *mockLogger) Warn(ctx context.Context, args ...any)                   {}
func (m *mockLogger) Warnf(ctx context.Context, fmt string, args ...any)     {}
func (m *mockLogger) Error(ctx context.Context, args ...any)                  {}
func (m *mockLogger) Errorf(ctx context.Context, fmt string, args ...any)    {}
func (m *mockLogger) DPanic(ctx context.Context, args ...any)                 {}
func (m *mockLogger) DPanicf(ctx context.Context, fmt string, args ...any)   {}
func (m *mockLogger) Panic(ctx context.Context, args ...any)                  {}
func (m *mockLogger) Panicf(ctx context.Context, fmt string, args ...any)    {}
func (m *mockLogger) Fatal(ctx context.Context, args ...any)                  {}
func (m *mockLogger) Fatalf(ctx context.Context, fmt string, args ...any)    {}

type countingVectorRepo struct {
	embedCalls atomic.Int32
}

func (r *countingVectorRepo) EmbedTask(_ context.Context, _ model.Task) error {
	r.embedCalls.Add(1)
	return nil
}
func (r *countingVectorRepo) SearchTasks(_ context.Context, _ repository.SearchTasksOptions) ([]repository.SearchResult, error) {
	return nil, nil
}
func (r *countingVectorRepo) SearchTasksWithFilter(_ context.Context, _ repository.SearchTasksOptions) ([]repository.SearchResult, error) {
	return nil, nil
}
func (r *countingVectorRepo) DeleteTask(_ context.Context, _ string) error { return nil }

type staticMemosRepo struct{}

func (r *staticMemosRepo) GetTask(_ context.Context, _ string) (model.Task, error) {
	return model.Task{ID: "memos/1", Content: "test"}, nil
}
func (r *staticMemosRepo) ListTasks(_ context.Context, _ repository.ListTasksOptions) ([]model.Task, error) {
	return nil, nil
}
func (r *staticMemosRepo) CreateTask(_ context.Context, _ repository.CreateTaskOptions) (model.Task, error) {
	return model.Task{}, nil
}
func (r *staticMemosRepo) CreateTasksBatch(_ context.Context, _ []repository.CreateTaskOptions) ([]model.Task, error) {
	return nil, nil
}
func (r *staticMemosRepo) UpdateTask(_ context.Context, _ string, _ string) error { return nil }

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestSyncTask_Debounce_CoalescesRapidCalls(t *testing.T) {
	vectorRepo := &countingVectorRepo{}
	uc := New(&staticMemosRepo{}, vectorRepo, &mockLogger{})

	ctx := context.Background()

	// Fire 5 rapid SyncTask calls for the same memoID
	for i := 0; i < 5; i++ {
		err := uc.SyncTask(ctx, "memos/1")
		assert.NoError(t, err)
	}

	// Returns immediately (async)
	// Wait for debounce delay + buffer
	time.Sleep(debounceDelay + 200*time.Millisecond)

	// Despite 5 calls, EmbedTask should only be called once
	assert.Equal(t, int32(1), vectorRepo.embedCalls.Load(), "debounce should coalesce 5 calls into 1")
}

func TestSyncTask_Debounce_DifferentIDs_RunIndependently(t *testing.T) {
	vectorRepo := &countingVectorRepo{}
	uc := New(&staticMemosRepo{}, vectorRepo, &mockLogger{})

	ctx := context.Background()

	// Fire once for each of 3 different memoIDs
	for _, id := range []string{"memos/1", "memos/2", "memos/3"} {
		err := uc.SyncTask(ctx, id)
		assert.NoError(t, err)
	}

	time.Sleep(debounceDelay + 200*time.Millisecond)

	// Each ID runs independently → 3 embed calls
	assert.Equal(t, int32(3), vectorRepo.embedCalls.Load())
}

func TestSyncTask_Debounce_ReturnsImmediately(t *testing.T) {
	vectorRepo := &countingVectorRepo{}
	uc := New(&staticMemosRepo{}, vectorRepo, &mockLogger{})

	start := time.Now()
	err := uc.SyncTask(context.Background(), "memos/1")
	elapsed := time.Since(start)

	assert.NoError(t, err)
	// Should return in < 50ms (async, no blocking)
	assert.Less(t, elapsed, 50*time.Millisecond)
}
