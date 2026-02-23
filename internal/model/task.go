package model

// Task represents a task stored in Memos.
type Task struct {
	ID         string   // Memos internal ID (name field, e.g. "memos/123")
	UID        string   // Memos short UID
	Content    string   // Full Markdown content
	Tags       []string // Extracted tags
	MemoURL    string   // Deep link to the Memos web UI
	Visibility string   // "PRIVATE" or "PUBLIC"
	CreateTime string   // RFC3339 creation time string from Memos API
	UpdateTime string   // RFC3339 last updated time string from Memos API
}
