package automation

import (
	"autonomous-task-management/internal/checklist"
	"autonomous-task-management/internal/task/repository"
	pkgLog "autonomous-task-management/pkg/log"
)

func New(
	memosRepo repository.MemosRepository,
	vectorRepo repository.VectorRepository,
	checklistSvc checklist.Service,
	l pkgLog.Logger,
) UseCase {
	matcher := NewTaskMatcher(memosRepo, vectorRepo, l)

	return &usecase{
		memosRepo:    memosRepo,
		vectorRepo:   vectorRepo,
		checklistSvc: checklistSvc,
		matcher:      matcher,
		l:            l,
	}
}
