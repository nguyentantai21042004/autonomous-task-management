package qwen

import (
	"fmt"
	"net/http"
)

// Config holds Qwen client configuration
type Config struct {
	APIKey     string
	Model      string
	BaseURL    string
	HTTPClient *http.Client
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("qwen: APIKey is required")
	}
	if c.Model == "" {
		c.Model = DefaultModel
	}
	if c.BaseURL == "" {
		c.BaseURL = DefaultBaseURL
	}
	if c.HTTPClient == nil {
		c.HTTPClient = &http.Client{Timeout: DefaultTimeout}
	}
	return nil
}

// qwenImpl is the internal implementation of IQwen
type qwenImpl struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
}

// Request represents a Qwen generation request
type Request struct {
	SystemInstruction *Content
	Messages          []Content
	Tools             []Tool
	Temperature       float64
	MaxTokens         int
}

// Content represents a message content
type Content struct {
	Role  string
	Parts []Part
}

// Part represents a message part
type Part struct {
	Text             string
	FunctionCall     *FunctionCall
	FunctionResponse *FunctionResponse
}

// Tool represents a function declaration
type Tool struct {
	Name        string
	Description string
	Parameters  map[string]interface{}
}

// FunctionCall represents a function call request
type FunctionCall struct {
	Name string
	Args map[string]interface{}
}

// FunctionResponse represents a function execution result
type FunctionResponse struct {
	Name     string
	Response interface{}
}

// Response represents a Qwen generation response
type Response struct {
	Content Content
	Usage   *Usage
}

// Usage tracks token consumption
type Usage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

// OpenAI-compatible types for Qwen API
type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Tools       []openAITool    `json:"tools,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openAITool struct {
	Type     string             `json:"type"`
	Function openAIFunctionDecl `json:"function"`
}

type openAIFunctionDecl struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type openAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function openAIFunctionCall `json:"function"`
}

type openAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAIResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []openAIChoice `json:"choices"`
	Usage   openAIUsage    `json:"usage"`
}

type openAIChoice struct {
	Index        int           `json:"index"`
	Message      openAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
