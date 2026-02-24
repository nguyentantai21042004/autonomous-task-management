package sync

// MemosWebhookPayload matches Memos API v1 webhook format.
type MemosWebhookPayload struct {
	ActivityType string `json:"activityType"` // e.g., "memos.memo.created"
	Memo         struct {
		Name string `json:"name"` // e.g., "memos/123"
		UID  string `json:"uid"`  // Short UID (Base58)
	} `json:"memo"`
}
