package usecase

import (
	"autonomous-task-management/internal/example/repository"
	"autonomous-task-management/pkg/log"
)

// implUseCase is the private implementation of example.UseCase.
type implUseCase struct {
	repo repository.Repository
	l    log.Logger
}

// New creates a new example UseCase implementation.
func New(repo repository.Repository, l log.Logger) *implUseCase {
	return &implUseCase{
		repo: repo,
		l:    l,
	}
}
