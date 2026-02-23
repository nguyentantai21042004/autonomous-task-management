package usecase

import (
	"context"

	"autonomous-task-management/internal/example"
	repo "autonomous-task-management/internal/example/repository"
)

// List returns a paginated list of Items.
func (uc *implUseCase) List(ctx context.Context, input example.ListItemsInput) (example.ListItemsOutput, error) {
	items, total, err := uc.repo.ListItems(ctx, repo.ListItemsOptions{
		Status: input.Status,
		Limit:  input.Limit,
		Offset: input.Offset,
	})
	if err != nil {
		uc.l.Errorf(ctx, "uc.List ListItems: %v", err)
		return example.ListItemsOutput{}, err
	}

	return example.ListItemsOutput{
		Items:  items,
		Total:  total,
		Limit:  input.Limit,
		Offset: input.Offset,
	}, nil
}
