package usecase

import (
	"autonomous-task-management/internal/router"
	"autonomous-task-management/pkg/llmprovider"
	pkgLog "autonomous-task-management/pkg/log"
)

type implUseCase struct {
	llm llmprovider.IManager
	l   pkgLog.Logger
}

func New(llm llmprovider.IManager, l pkgLog.Logger) router.UseCase {
	return &implUseCase{
		llm: llm,
		l:   l,
	}
}
