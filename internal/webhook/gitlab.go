package webhook

import (
	"encoding/json"
	"fmt"
	"time"

	"autonomous-task-management/internal/model"
)

// GitLabWebhookParser parses GitLab webhook payloads
type GitLabWebhookParser struct{}

func NewGitLabParser() *GitLabWebhookParser {
	return &GitLabWebhookParser{}
}

// ParsePushEvent parses GitLab push event
func (p *GitLabWebhookParser) ParsePushEvent(payload []byte) (*model.WebhookEvent, error) {
	var event struct {
		ObjectKind string `json:"object_kind"`
		Ref        string `json:"ref"`
		Project    struct {
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
		return nil, fmt.Errorf("failed to parse push event: %w", err)
	}

	// Extract branch name from ref
	branch := event.Ref
	if len(branch) > 11 && branch[:11] == "refs/heads/" {
		branch = branch[11:]
	}

	// Get last commit
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

// ParseMergeRequestEvent parses GitLab merge request event
func (p *GitLabWebhookParser) ParseMergeRequestEvent(payload []byte) (*model.WebhookEvent, error) {
	var event struct {
		ObjectKind       string `json:"object_kind"`
		ObjectAttributes struct {
			IID          int    `json:"iid"` // MR number
			Title        string `json:"title"`
			State        string `json:"state"` // opened, closed, merged
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
		return nil, fmt.Errorf("failed to parse merge request event: %w", err)
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

// ParseIssueEvent parses GitLab issue event
func (p *GitLabWebhookParser) ParseIssueEvent(payload []byte) (*model.WebhookEvent, error) {
	var event struct {
		ObjectKind       string `json:"object_kind"`
		ObjectAttributes struct {
			IID    int    `json:"iid"` // Issue number
			Title  string `json:"title"`
			State  string `json:"state"`
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
		return nil, fmt.Errorf("failed to parse issue event: %w", err)
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
