package task

import "errors"

// Domain-specific errors for the task package.
var (
	ErrEmptyInput    = errors.New("input text is empty")
	ErrNoTasksParsed = errors.New("no tasks parsed from input")
	ErrMemoCreate    = errors.New("failed to create memo")
	ErrEmptyQuery    = errors.New("search query is empty")
)
