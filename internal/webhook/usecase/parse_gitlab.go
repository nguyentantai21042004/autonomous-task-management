package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"autonomous-task-management/internal/model"
	"autonomous-task-management/internal/webhook"
)

func (uc *implUseCase) ParseGitLabEvent(ctx context.Context, payload []byte, eventType string, token string) (*model.WebhookEvent, error) {
	// Security check
	if err := uc.validateGitLabToken(token); err != nil {
		return nil, err
	}
	if err := uc.checkRateLimit("gitlab"); err != nil {
		return nil, err
	}

	// Parse payload
	var event *model.WebhookEvent
	var err error

	switch eventType {
	case "Push Hook":
		event, err = uc.parseGitLabPushEvent(payload)
	case "Merge Request Hook":
		event, err = uc.parseGitLabMergeRequestEvent(payload)
	case "Issue Hook":
		event, err = uc.parseGitLabIssueEvent(payload)
	default:
		return nil, webhook.ErrUnsupportedEvent
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse GitLab event: %w", err)
	}

	return event, nil
}

func (uc *implUseCase) parseGitLabPushEvent(payload []byte) (*model.WebhookEvent, error) {
	var event struct {
		Ref     string `json:"ref"`
		Project struct {
			PathWithNamespace string `json:"path_with_namespace"`
		} `json:"project"`
		Commits []struct {
			ID      string `json:"id"`
			Message string `json:"message"`
			Author  struct {
				Name string `json:"name"`
			} `json:"author"`
		} `json:"commits"`
	}

	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}

	branch := event.Ref
	if len(branch) > 11 && branch[:11] == "refs/heads/" {
		branch = branch[11:]
	}

	var commit, message, author string
	if len(event.Commits) > 0 {
		lastCommit := event.Commits[len(event.Commits)-1]
		commit = lastCommit.ID
		message = lastCommit.Message
		author = lastCommit.Author.Name
	}

	return &model.WebhookEvent{
		Source:     model.SourceGitLab,
		EventType:  "push",
		Repository: event.Project.PathWithNamespace,
		Branch:     branch,
		Commit:     commit,
		Author:     author,
		Message:    message,
		ReceivedAt: time.Now(),
	}, nil
}

func (uc *implUseCase) parseGitLabMergeRequestEvent(payload []byte) (*model.WebhookEvent, error) {
	var event struct {
		ObjectAttributes struct {
			IID          int    `json:"iid"`
			Title        string `json:"title"`
			Action       string `json:"action"`
			SourceBranch string `json:"source_branch"`
			LastCommit   struct {
				ID string `json:"id"`
			} `json:"last_commit"`
		} `json:"object_attributes"`
		User struct {
			Name string `json:"name"`
		} `json:"user"`
		Project struct {
			PathWithNamespace string `json:"path_with_namespace"`
		} `json:"project"`
	}

	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}

	return &model.WebhookEvent{
		Source:     model.SourceGitLab,
		EventType:  "merge_request",
		Repository: event.Project.PathWithNamespace,
		Branch:     event.ObjectAttributes.SourceBranch,
		Commit:     event.ObjectAttributes.LastCommit.ID,
		Author:     event.User.Name,
		Message:    event.ObjectAttributes.Title,
		PRNumber:   event.ObjectAttributes.IID,
		Action:     event.ObjectAttributes.Action,
		ReceivedAt: time.Now(),
	}, nil
}

func (uc *implUseCase) parseGitLabIssueEvent(payload []byte) (*model.WebhookEvent, error) {
	var event struct {
		ObjectAttributes struct {
			IID    int    `json:"iid"`
			Title  string `json:"title"`
			Action string `json:"action"`
		} `json:"object_attributes"`
		User struct {
			Name string `json:"name"`
		} `json:"user"`
		Project struct {
			PathWithNamespace string `json:"path_with_namespace"`
		} `json:"project"`
	}

	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}

	return &model.WebhookEvent{
		Source:      model.SourceGitLab,
		EventType:   "issue",
		Repository:  event.Project.PathWithNamespace,
		Author:      event.User.Name,
		Message:     event.ObjectAttributes.Title,
		IssueNumber: event.ObjectAttributes.IID,
		Action:      event.ObjectAttributes.Action,
		ReceivedAt:  time.Now(),
	}, nil
}
