package repository

// CreateItemOptions holds parameters for inserting a new Item.
type CreateItemOptions struct {
	Name        string
	Description string
}

// GetOneItemOptions holds filter parameters for fetching a single Item.
// All non-empty fields are applied as AND conditions.
type GetOneItemOptions struct {
	ID   string
	Name string
}

// ListItemsOptions holds filter and pagination parameters for listing Items.
type ListItemsOptions struct {
	Status  string
	Limit   int
	Offset  int
	OrderBy string
}

// UpdateItemOptions holds parameters for updating an existing Item.
type UpdateItemOptions struct {
	ID          string
	Name        string
	Description string
	Status      string
}
