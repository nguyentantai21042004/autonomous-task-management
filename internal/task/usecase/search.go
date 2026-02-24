package usecase

import (
	"context"
	"fmt"
	"strings"

	"autonomous-task-management/internal/model"
	"autonomous-task-management/internal/task"
	"autonomous-task-management/internal/task/repository"
)

// Search performs semantic search on tasks.
func (uc *implUseCase) Search(ctx context.Context, sc model.Scope, input task.SearchInput) (task.SearchOutput, error) {
	if input.Query == "" {
		return task.SearchOutput{}, task.ErrEmptyQuery
	}

	uc.l.Infof(ctx, "Search: user=%s query=%q", sc.UserID, input.Query)

	// Default limit
	limit := input.Limit
	if limit <= 0 {
		limit = 10
	}

	// Search in Qdrant
	if uc.vectorRepo == nil {
		uc.l.Errorf(ctx, "Search: vector repository is not initialized (likely missing Voyage API Key)")
		return task.SearchOutput{}, fmt.Errorf("semantic search is currently unavailable")
	}

	searchResults, err := uc.vectorRepo.SearchTasks(ctx, repository.SearchTasksOptions{
		Query: input.Query,
		Limit: limit,
		Tags:  input.Tags,
	})
	if err != nil {
		uc.l.Errorf(ctx, "Search: failed to search in Qdrant: %v", err)
		return task.SearchOutput{}, fmt.Errorf("failed to search: %w", err)
	}

	if len(searchResults) == 0 {
		uc.l.Infof(ctx, "Search: no results found for query %q", input.Query)
		return task.SearchOutput{
			Results: []task.SearchResultItem{},
			Count:   0,
		}, nil
	}

	// Fetch full task details from Memos
	results := make([]task.SearchResultItem, 0, len(searchResults))
	zombieVectors := make([]string, 0) // HOTFIX 4: Track zombie vectors for cleanup

	for _, sr := range searchResults {
		// Fetch from Memos
		memoTask, err := uc.repo.GetTask(ctx, sr.MemoID)
		if err != nil {
			// HOTFIX 4: Self-healing - auto cleanup zombie vectors
			if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "Not Found") {
				uc.l.Warnf(ctx, "Search: Task %s deleted in Memos. Self-healing: removing from Qdrant", sr.MemoID)
				zombieVectors = append(zombieVectors, sr.MemoID)

				// Async cleanup (don't block search)
				go func(memoID string) {
					cleanupCtx := context.Background()
					if err := uc.vectorRepo.DeleteTask(cleanupCtx, memoID); err != nil {
						uc.l.Errorf(cleanupCtx, "Self-healing: Failed to cleanup zombie vector %s: %v", memoID, err)
					} else {
						uc.l.Infof(cleanupCtx, "Self-healing: Successfully cleaned up zombie vector %s", memoID)
					}
				}(sr.MemoID)

				continue
			}

			uc.l.Warnf(ctx, "Search: failed to fetch task %s from Memos: %v", sr.MemoID, err)
			continue
		}

		results = append(results, task.SearchResultItem{
			MemoID:  memoTask.ID,
			MemoURL: memoTask.MemoURL,
			Content: memoTask.Content,
			Score:   sr.Score,
		})
	}

	// HOTFIX 4: Log self-healing stats
	if len(zombieVectors) > 0 {
		uc.l.Infof(ctx, "Search: Self-healing cleaned up %d zombie vectors: %v", len(zombieVectors), zombieVectors)
	}

	uc.l.Infof(ctx, "Search: found %d results (filtered from %d raw results)", len(results), len(searchResults))

	return task.SearchOutput{
		Results: results,
		Count:   len(results),
	}, nil
}
