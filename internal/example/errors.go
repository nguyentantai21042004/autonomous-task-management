package example

import "errors"

var (
	ErrItemNotFound   = errors.New("item not found")
	ErrDuplicateName  = errors.New("item name already exists")
	ErrInvalidPayload = errors.New("invalid payload")
)
