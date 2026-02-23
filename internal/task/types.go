package task

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

// CreateBulkOutput is the result of the bulk task creation operation.
type CreateBulkOutput struct {
	Tasks     []CreatedTask
	TaskCount int
}
