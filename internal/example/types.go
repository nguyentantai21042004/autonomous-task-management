package example

import "time"

// --- Item Domain Model ---

// Item is the core domain entity managed by this module.
type Item struct {
	ID          string
	Name        string
	Description string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// --- UseCase Inputs ---

type CreateItemInput struct {
	Name        string
	Description string
}

type ListItemsInput struct {
	Status string
	Limit  int
	Offset int
}

type UpdateItemInput struct {
	ID          string
	Name        string
	Description string
	Status      string
}

// --- UseCase Outputs ---

type CreateItemOutput struct {
	Item Item
}

type ListItemsOutput struct {
	Items  []Item
	Total  int
	Limit  int
	Offset int
}

type DetailItemOutput struct {
	Item Item
}

type UpdateItemOutput struct {
	Item Item
}
