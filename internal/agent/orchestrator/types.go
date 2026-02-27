package orchestrator

import (
	"time"

	"autonomous-task-management/pkg/llmprovider"
)

// SessionMemory holds the recent conversation history for a user.
type SessionMemory struct {
	UserID      string
	Messages    []llmprovider.Message
	LastUpdated time.Time
}
