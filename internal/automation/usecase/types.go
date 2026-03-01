package usecase

// MatchCriteria defines how to match webhook event to tasks
type MatchCriteria struct {
	Repository string   // Repository name (e.g., "user/repo")
	Tags       []string // Tags to match (e.g., ["#repo/myproject", "#pr/123"])
	Keywords   []string // Keywords in content
}

// TaskMatch represents a matched task
type TaskMatch struct {
	TaskID      string  // Memos task ID
	Content     string  // Task content
	MatchScore  float64 // Match confidence (0-1)
	MatchReason string  // Why it matched
}
