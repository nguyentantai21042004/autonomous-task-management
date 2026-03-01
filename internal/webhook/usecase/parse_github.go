package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"autonomous-task-management/internal/model"
	"autonomous-task-management/internal/webhook"
)

func (uc *implUseCase) ParseGitHubEvent(ctx context.Context, payload []byte, eventType string, signature string) (*model.WebhookEvent, error) {
	// Security check
	if err := uc.validateGitHubSignature(payload, signature); err != nil {
		return nil, err
	}
	if err := uc.checkRateLimit("github"); err != nil {
		return nil, err
	}

	// Parse payload
	var event *model.WebhookEvent
	var err error

	switch eventType {
	case "push":
		event, err = uc.parseGitHubPushEvent(payload)
	case "pull_request":
		event, err = uc.parseGitHubPullRequestEvent(payload)
	case "issues":
		event, err = uc.parseGitHubIssueEvent(payload)
	default:
		return nil, webhook.ErrUnsupportedEvent
	}

	if err != nil {
		return nil, fmt.Errorf("failed to parse GitHub event: %w", err)
	}

	return event, nil
}

func (uc *implUseCase) parseGitHubPushEvent(payload []byte) (*model.WebhookEvent, error) {
	var event struct {
		Ref        string `json:"ref"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
		HeadCommit struct {
			ID      string `json:"id"`
			Message string `json:"message"`
			Author  struct {
				Name string `json:"name"`
			} `json:"author"`
		} `json:"head_commit"`
	}

	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}

	branch := event.Ref
	if len(branch) > 11 && branch[:11] == "refs/heads/" {
		branch = branch[11:]
	}

	return &model.WebhookEvent{
		Source:     model.SourceGitHub,
		EventType:  "push",
		Repository: event.Repository.FullName,
		Branch:     branch,
		Commit:     event.HeadCommit.ID,
		Author:     event.HeadCommit.Author.Name,
		Message:    event.HeadCommit.Message,
		ReceivedAt: time.Now(),
	}, nil
}

func (uc *implUseCase) parseGitHubPullRequestEvent(payload []byte) (*model.WebhookEvent, error) {
	var event struct {
		Action      string `json:"action"`
		Number      int    `json:"number"`
		PullRequest struct {
			Title string `json:"title"`
			Head  struct {
				Ref string `json:"ref"`
				SHA string `json:"sha"`
			} `json:"head"`
			User struct {
				Login string `json:"login"`
			} `json:"user"`
			Merged bool `json:"merged"`
		} `json:"pull_request"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	}

	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}

	action := event.Action
	if action == "closed" && event.PullRequest.Merged {
		action = "merged"
	}

	return &model.WebhookEvent{
		Source:     model.SourceGitHub,
		EventType:  "pull_request",
		Repository: event.Repository.FullName,
		Branch:     event.PullRequest.Head.Ref,
		Commit:     event.PullRequest.Head.SHA,
		Author:     event.PullRequest.User.Login,
		Message:    event.PullRequest.Title,
		PRNumber:   event.Number,
		Action:     action,
		ReceivedAt: time.Now(),
	}, nil
}

func (uc *implUseCase) parseGitHubIssueEvent(payload []byte) (*model.WebhookEvent, error) {
	var event struct {
		Action string `json:"action"`
		Issue  struct {
			Number int    `json:"number"`
			Title  string `json:"title"`
			User   struct {
				Login string `json:"login"`
			} `json:"user"`
		} `json:"issue"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	}

	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}

	return &model.WebhookEvent{
		Source:      model.SourceGitHub,
		EventType:   "issue",
		Repository:  event.Repository.FullName,
		Author:      event.Issue.User.Login,
		Message:     event.Issue.Title,
		IssueNumber: event.Issue.Number,
		Action:      event.Action,
		ReceivedAt:  time.Now(),
	}, nil
}
