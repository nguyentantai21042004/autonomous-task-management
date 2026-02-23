package gemini

// GenerateRequest is the top-level request body for Gemini API.
type GenerateRequest struct {
	Contents         []Content         `json:"contents"`
	GenerationConfig *GenerationConfig `json:"generationConfig,omitempty"`
}

// Content wraps a list of Part objects to form a message.
type Content struct {
	Parts []Part `json:"parts"`
}

// Part holds a text segment for a content message.
type Part struct {
	Text string `json:"text"`
}

// GenerationConfig holds optional generation settings.
type GenerationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}

// GenerateResponse is the top-level response body from Gemini API.
type GenerateResponse struct {
	Candidates []Candidate `json:"candidates"`
}

// Candidate represents a single response candidate.
type Candidate struct {
	Content Content `json:"content"`
}

// ParsedTask is a task extracted from user input by the LLM.
type ParsedTask struct {
	Title                    string   `json:"title"`
	Description              string   `json:"description"`
	DueDateAbsolute          string   `json:"due_date_absolute"`
	Priority                 string   `json:"priority"`
	Tags                     []string `json:"tags"`
	EstimatedDurationMinutes int      `json:"estimated_duration_minutes"`
}
