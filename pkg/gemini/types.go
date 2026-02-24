package gemini

// GenerateRequest is the top-level request body for Gemini API.
type GenerateRequest struct {
	SystemInstruction *Content          `json:"system_instruction,omitempty"`
	Contents         []Content         `json:"contents"`
	Tools            []Tool            `json:"tools,omitempty"` // Added for function calling
	GenerationConfig *GenerationConfig `json:"generationConfig,omitempty"`
}

// Tool represents a collection of function declarations.
type Tool struct {
	FunctionDeclarations []FunctionDeclaration `json:"functionDeclarations,omitempty"`
}

// FunctionDeclaration defines a function that the model can call.
type FunctionDeclaration struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"` // JSON Schema format
}

// Content wraps a list of Part objects to form a message.
type Content struct {
	Role  string `json:"role,omitempty"` // Added for multi-turn conversations
	Parts []Part `json:"parts"`
}

// Part holds a text segment or a function call for a content message.
type Part struct {
	Text             string            `json:"text,omitempty"`
	FunctionCall     *FunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *FunctionResponse `json:"functionResponse,omitempty"`
}

// FunctionCall represents a model's request to call a function.
type FunctionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

// FunctionResponse represents the result of a function call executed by the client.
type FunctionResponse struct {
	Name     string      `json:"name"`
	Response interface{} `json:"response"`
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
