package orchestrator

import (
	"sync"
	"time"

	"autonomous-task-management/internal/agent"
	"autonomous-task-management/pkg/llmprovider"
	pkgLog "autonomous-task-management/pkg/log"
)

type Orchestrator struct {
	llm          *llmprovider.Manager
	registry     *agent.ToolRegistry
	l            pkgLog.Logger
	timezone     string
	sessionCache map[string]*SessionMemory
	cacheMutex   sync.RWMutex
	cacheTTL     time.Duration
}

func New(llm *llmprovider.Manager, registry *agent.ToolRegistry, l pkgLog.Logger, timezone string) *Orchestrator {
	if timezone == "" {
		timezone = "Asia/Ho_Chi_Minh"
	}
	o := &Orchestrator{
		llm:          llm,
		registry:     registry,
		l:            l,
		timezone:     timezone,
		sessionCache: make(map[string]*SessionMemory),
		cacheTTL:     10 * time.Minute,
	}

	go o.cleanupExpiredSessions()

	return o
}
