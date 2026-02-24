package model

import "time"

// WebhookSource represents the source platform
type WebhookSource string

const (
	SourceGitHub WebhookSource = "github"
	SourceGitLab WebhookSource = "gitlab"
	SourceManual WebhookSource = "manual"
)

// WebhookEvent represents a parsed webhook event
type WebhookEvent struct {
	Source      WebhookSource          // Platform source
	EventType   string                 // Event type (push, pull_request, etc.)
	Repository  string                 // Repository name
	Branch      string                 // Branch name
	Commit      string                 // Commit SHA
	Author      string                 // Event author
	Message     string                 // Commit/PR message
	PRNumber    int                    // PR number (if applicable)
	IssueNumber int                    // Issue number (if applicable)
	Action      string                 // Action (opened, closed, merged, etc.)
	Metadata    map[string]interface{} // Additional metadata
	ReceivedAt  time.Time              // When webhook was received
}
