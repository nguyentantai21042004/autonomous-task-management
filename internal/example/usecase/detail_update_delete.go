package usecase

import (
	"context"

	"autonomous-task-management/internal/example"
	repo "autonomous-task-management/internal/example/repository"
)

// Detail retrieves a single Item by ID. Returns ErrItemNotFound when not found.
func (uc *implUseCase) Detail(ctx context.Context, id string) (example.DetailItemOutput, error) {
	item, err := uc.repo.GetOneItem(ctx, repo.GetOneItemOptions{ID: id})
	if err != nil {
		uc.l.Errorf(ctx, "uc.Detail GetOneItem: %v", err)
		return example.DetailItemOutput{}, err
	}
	if item.ID == "" {
		return example.DetailItemOutput{}, example.ErrItemNotFound
	}
	return example.DetailItemOutput{Item: item}, nil
}

// Update modifies an existing Item. Returns ErrItemNotFound when not found.
func (uc *implUseCase) Update(ctx context.Context, input example.UpdateItemInput) (example.UpdateItemOutput, error) {
	// Ensure item exists
	existing, err := uc.repo.GetOneItem(ctx, repo.GetOneItemOptions{ID: input.ID})
	if err != nil {
		uc.l.Errorf(ctx, "uc.Update GetOneItem: %v", err)
		return example.UpdateItemOutput{}, err
	}
	if existing.ID == "" {
		return example.UpdateItemOutput{}, example.ErrItemNotFound
	}

	item, err := uc.repo.UpdateItem(ctx, repo.UpdateItemOptions{
		ID:          input.ID,
		Name:        uc.coalesce(input.Name, existing.Name),
		Description: uc.coalesce(input.Description, existing.Description),
		Status:      uc.coalesce(input.Status, existing.Status),
	})
	if err != nil {
		uc.l.Errorf(ctx, "uc.Update UpdateItem: %v", err)
		return example.UpdateItemOutput{}, err
	}
	return example.UpdateItemOutput{Item: item}, nil
}

// Delete removes an Item by ID. Returns ErrItemNotFound when not found.
func (uc *implUseCase) Delete(ctx context.Context, id string) error {
	existing, err := uc.repo.GetOneItem(ctx, repo.GetOneItemOptions{ID: id})
	if err != nil {
		uc.l.Errorf(ctx, "uc.Delete GetOneItem: %v", err)
		return err
	}
	if existing.ID == "" {
		return example.ErrItemNotFound
	}
	if err := uc.repo.DeleteItem(ctx, id); err != nil {
		uc.l.Errorf(ctx, "uc.Delete DeleteItem: %v", err)
		return err
	}
	return nil
}
