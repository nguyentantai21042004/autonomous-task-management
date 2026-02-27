package gemini

import (
	"fmt"
	"net/http"
)

// Config holds Gemini client configuration
type Config struct {
	APIKey     string
	Model      string
	APIURL     string
	HTTPClient *http.Client
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("gemini: APIKey is required")
	}
	if c.Model == "" {
		c.Model = DefaultModel
	}
	if c.APIURL == "" {
		c.APIURL = DefaultAPIURL
	}
	if c.HTTPClient == nil {
		c.HTTPClient = &http.Client{Timeout: DefaultTimeout}
	}
	return nil
}

// geminiImpl is the internal implementation of IGemini
type geminiImpl struct {
	apiKey     string
	model      string
	apiURL     string
	httpClient *http.Client
}

// Request represents a Gemini generation request
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

// Response represents a Gemini generation response
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

// Internal Gemini API types
type geminiRequest struct {
	SystemInstruction *geminiContent          `json:"system_instruction,omitempty"`
	Contents          []geminiContent         `json:"contents"`
	Tools             []geminiTool            `json:"tools,omitempty"`
	GenerationConfig  *geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiTool struct {
	FunctionDeclarations []geminiFunctionDeclaration `json:"functionDeclarations,omitempty"`
}

type geminiFunctionDeclaration struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text             string                  `json:"text,omitempty"`
	FunctionCall     *geminiFunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *geminiFunctionResponse `json:"functionResponse,omitempty"`
}

type geminiFunctionCall struct {
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

type geminiFunctionResponse struct {
	Name     string      `json:"name"`
	Response interface{} `json:"response"`
}

type geminiGenerationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}

type geminiResponse struct {
	Candidates []geminiCandidate `json:"candidates"`
}

type geminiCandidate struct {
	Content geminiContent `json:"content"`
}
