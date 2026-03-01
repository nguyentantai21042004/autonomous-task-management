package usecase

import (
	"autonomous-task-management/internal/automation"
	"autonomous-task-management/internal/checklist"
	"autonomous-task-management/internal/task/repository"
	pkgLog "autonomous-task-management/pkg/log"
)

type implUseCase struct {
	memosRepo    repository.MemosRepository
	vectorRepo   repository.VectorRepository
	checklistSvc checklist.UseCase
	matcher      *taskMatcher
	l            pkgLog.Logger
}

func New(
	memosRepo repository.MemosRepository,
	vectorRepo repository.VectorRepository,
	checklistSvc checklist.UseCase,
	l pkgLog.Logger,
) automation.UseCase {
	matcher := &taskMatcher{
		memosRepo:  memosRepo,
		vectorRepo: vectorRepo,
		l:          l,
	}

	return &implUseCase{
		memosRepo:    memosRepo,
		vectorRepo:   vectorRepo,
		checklistSvc: checklistSvc,
		matcher:      matcher,
		l:            l,
	}
}
