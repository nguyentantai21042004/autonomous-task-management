package usecase

import (
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"

	"autonomous-task-management/internal/agent"
	"autonomous-task-management/internal/agent/graph"
	"autonomous-task-management/pkg/llmprovider"
	pkgLog "autonomous-task-management/pkg/log"
)

const (
	stateCacheSize = 1000
	stateCacheTTL  = 30 * time.Minute
)

type implUseCase struct {
	llm        llmprovider.IManager
	registry   *agent.ToolRegistry
	l          pkgLog.Logger
	timezone   string
	engine     *graph.Engine
	stateCache *expirable.LRU[string, *graph.GraphState]
}

// New tao agent UseCase moi voi Graph Engine va expirable LRU cache.
// Thay the V1.2 dung map+mutex+cleanup goroutine.
func New(llm llmprovider.IManager, registry *agent.ToolRegistry, l pkgLog.Logger, timezone string) agent.UseCase {
	if timezone == "" {
		timezone = "Asia/Ho_Chi_Minh"
	}

	cache := expirable.NewLRU[string, *graph.GraphState](stateCacheSize, nil, stateCacheTTL)
	engine := graph.NewEngine(llm, registry, l, SystemPromptAgent)

	return &implUseCase{
		llm:        llm,
		registry:   registry,
		l:          l,
		timezone:   timezone,
		engine:     engine,
		stateCache: cache,
	}
}
