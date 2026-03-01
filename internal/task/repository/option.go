package repository

// CreateTaskOptions holds the parameters for creating a task in Memos.
type CreateTaskOptions struct {
	Content    string   // Full Markdown content body
	Tags       []string // Tag strings like "#domain/ahamove"
	Visibility string   // "PRIVATE" or "PUBLIC" (default: "PRIVATE")
}

// ListTasksOptions holds the parameters for listing tasks from Memos.
type ListTasksOptions struct {
	Tag    string // Filter by a specific tag
	Limit  int    // Max number of results (default 20)
	Offset int    // Pagination offset
}

// SearchTasksOptions defines search parameters.
type SearchTasksOptions struct {
	Query  string   // Natural language query
	Limit  int      // Top-K results
	Tags   []string // Filter by tags (optional)
	Filter PayloadFilter
}

// PayloadFilter represents Qdrant filter condition structure
type PayloadFilter struct {
	Should []Condition
}

type Condition struct {
	Key   string
	Match MatchAny
}

type MatchAny struct {
	Values []string
}

// SearchResult represents a semantic search result.
type SearchResult struct {
	MemoID  string
	Score   float64
	Payload map[string]interface{}
}
