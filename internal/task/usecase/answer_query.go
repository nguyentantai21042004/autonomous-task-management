package usecase

import (
	"context"
	"fmt"
	"strings"

	"autonomous-task-management/internal/model"
	"autonomous-task-management/internal/task"
	"autonomous-task-management/internal/task/repository"
	"autonomous-task-management/pkg/gemini"
)

const (
	MaxTasksInContext = 5   // Top-5 most relevant tasks
	MaxCharsPerTask   = 800 // Truncate each task to 800 chars
)

// AnswerQuery uses RAG to answer questions about tasks.
func (uc *implUseCase) AnswerQuery(ctx context.Context, sc model.Scope, input task.QueryInput) (task.QueryOutput, error) {
	if input.Query == "" {
		return task.QueryOutput{}, task.ErrEmptyQuery
	}

	uc.l.Infof(ctx, "AnswerQuery: user=%s query=%q", sc.UserID, input.Query)

	// Step 1: Search for relevant tasks
	searchResults, err := uc.vectorRepo.SearchTasks(ctx, repository.SearchTasksOptions{
		Query: input.Query,
		Limit: MaxTasksInContext,
	})
	if err != nil {
		return task.QueryOutput{}, fmt.Errorf("failed to search tasks: %w", err)
	}

	if len(searchResults) == 0 {
		return task.QueryOutput{
			Answer:      "Không tìm thấy task nào liên quan đến câu hỏi của bạn.",
			SourceCount: 0,
		}, nil
	}

	// Step 2: Build context with truncation
	var contextBuilder strings.Builder
	contextBuilder.WriteString("Ngữ cảnh (Các task liên quan):\n\n")

	for i, sr := range searchResults {
		memoTask, err := uc.repo.GetTask(ctx, sr.MemoID)
		if err != nil {
			uc.l.Warnf(ctx, "Failed to fetch task %s: %v", sr.MemoID, err)
			continue
		}

		// CRITICAL: Truncate to prevent token overflow
		safeContent := truncateText(memoTask.Content, MaxCharsPerTask)

		contextBuilder.WriteString(fmt.Sprintf("-- Task %d (Độ phù hợp: %.0f%%, Link: %s) --\n%s\n\n",
			i+1, sr.Score*100, memoTask.MemoURL, safeContent))
	}

	// Step 3: Build prompt
	prompt := fmt.Sprintf(`%s

Nhiệm vụ: Trả lời câu hỏi sau dựa trên ngữ cảnh được cung cấp.
- Nếu ngữ cảnh không có thông tin, hãy nói rõ là không biết.
- Luôn đính kèm link task nếu có trích dẫn.
- Trả lời bằng tiếng Việt, ngắn gọn và rõ ràng.

Câu hỏi: "%s"`, contextBuilder.String(), input.Query)

	// Step 4: Call LLM
	req := gemini.GenerateRequest{
		Contents: []gemini.Content{
			{Parts: []gemini.Part{{Text: prompt}}},
		},
		GenerationConfig: &gemini.GenerationConfig{
			Temperature:     0.3, // Lower temperature for factual answers
			MaxOutputTokens: 1024,
		},
	}

	resp, err := uc.llm.GenerateContent(ctx, req)
	if err != nil {
		return task.QueryOutput{}, fmt.Errorf("LLM failed: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return task.QueryOutput{}, fmt.Errorf("empty LLM response")
	}

	answerText := resp.Candidates[0].Content.Parts[0].Text

	return task.QueryOutput{
		Answer:      answerText,
		SourceTasks: searchResults,
		SourceCount: len(searchResults),
	}, nil
}

// truncateText safely truncates text to maxLen (Unicode-safe for Vietnamese).
func truncateText(text string, maxLen int) string {
	runes := []rune(text) // Convert to Unicode characters (not bytes)
	if len(runes) <= maxLen {
		return text
	}
	return string(runes[:maxLen]) + "... [đã cắt bớt]"
}
