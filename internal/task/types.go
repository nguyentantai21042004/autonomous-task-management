package task

import "autonomous-task-management/internal/task/repository"

// CreateBulkInput is the input for bulk task creation.
// UserID is stored in models.Scope, not here (per convention fixes).
type CreateBulkInput struct {
	RawText        string // Natural language task descriptions from the user
	TelegramChatID int64  // Used to send response back to user
}

// CreatedTask represents a single task that was successfully created.
type CreatedTask struct {
	MemoID       string // Memos internal ID
	MemoURL      string // Deep link to the memo
	CalendarLink string // Deep link to the Google Calendar event (may be empty)
	Title        string
}

// SearchInput is the input for semantic search.
type SearchInput struct {
	Query string   `json:"query"` // Natural language query
	Limit int      `json:"limit"` // Max results (default 10)
	Tags  []string `json:"tags"`  // Filter by tags (optional)
}

// SearchResultItem represents a single search result.
type SearchResultItem struct {
	MemoID  string  `json:"memo_id"`
	MemoURL string  `json:"memo_url"`
	Content string  `json:"content"`
	Score   float64 `json:"score"` // Similarity score (0-1)
}

// SearchOutput is the result of semantic search.
type SearchOutput struct {
	Results []SearchResultItem `json:"results"`
	Count   int                `json:"count"`
}

// CreateBulkOutput is the result of the bulk task creation operation.
type CreateBulkOutput struct {
	Tasks     []CreatedTask
	TaskCount int
}

// QueryInput is the input for RAG-based question answering.
type QueryInput struct {
	Query string // Natural language question
}

// QueryOutput is the result of RAG-based question answering.
type QueryOutput struct {
	Answer      string                    // LLM-generated answer
	SourceTasks []repository.SearchResult // Source tasks used
	SourceCount int                       // Number of sources
}
