package llmprovider

import "time"

// Request represents a normalized LLM generation request
type Request struct {
	SystemInstruction *Message
	Messages          []Message
	Tools             []Tool
	Temperature       float64
	MaxTokens         int
}

// Message represents a conversation message
type Message struct {
	Role  string // "user", "assistant", "system"
	Parts []Part
}

// Part represents a message part (text or function call)
type Part struct {
	Text             string
	FunctionCall     *FunctionCall
	FunctionResponse *FunctionResponse
}

// Tool represents a function declaration
type Tool struct {
	Name        string
	Description string
	Parameters  map[string]interface{} // JSON Schema
}

// FunctionCall represents a model's function call request
type FunctionCall struct {
	Name string
	Args map[string]interface{}
}

// FunctionResponse represents a function execution result
type FunctionResponse struct {
	Name     string
	Response interface{}
}

// Response represents a normalized LLM generation response
type Response struct {
	Content      Message
	ProviderName string
	ModelName    string
	Usage        *Usage
}

// Usage tracks token consumption
type Usage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

// Config defines configuration for the Provider Manager
type Config struct {
	FallbackEnabled bool
	RetryAttempts   int
	RetryDelay      time.Duration
	MaxTotalTimeout time.Duration // Global timeout for entire fallback chain
}
