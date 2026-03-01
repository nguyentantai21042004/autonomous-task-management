package usecase

import (
	"regexp"

	"autonomous-task-management/internal/checklist"
)

const (
	CheckboxUnchecked = `- [ ]`
	CheckboxChecked   = `- [x]`
	// Regex pattern: captures indent, checkbox state, and text
	// Example: "  - [x] Task name" → groups: ["  ", "x", "Task name"]
	CheckboxPattern = `(?m)^(\s*)- \[([ xX])\] (.+)$`
)

type implUseCase struct {
	pattern *regexp.Regexp
}

func New() checklist.UseCase {
	return &implUseCase{
		pattern: regexp.MustCompile(CheckboxPattern),
	}
}
