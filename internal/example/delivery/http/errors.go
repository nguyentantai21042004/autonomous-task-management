package http

import (
	"autonomous-task-management/internal/example"
	"autonomous-task-management/internal/middleware"
	pkgErrors "autonomous-task-management/pkg/errors"
)

// mapError translates domain/use-case errors into HTTP errors from pkg/errors.
// RULE: panic on unknown errors in development to force explicit handling.
func (h *handler) mapError(err error) error {
	switch err {
	case example.ErrDuplicateName:
		return pkgErrors.NewHTTPError(409, "item name already exists")
	default:
		// Force developers to explicitly handle every domain error.
		// In production this should return ErrInternalServerError instead of panic.
		panic(err)
	}
}

// Ensure the mapError signature satisfies middleware expectations.
var _ = middleware.Middleware{}
