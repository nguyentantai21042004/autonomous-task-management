package router

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"autonomous-task-management/pkg/gemini"
)

// Classify determines user intent from message
// Convention: Method accepts context.Context as first parameter
func (r *SemanticRouter) Classify(ctx context.Context, message string, conversationHistory []string) (RouterOutput, error) {
	// Build prompt with conversation history
	historyContext := ""
	if len(conversationHistory) > 0 {
		historyContext = PromptHistoryPrefix
		for i, msg := range conversationHistory {
			historyContext += fmt.Sprintf("%d. %s\n", i+1, msg)
		}
		historyContext += "\n"
	}

	prompt := historyContext + fmt.Sprintf(PromptRouterSystem, message)

	// Call Gemini with structured output
	resp, err := r.llm.GenerateContent(ctx, gemini.GenerateRequest{
		Contents: []gemini.Content{
			{
				Role: "user",
				Parts: []gemini.Part{
					{Text: prompt},
				},
			},
		},
		GenerationConfig: &gemini.GenerationConfig{
			Temperature: RouterTemperature,
		},
	})
	if err != nil {
		return RouterOutput{}, fmt.Errorf("%s: %s: %w", LogPrefixClassify, ErrMsgLLMCallFailed, err)
	}

	// Extract text from response
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		r.l.Warnf(ctx, "%s: %s", LogPrefixClassify, ErrMsgEmptyResponse)
		return RouterOutput{
			Intent:     RouterFallbackIntent,
			Confidence: RouterFallbackConfidence,
			Reasoning:  ReasonEmptyResponse,
		}, nil
	}

	responseText := resp.Candidates[0].Content.Parts[0].Text

	// Strip markdown code blocks if present (```json ... ```)
	responseText = strings.TrimSpace(responseText)
	if strings.HasPrefix(responseText, "```json") {
		responseText = strings.TrimPrefix(responseText, "```json")
		responseText = strings.TrimSuffix(responseText, "```")
		responseText = strings.TrimSpace(responseText)
	} else if strings.HasPrefix(responseText, "```") {
		responseText = strings.TrimPrefix(responseText, "```")
		responseText = strings.TrimSuffix(responseText, "```")
		responseText = strings.TrimSpace(responseText)
	}

	// Parse JSON response
	var output RouterOutput
	if err := json.Unmarshal([]byte(responseText), &output); err != nil {
		r.l.Warnf(ctx, "%s: %s: %v", LogPrefixClassify, ErrMsgJSONParseFailed, err)
		return RouterOutput{
			Intent:     RouterFallbackIntent,
			Confidence: RouterFallbackConfidence,
			Reasoning:  ReasonParsingError,
		}, nil
	}

	r.l.Infof(ctx, "%s: Classified as %s (confidence: %d%%)", LogPrefixClassify, output.Intent, output.Confidence)
	return output, nil
}
