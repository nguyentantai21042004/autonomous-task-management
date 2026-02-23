package postgre

import (
	"fmt"
	"strings"

	repo "autonomous-task-management/internal/example/repository"
)

// buildGetOneQuery builds WHERE clause + args for GetOneItem.
// All non-empty fields are applied as AND conditions.
func (r *implRepository) buildGetOneQuery(opt repo.GetOneItemOptions) (string, []any) {
	var conditions []string
	var args []any
	idx := 1

	if opt.ID != "" {
		conditions = append(conditions, fmt.Sprintf("id = $%d", idx))
		args = append(args, opt.ID)
		idx++
	}
	if opt.Name != "" {
		conditions = append(conditions, fmt.Sprintf("name = $%d", idx))
		args = append(args, opt.Name)
		idx++
	}

	if len(conditions) == 0 {
		return "1=1", args
	}
	return strings.Join(conditions, " AND "), args
}

// buildCountQuery builds WHERE clause + args for counting Items (no pagination).
func (r *implRepository) buildCountQuery(opt repo.ListItemsOptions) (string, []any) {
	var conditions []string
	var args []any
	idx := 1

	if opt.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", idx))
		args = append(args, opt.Status)
		idx++
	}

	if len(conditions) == 0 {
		return "1=1", args
	}
	return strings.Join(conditions, " AND "), args
}

// buildListQuery builds the full WHERE + ORDER + LIMIT + OFFSET clause for ListItems.
func (r *implRepository) buildListQuery(opt repo.ListItemsOptions) (string, []any) {
	var parts []string
	var conditions []string
	var args []any
	idx := 1

	// Filters
	if opt.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", idx))
		args = append(args, opt.Status)
		idx++
	}

	if len(conditions) > 0 {
		parts = append(parts, "WHERE "+strings.Join(conditions, " AND "))
	}

	// Sorting
	orderBy := opt.OrderBy
	if orderBy == "" {
		orderBy = "created_at DESC"
	}
	parts = append(parts, fmt.Sprintf("ORDER BY %s", orderBy))

	// Pagination
	if opt.Limit > 0 {
		parts = append(parts, fmt.Sprintf("LIMIT $%d", idx))
		args = append(args, opt.Limit)
		idx++
	}
	if opt.Offset > 0 {
		parts = append(parts, fmt.Sprintf("OFFSET $%d", idx))
		args = append(args, opt.Offset)
	}

	return strings.Join(parts, " "), args
}
