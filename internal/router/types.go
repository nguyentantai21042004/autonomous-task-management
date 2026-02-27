package router

// Intent represents user's intention
type Intent string

const (
	IntentCreateTask      Intent = "CREATE_TASK"
	IntentSearchTask      Intent = "SEARCH_TASK"
	IntentManageChecklist Intent = "MANAGE_CHECKLIST"
	IntentConversation    Intent = "CONVERSATION"
)

// RouterOutput is the structured response from Semantic Router
type RouterOutput struct {
	Intent     Intent `json:"intent"`
	Confidence int    `json:"confidence"` // 0-100
	Reasoning  string `json:"reasoning"`  // Optional: Why this intent was chosen
}
