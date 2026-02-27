package llmprovider

import (
	"context"
	"encoding/json"
	"fmt"

	"autonomous-task-management/pkg/deepseek"
	"autonomous-task-management/pkg/gemini"
	"autonomous-task-management/pkg/qwen"
)

// GeminiAdapter adapts pkg/gemini to llmprovider.Provider interface
type GeminiAdapter struct {
	client gemini.IGemini
}

// NewGeminiAdapter creates a new Gemini adapter
func NewGeminiAdapter(client gemini.IGemini) *GeminiAdapter {
	return &GeminiAdapter{client: client}
}

// GenerateContent implements Provider interface
func (a *GeminiAdapter) GenerateContent(ctx context.Context, req *Request) (*Response, error) {
	geminiReq := &gemini.Request{
		SystemInstruction: convertToGeminiContent(req.SystemInstruction),
		Messages:          convertToGeminiContents(req.Messages),
		Tools:             convertToGeminiTools(req.Tools),
		Temperature:       req.Temperature,
		MaxTokens:         req.MaxTokens,
	}

	resp, err := a.client.GenerateContent(ctx, geminiReq)
	if err != nil {
		return nil, err
	}

	return &Response{
		Content:      convertFromGeminiContent(resp.Content),
		ProviderName: "gemini",
		ModelName:    a.client.Model(),
		Usage: &Usage{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		},
	}, nil
}

// Name returns provider name
func (a *GeminiAdapter) Name() string {
	return "gemini"
}

// Model returns model name
func (a *GeminiAdapter) Model() string {
	return a.client.Model()
}

// QwenAdapter adapts pkg/qwen to llmprovider.Provider interface
type QwenAdapter struct {
	client qwen.IQwen
}

// NewQwenAdapter creates a new Qwen adapter
func NewQwenAdapter(client qwen.IQwen) *QwenAdapter {
	return &QwenAdapter{client: client}
}

// GenerateContent implements Provider interface
func (a *QwenAdapter) GenerateContent(ctx context.Context, req *Request) (*Response, error) {
	qwenReq := &qwen.Request{
		SystemInstruction: convertToQwenContent(req.SystemInstruction),
		Messages:          convertToQwenContents(req.Messages),
		Tools:             convertToQwenTools(req.Tools),
		Temperature:       req.Temperature,
		MaxTokens:         req.MaxTokens,
	}

	resp, err := a.client.GenerateContent(ctx, qwenReq)
	if err != nil {
		return nil, err
	}

	return &Response{
		Content:      convertFromQwenContent(resp.Content),
		ProviderName: "qwen",
		ModelName:    a.client.Model(),
		Usage: &Usage{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		},
	}, nil
}

// Name returns provider name
func (a *QwenAdapter) Name() string {
	return "qwen"
}

// Model returns model name
func (a *QwenAdapter) Model() string {
	return a.client.Model()
}

// Conversion helpers for Gemini
func convertToGeminiContent(msg *Message) *gemini.Content {
	if msg == nil {
		return nil
	}
	parts := make([]gemini.Part, len(msg.Parts))
	for i, p := range msg.Parts {
		parts[i] = gemini.Part{Text: p.Text}
		if p.FunctionCall != nil {
			parts[i].FunctionCall = &gemini.FunctionCall{
				Name: p.FunctionCall.Name,
				Args: p.FunctionCall.Args,
			}
		}
		if p.FunctionResponse != nil {
			parts[i].FunctionResponse = &gemini.FunctionResponse{
				Name:     p.FunctionResponse.Name,
				Response: p.FunctionResponse.Response,
			}
		}
	}
	return &gemini.Content{Role: msg.Role, Parts: parts}
}

func convertToGeminiContents(msgs []Message) []gemini.Content {
	contents := make([]gemini.Content, len(msgs))
	for i, msg := range msgs {
		contents[i] = *convertToGeminiContent(&msg)
	}
	return contents
}

func convertToGeminiTools(tools []Tool) []gemini.Tool {
	geminiTools := make([]gemini.Tool, len(tools))
	for i, t := range tools {
		geminiTools[i] = gemini.Tool{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  t.Parameters,
		}
	}
	return geminiTools
}

func convertFromGeminiContent(content gemini.Content) Message {
	parts := make([]Part, len(content.Parts))
	for i, p := range content.Parts {
		parts[i] = Part{Text: p.Text}
		if p.FunctionCall != nil {
			parts[i].FunctionCall = &FunctionCall{
				Name: p.FunctionCall.Name,
				Args: p.FunctionCall.Args,
			}
		}
		if p.FunctionResponse != nil {
			parts[i].FunctionResponse = &FunctionResponse{
				Name:     p.FunctionResponse.Name,
				Response: p.FunctionResponse.Response,
			}
		}
	}
	return Message{Role: content.Role, Parts: parts}
}

// Conversion helpers for Qwen
func convertToQwenContent(msg *Message) *qwen.Content {
	if msg == nil {
		return nil
	}
	parts := make([]qwen.Part, len(msg.Parts))
	for i, p := range msg.Parts {
		parts[i] = qwen.Part{Text: p.Text}
		if p.FunctionCall != nil {
			parts[i].FunctionCall = &qwen.FunctionCall{
				Name: p.FunctionCall.Name,
				Args: p.FunctionCall.Args,
			}
		}
		if p.FunctionResponse != nil {
			parts[i].FunctionResponse = &qwen.FunctionResponse{
				Name:     p.FunctionResponse.Name,
				Response: p.FunctionResponse.Response,
			}
		}
	}
	return &qwen.Content{Role: msg.Role, Parts: parts}
}

func convertToQwenContents(msgs []Message) []qwen.Content {
	contents := make([]qwen.Content, len(msgs))
	for i, msg := range msgs {
		contents[i] = *convertToQwenContent(&msg)
	}
	return contents
}

func convertToQwenTools(tools []Tool) []qwen.Tool {
	qwenTools := make([]qwen.Tool, len(tools))
	for i, t := range tools {
		qwenTools[i] = qwen.Tool{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  t.Parameters,
		}
	}
	return qwenTools
}

func convertFromQwenContent(content qwen.Content) Message {
	parts := make([]Part, len(content.Parts))
	for i, p := range content.Parts {
		parts[i] = Part{Text: p.Text}
		if p.FunctionCall != nil {
			parts[i].FunctionCall = &FunctionCall{
				Name: p.FunctionCall.Name,
				Args: p.FunctionCall.Args,
			}
		}
		if p.FunctionResponse != nil {
			parts[i].FunctionResponse = &FunctionResponse{
				Name:     p.FunctionResponse.Name,
				Response: p.FunctionResponse.Response,
			}
		}
	}
	return Message{Role: content.Role, Parts: parts}
}

// DeepSeekAdapter adapts pkg/deepseek to llmprovider.Provider interface
type DeepSeekAdapter struct {
	client deepseek.IDeepSeek
}

// NewDeepSeekAdapter creates a new DeepSeek adapter
func NewDeepSeekAdapter(client deepseek.IDeepSeek) *DeepSeekAdapter {
	return &DeepSeekAdapter{client: client}
}

// GenerateContent implements Provider interface
func (a *DeepSeekAdapter) GenerateContent(ctx context.Context, req *Request) (*Response, error) {
	deepseekReq := &deepseek.Request{
		Messages: convertToDeepSeekMessages(req.Messages),
	}

	// Add system instruction as first message if present
	if req.SystemInstruction != nil && len(req.SystemInstruction.Parts) > 0 {
		systemMsg := deepseek.Message{
			Role:    "system",
			Content: req.SystemInstruction.Parts[0].Text,
		}
		deepseekReq.Messages = append([]deepseek.Message{systemMsg}, deepseekReq.Messages...)
	}

	// Add tools if present
	if len(req.Tools) > 0 {
		deepseekReq.Tools = convertToDeepSeekTools(req.Tools)
	}

	resp, err := a.client.GenerateContent(ctx, deepseekReq)
	if err != nil {
		return nil, fmt.Errorf("deepseek: %w", err)
	}

	return convertFromDeepSeekResponse(resp), nil
}

// Name returns the provider name
func (a *DeepSeekAdapter) Name() string {
	return "deepseek"
}

// Model returns the model name
func (a *DeepSeekAdapter) Model() string {
	return "deepseek-chat"
}

// Conversion helpers for DeepSeek
func convertToDeepSeekMessages(msgs []Message) []deepseek.Message {
	messages := make([]deepseek.Message, 0, len(msgs))
	for _, msg := range msgs {
		dsMsg := deepseek.Message{
			Role: msg.Role,
		}

		// Handle text content
		if len(msg.Parts) > 0 && msg.Parts[0].Text != "" {
			dsMsg.Content = msg.Parts[0].Text
		}

		// Handle function calls
		if len(msg.Parts) > 0 && msg.Parts[0].FunctionCall != nil {
			fc := msg.Parts[0].FunctionCall
			argsJSON, _ := json.Marshal(fc.Args)
			dsMsg.ToolCalls = []deepseek.ToolCall{
				{
					ID:   "call_" + fc.Name,
					Type: "function",
					Function: deepseek.FunctionCall{
						Name:      fc.Name,
						Arguments: string(argsJSON),
					},
				},
			}
		}

		// Handle function responses
		if len(msg.Parts) > 0 && msg.Parts[0].FunctionResponse != nil {
			fr := msg.Parts[0].FunctionResponse
			dsMsg.Role = "tool"
			dsMsg.ToolCallID = "call_" + fr.Name
			dsMsg.Name = fr.Name
			responseJSON, _ := json.Marshal(fr.Response)
			dsMsg.Content = string(responseJSON)
		}

		messages = append(messages, dsMsg)
	}
	return messages
}

func convertToDeepSeekTools(tools []Tool) []deepseek.Tool {
	dsTools := make([]deepseek.Tool, len(tools))
	for i, t := range tools {
		dsTools[i] = deepseek.Tool{
			Type: "function",
			Function: deepseek.FunctionDef{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		}
	}
	return dsTools
}

func convertFromDeepSeekResponse(resp *deepseek.Response) *Response {
	if len(resp.Choices) == 0 {
		return &Response{
			Content: Message{
				Role:  "assistant",
				Parts: []Part{},
			},
			ProviderName: "deepseek",
			ModelName:    resp.Model,
			Usage: &Usage{
				InputTokens:  resp.Usage.PromptTokens,
				OutputTokens: resp.Usage.CompletionTokens,
				TotalTokens:  resp.Usage.TotalTokens,
			},
		}
	}

	choice := resp.Choices[0]
	parts := []Part{}

	// Handle text content
	if choice.Message.Content != "" {
		parts = append(parts, Part{Text: choice.Message.Content})
	}

	// Handle function calls
	if len(choice.Message.ToolCalls) > 0 {
		tc := choice.Message.ToolCalls[0]
		var args map[string]interface{}
		json.Unmarshal([]byte(tc.Function.Arguments), &args)
		parts = append(parts, Part{
			FunctionCall: &FunctionCall{
				Name: tc.Function.Name,
				Args: args,
			},
		})
	}

	return &Response{
		Content: Message{
			Role:  "assistant",
			Parts: parts,
		},
		ProviderName: "deepseek",
		ModelName:    resp.Model,
		Usage: &Usage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		},
	}
}
