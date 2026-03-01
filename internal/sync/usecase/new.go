package usecase

import (
	"autonomous-task-management/internal/sync"
	"autonomous-task-management/internal/task/repository"
	pkgLog "autonomous-task-management/pkg/log"
)

type implUseCase struct {
	memosRepo  repository.MemosRepository
	vectorRepo repository.VectorRepository
	l          pkgLog.Logger
}

func New(memosRepo repository.MemosRepository, vectorRepo repository.VectorRepository, l pkgLog.Logger) sync.UseCase {
	return &implUseCase{
		memosRepo:  memosRepo,
		vectorRepo: vectorRepo,
		l:          l,
	}
}
