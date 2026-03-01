package usecase

import (
	"context"
	"fmt"
	"time"
)

func (uc *implUseCase) SyncTask(ctx context.Context, memoID string) error {
	maxRetries := 3
	backoff := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		// Fetch task from Memos
		task, err := uc.memosRepo.GetTask(ctx, memoID)
		if err != nil {
			uc.l.Warnf(ctx, "sync: fetch memo failed (retry %d/%d): %v", i+1, maxRetries, err)
			time.Sleep(backoff)
			backoff *= 2
			continue
		}

		// Embed to Qdrant
		if err := uc.vectorRepo.EmbedTask(ctx, task); err != nil {
			uc.l.Warnf(ctx, "sync: embed failed (retry %d/%d): %v", i+1, maxRetries, err)
			time.Sleep(backoff)
			backoff *= 2
			continue
		}

		// Success
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
