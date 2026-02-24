package orchestrator

import (
	"time"

	"autonomous-task-management/pkg/gemini"
)

// SessionMemory holds the recent conversation history for a user.
type SessionMemory struct {
	UserID      string
	Messages    []gemini.Content
	LastUpdated time.Time
}
