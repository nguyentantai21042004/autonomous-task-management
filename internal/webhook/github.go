package webhook

import (
	"encoding/json"
	"fmt"
	"time"

	"autonomous-task-management/internal/model"
)

// GitHubWebhookParser parses GitHub webhook payloads
type GitHubWebhookParser struct{}

func NewGitHubParser() *GitHubWebhookParser {
	return &GitHubWebhookParser{}
}

// ParsePushEvent parses GitHub push event
func (p *GitHubWebhookParser) ParsePushEvent(payload []byte) (*model.WebhookEvent, error) {
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
		return nil, fmt.Errorf("failed to parse push event: %w", err)
	}

	// Extract branch name from ref (refs/heads/main → main)
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

// ParsePullRequestEvent parses GitHub pull request event
func (p *GitHubWebhookParser) ParsePullRequestEvent(payload []byte) (*model.WebhookEvent, error) {
	var event struct {
		Action      string `json:"action"` // opened, closed, merged, etc.
		Number      int    `json:"number"`
		PullRequest struct {
			Title string `json:"title"`
			Head  struct {
				Ref string `json:"ref"` // Branch name
				SHA string `json:"sha"` // Commit SHA
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
		return nil, fmt.Errorf("failed to parse pull request event: %w", err)
	}

	// Determine action (merged takes precedence over closed)
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

// ParseIssueEvent parses GitHub issue event
func (p *GitHubWebhookParser) ParseIssueEvent(payload []byte) (*model.WebhookEvent, error) {
	var event struct {
		Action string `json:"action"` // opened, closed, etc.
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
		return nil, fmt.Errorf("failed to parse issue event: %w", err)
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
