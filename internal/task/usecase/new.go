package usecase

import (
	"autonomous-task-management/internal/task"
	"autonomous-task-management/internal/task/repository"
	"autonomous-task-management/pkg/datemath"
	"autonomous-task-management/pkg/llmprovider"
	pkgLog "autonomous-task-management/pkg/log"
	"autonomous-task-management/pkg/voyage"
)

type implUseCase struct {
	l          pkgLog.Logger
	llm        llmprovider.IManager
	calendar   task.CalendarClient
	repo       repository.MemosRepository
	vectorRepo repository.VectorRepository
	dateMath   datemath.IParser
	reranker   *voyage.Reranker // optional; nil = skip reranking
	timezone   string
	memosURL   string
}

// New creates a new task UseCase instance.
// reranker is optional — pass nil to disable cross-encoder reranking.
func New(
	l pkgLog.Logger,
	llm llmprovider.IManager,
	calendar task.CalendarClient,
	repo repository.MemosRepository,
	vectorRepo repository.VectorRepository,
	dateMath datemath.IParser,
	reranker *voyage.Reranker,
	timezone string,
	memosURL string,
) task.UseCase {
	return &implUseCase{
		l:          l,
		llm:        llm,
		calendar:   calendar,
		repo:       repo,
		vectorRepo: vectorRepo,
		dateMath:   dateMath,
		reranker:   reranker,
		timezone:   timezone,
		memosURL:   memosURL,
	}
}
