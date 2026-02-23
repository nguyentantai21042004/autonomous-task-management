package task

import "errors"

// Domain-specific errors for the task package.
var (
	ErrEmptyInput    = errors.New("task: raw text input cannot be empty")
	ErrNoTasksParsed = errors.New("task: LLM returned zero tasks from input")
	ErrMemoCreate    = errors.New("task: failed to create memo in Memos")
)
