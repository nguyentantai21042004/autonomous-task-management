package usecase

import (
	"context"
	"fmt"

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
	for _, sr := range searchResults {
		// Fetch from Memos
		memoTask, err := uc.repo.GetTask(ctx, sr.MemoID)
		if err != nil {
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

	uc.l.Infof(ctx, "Search: found %d results", len(results))

	return task.SearchOutput{
		Results: results,
		Count:   len(results),
	}, nil
}
