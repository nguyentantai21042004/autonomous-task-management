package test

// TestMessageRequest represents a test message request
type TestMessageRequest struct {
	Text   string `json:"text" binding:"required"`
	UserID int64  `json:"user_id"`
}

// TestMessageResponse represents a test message response
type TestMessageResponse struct {
	Success    bool     `json:"success"`
	Intent     string   `json:"intent,omitempty"`
	Confidence int      `json:"confidence,omitempty"`
	Reasoning  string   `json:"reasoning,omitempty"`
	Text       string   `json:"text"`
	UserID     int64    `json:"user_id"`
	History    []string `json:"history,omitempty"`
	Error      string   `json:"error,omitempty"`
	Details    string   `json:"details,omitempty"`
}

// ResetSessionRequest represents a reset session request
type ResetSessionRequest struct {
	UserID int64 `json:"user_id"`
}

// ResetSessionResponse represents a reset session response
type ResetSessionResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	UserID  int64  `json:"user_id"`
}

// HealthCheckResponse represents a health check response
type HealthCheckResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}
