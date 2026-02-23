package postgre

import (
	"database/sql"
	"fmt"

	"autonomous-task-management/internal/example/repository"
	"autonomous-task-management/pkg/log"
)

type implRepository struct {
	db *sql.DB
	l  log.Logger
}

// New creates a new PostgreSQL-backed Repository for the example domain.
func New(db *sql.DB, l log.Logger) repository.Repository {
	if db == nil {
		panic("example/repository/postgre: db is required")
	}
	return &implRepository{db: db, l: l}
}

// dsn is a helper to return a method-scoped context string for logging.
func (r *implRepository) dsn(method string) string {
	return fmt.Sprintf("example/repository/postgre.%s", method)
}
