package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"autonomous-task-management/internal/router"
	"autonomous-task-management/pkg/llmprovider"
)

func (uc *implUseCase) Classify(ctx context.Context, message string, conversationHistory []string) (router.RouterOutput, error) {
	// Fast path: rule-based classifier — bỏ qua LLM nếu đủ tự tin
	if result, confident := classifyByRules(message); confident {
		uc.l.Infof(ctx, "%s: rule-based → %s (confidence: %d%%)", LogPrefixClassify, result.Intent, result.Confidence)
		return result, nil
	}

	// Slow path: LLM fallback cho ambiguous messages
	historyContext := ""
	if len(conversationHistory) > 0 {
		historyContext = PromptHistoryPrefix
		for i, msg := range conversationHistory {
			historyContext += fmt.Sprintf("%d. %s\n", i+1, msg)
		}
		historyContext += "\n"
	}

	prompt := historyContext + fmt.Sprintf(PromptRouterSystem, message)

	// Call Provider Manager with normalized request
	resp, err := uc.llm.GenerateContent(ctx, &llmprovider.Request{
		Messages: []llmprovider.Message{
			{
				Role: "user",
				Parts: []llmprovider.Part{
					{Text: prompt},
				},
			},
		},
		Temperature: RouterTemperature,
	})
	if err != nil {
		return router.RouterOutput{}, fmt.Errorf("%s: %s: %w", LogPrefixClassify, ErrMsgLLMCallFailed, err)
	}

	if len(resp.Content.Parts) == 0 {
		uc.l.Warnf(ctx, "%s: %s", LogPrefixClassify, ErrMsgEmptyResponse)
		return router.RouterOutput{
			Intent:     RouterFallbackIntent,
			Confidence: RouterFallbackConfidence,
			Reasoning:  ReasonEmptyResponse,
		}, nil
	}

	responseText := resp.Content.Parts[0].Text

	// Clean JSON response
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

	var output router.RouterOutput
	if err := json.Unmarshal([]byte(responseText), &output); err != nil {
		uc.l.Warnf(ctx, "%s: %s: %v", LogPrefixClassify, ErrMsgJSONParseFailed, err)
		return router.RouterOutput{
			Intent:     RouterFallbackIntent,
			Confidence: RouterFallbackConfidence,
			Reasoning:  ReasonParsingError,
		}, nil
	}

	uc.l.Infof(ctx, "%s: Classified as %s (confidence: %d%%)", LogPrefixClassify, output.Intent, output.Confidence)
	return output, nil
}
