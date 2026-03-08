package usecase

import (
	"sync"
	"time"

	pkgSync "autonomous-task-management/internal/sync"
	"autonomous-task-management/internal/task/repository"
	pkgLog "autonomous-task-management/pkg/log"
)

// debounceDelay: khoảng thời gian chờ trước khi re-embed.
// Nếu có event mới trong thời gian này → timer bị reset → chỉ chạy 1 lần cuối cùng.
const debounceDelay = 2 * time.Second

type implUseCase struct {
	memosRepo  repository.MemosRepository
	vectorRepo repository.VectorRepository
	l          pkgLog.Logger

	// debounce: mỗi memoID có 1 pending timer
	mu       sync.Mutex
	timers   map[string]*time.Timer
}

func New(memosRepo repository.MemosRepository, vectorRepo repository.VectorRepository, l pkgLog.Logger) pkgSync.UseCase {
	return &implUseCase{
		memosRepo:  memosRepo,
		vectorRepo: vectorRepo,
		l:          l,
		timers:     make(map[string]*time.Timer),
	}
}
