package automation

import "errors"

var (
	ErrNoTasksFound = errors.New("automation: no matching tasks found")
)
