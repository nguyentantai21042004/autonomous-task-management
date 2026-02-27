package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// newGeminiImpl creates a new Gemini implementation
func newGeminiImpl(cfg Config) *geminiImpl {
	return &geminiImpl{
		apiKey:     cfg.APIKey,
		model:      cfg.Model,
		apiURL:     cfg.APIURL,
		httpClient: cfg.HTTPClient,
	}
}

// GenerateContent sends a generation request to Gemini API
func (g *geminiImpl) GenerateContent(ctx context.Context, req *Request) (*Response, error) {
	geminiReq := g.transformRequest(req)
	geminiResp, err := g.callAPI(ctx, geminiReq)
	if err != nil {
		return nil, err
	}
	return g.transformResponse(geminiResp), nil
}

// Model returns the model being used
func (g *geminiImpl) Model() string {
	return g.model
}

// callAPI sends a request to the Gemini API
func (g *geminiImpl) callAPI(ctx context.Context, req geminiRequest) (*geminiResponse, error) {
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", g.apiURL, g.model, g.apiKey)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("gemini: failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("gemini: failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gemini: failed to call API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gemini: API error %d: %s", resp.StatusCode, string(raw))
	}

	var result geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("gemini: failed to decode response: %w", err)
	}

	return &result, nil
}

// transformRequest converts request to Gemini API format
func (g *geminiImpl) transformRequest(req *Request) geminiRequest {
	geminiReq := geminiRequest{
		Contents: make([]geminiContent, len(req.Messages)),
	}

	if req.SystemInstruction != nil {
		geminiReq.SystemInstruction = &geminiContent{
			Parts: transformParts(req.SystemInstruction.Parts),
		}
	}

	for i, msg := range req.Messages {
		geminiReq.Contents[i] = geminiContent{
			Role:  msg.Role,
			Parts: transformParts(msg.Parts),
		}
	}

	if len(req.Tools) > 0 {
		functionDecls := make([]geminiFunctionDeclaration, len(req.Tools))
		for i, tool := range req.Tools {
			functionDecls[i] = geminiFunctionDeclaration{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			}
		}
		geminiReq.Tools = []geminiTool{{FunctionDeclarations: functionDecls}}
	}

	if req.Temperature > 0 || req.MaxTokens > 0 {
		geminiReq.GenerationConfig = &geminiGenerationConfig{
			Temperature:     req.Temperature,
			MaxOutputTokens: req.MaxTokens,
		}
	}

	return geminiReq
}

func transformParts(parts []Part) []geminiPart {
	geminiParts := make([]geminiPart, len(parts))
	for i, part := range parts {
		geminiParts[i] = geminiPart{Text: part.Text}
		if part.FunctionCall != nil {
			geminiParts[i].FunctionCall = &geminiFunctionCall{
				Name: part.FunctionCall.Name,
				Args: part.FunctionCall.Args,
			}
		}
		if part.FunctionResponse != nil {
			geminiParts[i].FunctionResponse = &geminiFunctionResponse{
				Name:     part.FunctionResponse.Name,
				Response: part.FunctionResponse.Response,
			}
		}
	}
	return geminiParts
}

// transformResponse converts Gemini API response to standard format
func (g *geminiImpl) transformResponse(resp *geminiResponse) *Response {
	if len(resp.Candidates) == 0 {
		return &Response{Usage: &Usage{}}
	}

	candidate := resp.Candidates[0]
	content := candidate.Content

	parts := make([]Part, len(content.Parts))
	for i, part := range content.Parts {
		parts[i] = Part{Text: part.Text}
		if part.FunctionCall != nil {
			parts[i].FunctionCall = &FunctionCall{
				Name: part.FunctionCall.Name,
				Args: part.FunctionCall.Args,
			}
		}
		if part.FunctionResponse != nil {
			parts[i].FunctionResponse = &FunctionResponse{
				Name:     part.FunctionResponse.Name,
				Response: part.FunctionResponse.Response,
			}
		}
	}

	return &Response{
		Content: Content{Role: content.Role, Parts: parts},
		Usage:   &Usage{},
	}
}
