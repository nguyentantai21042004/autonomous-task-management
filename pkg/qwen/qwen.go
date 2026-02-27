package qwen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// newQwenImpl creates a new Qwen implementation
func newQwenImpl(cfg Config) *qwenImpl {
	return &qwenImpl{
		apiKey:     cfg.APIKey,
		baseURL:    cfg.BaseURL,
		model:      cfg.Model,
		httpClient: cfg.HTTPClient,
	}
}

// GenerateContent sends a generation request to Qwen API
func (q *qwenImpl) GenerateContent(ctx context.Context, req *Request) (*Response, error) {
	openAIReq := q.transformRequest(req)

	body, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, fmt.Errorf("qwen: failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		q.baseURL+"/chat/completions", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("qwen: failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+q.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := q.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("qwen: API call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("qwen: API error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var openAIResp openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		return nil, fmt.Errorf("qwen: failed to decode response: %w", err)
	}

	return q.transformResponse(&openAIResp), nil
}

// Model returns the model being used
func (q *qwenImpl) Model() string {
	return q.model
}

// transformRequest converts request to OpenAI-compatible format
func (q *qwenImpl) transformRequest(req *Request) *openAIRequest {
	openAIReq := &openAIRequest{
		Model:       q.model,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Messages:    make([]openAIMessage, 0),
	}

	if req.SystemInstruction != nil {
		systemMsg := q.transformMessage(req.SystemInstruction)
		systemMsg.Role = "system"
		openAIReq.Messages = append(openAIReq.Messages, systemMsg)
	}

	for _, msg := range req.Messages {
		openAIReq.Messages = append(openAIReq.Messages, q.transformMessage(&msg))
	}

	if len(req.Tools) > 0 {
		openAIReq.Tools = make([]openAITool, len(req.Tools))
		for i, tool := range req.Tools {
			openAIReq.Tools[i] = openAITool{
				Type: "function",
				Function: openAIFunctionDecl{
					Name:        tool.Name,
					Description: tool.Description,
					Parameters:  tool.Parameters,
				},
			}
		}
	}

	return openAIReq
}

func (q *qwenImpl) transformMessage(msg *Content) openAIMessage {
	openAIMsg := openAIMessage{Role: msg.Role}

	for _, part := range msg.Parts {
		if part.Text != "" {
			if openAIMsg.Content != "" {
				openAIMsg.Content += "\n"
			}
			openAIMsg.Content += part.Text
		}

		if part.FunctionCall != nil {
			argsJSON, _ := json.Marshal(part.FunctionCall.Args)
			toolCall := openAIToolCall{
				ID:   "call_" + part.FunctionCall.Name,
				Type: "function",
				Function: openAIFunctionCall{
					Name:      part.FunctionCall.Name,
					Arguments: string(argsJSON),
				},
			}
			openAIMsg.ToolCalls = append(openAIMsg.ToolCalls, toolCall)
		}

		if part.FunctionResponse != nil {
			openAIMsg.Role = "tool"
			openAIMsg.ToolCallID = "call_" + part.FunctionResponse.Name
			responseJSON, _ := json.Marshal(part.FunctionResponse.Response)
			openAIMsg.Content = string(responseJSON)
		}
	}

	return openAIMsg
}

func (q *qwenImpl) transformResponse(resp *openAIResponse) *Response {
	if resp == nil || len(resp.Choices) == 0 {
		return &Response{Usage: &Usage{}}
	}

	choice := resp.Choices[0]
	message := Content{
		Role:  choice.Message.Role,
		Parts: make([]Part, 0),
	}

	if choice.Message.Content != "" {
		message.Parts = append(message.Parts, Part{Text: choice.Message.Content})
	}

	for _, toolCall := range choice.Message.ToolCalls {
		if toolCall.Type == "function" {
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
				args = make(map[string]interface{})
			}

			message.Parts = append(message.Parts, Part{
				FunctionCall: &FunctionCall{
					Name: toolCall.Function.Name,
					Args: args,
				},
			})
		}
	}

	usage := &Usage{
		InputTokens:  resp.Usage.PromptTokens,
		OutputTokens: resp.Usage.CompletionTokens,
		TotalTokens:  resp.Usage.TotalTokens,
	}

	return &Response{
		Content: message,
		Usage:   usage,
	}
}
