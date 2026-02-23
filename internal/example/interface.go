package example

import "context"

//go:generate mockery --name UseCase
type UseCase interface {
	// Item CRUD
	Create(ctx context.Context, input CreateItemInput) (CreateItemOutput, error)
	List(ctx context.Context, input ListItemsInput) (ListItemsOutput, error)
	Detail(ctx context.Context, id string) (DetailItemOutput, error)
	Update(ctx context.Context, input UpdateItemInput) (UpdateItemOutput, error)
	Delete(ctx context.Context, id string) error
}
