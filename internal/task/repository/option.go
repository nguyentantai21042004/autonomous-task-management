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
