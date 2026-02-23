package postgre

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"autonomous-task-management/internal/example"
	repo "autonomous-task-management/internal/example/repository"
)

// CreateItem inserts a new Item row and returns the created entity.
func (r *implRepository) CreateItem(ctx context.Context, opt repo.CreateItemOptions) (example.Item, error) {
	const query = `
		INSERT INTO example_items (name, description, status, created_at, updated_at)
		VALUES ($1, $2, 'active', NOW(), NOW())
		RETURNING id, name, description, status, created_at, updated_at`

	var item example.Item
	err := r.db.QueryRowContext(ctx, query, opt.Name, opt.Description).Scan(
		&item.ID, &item.Name, &item.Description, &item.Status, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		r.l.Errorf(ctx, "%s: %v", r.dsn("CreateItem"), err)
		return example.Item{}, repo.ErrFailedToInsert
	}
	return item, nil
}

// GetOneItem retrieves a single Item by the provided filters (AND condition).
// Returns zero-value Item (ID == "") when not found — do NOT return error for not-found.
func (r *implRepository) GetOneItem(ctx context.Context, opt repo.GetOneItemOptions) (example.Item, error) {
	mods, args := r.buildGetOneQuery(opt)
	baseQuery := `SELECT id, name, description, status, created_at, updated_at FROM example_items`
	query := fmt.Sprintf("%s WHERE %s LIMIT 1", baseQuery, mods)

	var item example.Item
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&item.ID, &item.Name, &item.Description, &item.Status, &item.CreatedAt, &item.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return example.Item{}, nil // not found → zero value, no error
	}
	if err != nil {
		r.l.Errorf(ctx, "%s: %v", r.dsn("GetOneItem"), err)
		return example.Item{}, repo.ErrFailedToGet
	}
	return item, nil
}

// ListItems returns a paginated list of Items and the total count.
func (r *implRepository) ListItems(ctx context.Context, opt repo.ListItemsOptions) ([]example.Item, int, error) {
	// 1. Count total (without pagination)
	countMods, countArgs := r.buildCountQuery(opt)
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM example_items WHERE %s", countMods)
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		r.l.Errorf(ctx, "%s count: %v", r.dsn("ListItems"), err)
		return nil, 0, repo.ErrFailedToList
	}

	// 2. Fetch page
	mods, args := r.buildListQuery(opt)
	query := fmt.Sprintf(
		`SELECT id, name, description, status, created_at, updated_at FROM example_items %s`,
		mods,
	)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.l.Errorf(ctx, "%s: %v", r.dsn("ListItems"), err)
		return nil, 0, repo.ErrFailedToList
	}
	defer rows.Close()

	var items []example.Item
	for rows.Next() {
		var item example.Item
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, 0, repo.ErrFailedToList
		}
		items = append(items, item)
	}
	return items, total, nil
}

// UpdateItem updates an Item by ID and returns the updated entity.
func (r *implRepository) UpdateItem(ctx context.Context, opt repo.UpdateItemOptions) (example.Item, error) {
	const query = `
		UPDATE example_items
		SET name = $1, description = $2, status = $3, updated_at = $4
		WHERE id = $5
		RETURNING id, name, description, status, created_at, updated_at`

	var item example.Item
	err := r.db.QueryRowContext(ctx, query, opt.Name, opt.Description, opt.Status, time.Now(), opt.ID).Scan(
		&item.ID, &item.Name, &item.Description, &item.Status, &item.CreatedAt, &item.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return example.Item{}, nil
	}
	if err != nil {
		r.l.Errorf(ctx, "%s: %v", r.dsn("UpdateItem"), err)
		return example.Item{}, repo.ErrFailedToUpdate
	}
	return item, nil
}

// DeleteItem removes an Item by ID.
func (r *implRepository) DeleteItem(ctx context.Context, id string) error {
	const query = `DELETE FROM example_items WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.l.Errorf(ctx, "%s: %v", r.dsn("DeleteItem"), err)
		return repo.ErrFailedToDelete
	}
	return nil
}
