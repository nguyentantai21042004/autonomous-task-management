package repository

import "errors"

var (
	ErrFailedToInsert = errors.New("failed to insert record")
	ErrFailedToGet    = errors.New("failed to get record")
	ErrFailedToList   = errors.New("failed to list records")
	ErrFailedToUpdate = errors.New("failed to update record")
	ErrFailedToDelete = errors.New("failed to delete record")
)
