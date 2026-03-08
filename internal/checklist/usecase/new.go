package usecase

import (
	"regexp"

	"autonomous-task-management/internal/checklist"
	"autonomous-task-management/internal/task/repository"
	pkgLog "autonomous-task-management/pkg/log"
)

const (
	CheckboxUnchecked = `- [ ]`
	CheckboxChecked   = `- [x]`
	// Regex pattern: captures indent, checkbox state, and text
	// Example: "  - [x] Task name" → groups: ["  ", "x", "Task name"]
	CheckboxPattern = `(?m)^(\s*)- \[([ xX])\] (.+)$`
)

type implUseCase struct {
	pattern    *regexp.Regexp
	memosRepo  repository.MemosRepository
	vectorRepo repository.VectorRepository
	l          pkgLog.Logger
}

func New(memosRepo repository.MemosRepository, vectorRepo repository.VectorRepository, l pkgLog.Logger) checklist.UseCase {
	return &implUseCase{
		pattern:    regexp.MustCompile(CheckboxPattern),
		memosRepo:  memosRepo,
		vectorRepo: vectorRepo,
		l:          l,
	}
}
