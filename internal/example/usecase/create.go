package usecase

import (
	"context"

	"autonomous-task-management/internal/example"
	repo "autonomous-task-management/internal/example/repository"
)

// Create creates a new Item after checking for name uniqueness.
func (uc *implUseCase) Create(ctx context.Context, input example.CreateItemInput) (example.CreateItemOutput, error) {
	// Business validation: check for duplicate name
	existing, err := uc.repo.GetOneItem(ctx, repo.GetOneItemOptions{Name: input.Name})
	if err != nil {
		uc.l.Errorf(ctx, "uc.Create GetOneItem: %v", err)
		return example.CreateItemOutput{}, err
	}
	if existing.ID != "" {
		return example.CreateItemOutput{}, example.ErrDuplicateName
	}

	// Persist
	item, err := uc.repo.CreateItem(ctx, repo.CreateItemOptions{
		Name:        input.Name,
		Description: input.Description,
	})
	if err != nil {
		uc.l.Errorf(ctx, "uc.Create CreateItem: %v", err)
		return example.CreateItemOutput{}, err
	}

	return example.CreateItemOutput{Item: item}, nil
}
