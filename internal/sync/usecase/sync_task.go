package usecase

import (
	"context"
	"fmt"
	"time"
)

// SyncTask debounces re-embed requests for the same memoID.
// Nếu cùng một memoID được gọi nhiều lần trong debounceDelay → chỉ chạy 1 lần cuối.
func (uc *implUseCase) SyncTask(ctx context.Context, memoID string) error {
	uc.mu.Lock()
	if existing, ok := uc.timers[memoID]; ok {
		existing.Stop()
	}

	timer := time.AfterFunc(debounceDelay, func() {
		bgCtx := context.Background()
		uc.mu.Lock()
		delete(uc.timers, memoID)
		uc.mu.Unlock()

		if err := uc.doSync(bgCtx, memoID); err != nil {
			uc.l.Errorf(bgCtx, "sync: debounced sync failed for %s: %v", memoID, err)
		}
	})
	uc.timers[memoID] = timer
	uc.mu.Unlock()

	return nil
}

// doSync thực sự fetch + embed, với retry.
func (uc *implUseCase) doSync(ctx context.Context, memoID string) error {
	maxRetries := 3
	backoff := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		task, err := uc.memosRepo.GetTask(ctx, memoID)
		if err != nil {
			uc.l.Warnf(ctx, "sync: fetch memo failed (retry %d/%d): %v", i+1, maxRetries, err)
			time.Sleep(backoff)
			backoff *= 2
			continue
		}

		if err := uc.vectorRepo.EmbedTask(ctx, task); err != nil {
			uc.l.Warnf(ctx, "sync: embed failed (retry %d/%d): %v", i+1, maxRetries, err)
			time.Sleep(backoff)
			backoff *= 2
			continue
		}

		uc.l.Infof(ctx, "sync: successfully synced task %s to Qdrant", memoID)
		return nil
	}

	return fmt.Errorf("sync: failed to sync task %s after %d retries", memoID, maxRetries)
}

func (uc *implUseCase) DeleteTask(ctx context.Context, memoID string) error {
	if err := uc.vectorRepo.DeleteTask(ctx, memoID); err != nil {
		return fmt.Errorf("sync: failed to delete task %s: %w", memoID, err)
	}
	uc.l.Infof(ctx, "sync: deleted task %s from Qdrant", memoID)
	return nil
}
