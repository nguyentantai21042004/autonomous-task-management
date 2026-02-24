package webhook

import (
	"autonomous-task-management/internal/automation"
	pkgLog "autonomous-task-management/pkg/log"
)

type Handler struct {
	automationUC automation.UseCase
	security     *SecurityValidator
	githubParser *GitHubWebhookParser
	gitlabParser *GitLabWebhookParser
	l            pkgLog.Logger
}

func NewHandler(
	automationUC automation.UseCase,
	securityConfig SecurityConfig,
	l pkgLog.Logger,
) *Handler {
	return &Handler{
		automationUC: automationUC,
		security:     NewSecurityValidator(securityConfig),
		githubParser: NewGitHubParser(),
		gitlabParser: NewGitLabParser(),
		l:            l,
	}
}
