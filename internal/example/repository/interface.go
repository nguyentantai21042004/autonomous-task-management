package repository

import (
	"context"

	"autonomous-task-management/internal/example"
)

// Repository is the composed interface for the example domain data store.
type Repository interface {
	ItemRepository
}

// ItemRepository defines all data access methods for the Item entity.
type ItemRepository interface {
	CreateItem(ctx context.Context, opt CreateItemOptions) (example.Item, error)
	GetOneItem(ctx context.Context, opt GetOneItemOptions) (example.Item, error)
	ListItems(ctx context.Context, opt ListItemsOptions) ([]example.Item, int, error)
	UpdateItem(ctx context.Context, opt UpdateItemOptions) (example.Item, error)
	DeleteItem(ctx context.Context, id string) error
}
