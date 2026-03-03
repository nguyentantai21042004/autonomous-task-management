package usecase

import (
	"sync"
	"time"

	"autonomous-task-management/internal/agent"
	"autonomous-task-management/pkg/llmprovider"
	pkgLog "autonomous-task-management/pkg/log"
)

type implUseCase struct {
	llm          llmprovider.IManager
	registry     *agent.ToolRegistry
	l            pkgLog.Logger
	timezone     string
	sessionCache map[string]*agent.SessionMemory
	cacheMutex   sync.RWMutex
	cacheTTL     time.Duration
	stopCleanup  chan struct{} // Channel to stop cleanup goroutine
}

func New(llm llmprovider.IManager, registry *agent.ToolRegistry, l pkgLog.Logger, timezone string) agent.UseCase {
	if timezone == "" {
		timezone = "Asia/Ho_Chi_Minh"
	}
	uc := &implUseCase{
		llm:          llm,
		registry:     registry,
		l:            l,
		timezone:     timezone,
		sessionCache: make(map[string]*agent.SessionMemory),
		cacheTTL:     10 * time.Minute,
		stopCleanup:  make(chan struct{}),
	}

	go uc.cleanupExpiredSessions()

	return uc
}
