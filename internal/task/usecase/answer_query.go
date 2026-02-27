package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"autonomous-task-management/internal/model"
	"autonomous-task-management/internal/task"
	"autonomous-task-management/internal/task/repository"
	"autonomous-task-management/pkg/llmprovider"
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

	zombieVectors := make([]string, 0) // 🆕 Track zombie vectors for cleanup

	for i, sr := range searchResults {
		memoTask, err := uc.repo.GetTask(ctx, sr.MemoID)
		if err != nil {
			// 🆕 Self-healing: cleanup zombie vectors
			if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "Not Found") {
				uc.l.Warnf(ctx, "AnswerQuery: Task %s deleted in Memos. Self-healing: removing from Qdrant", sr.MemoID)
				zombieVectors = append(zombieVectors, sr.MemoID)

				// Async cleanup (don't block RAG)
				go func(memoID string) {
					cleanupCtx := context.Background()
					if err := uc.vectorRepo.DeleteTask(cleanupCtx, memoID); err != nil {
						uc.l.Errorf(cleanupCtx, "Self-healing: Failed to cleanup zombie vector %s: %v", memoID, err)
					} else {
						uc.l.Infof(cleanupCtx, "Self-healing: Successfully cleaned up zombie vector %s", memoID)
					}
				}(sr.MemoID)

				continue
			}

			uc.l.Warnf(ctx, "AnswerQuery: failed to fetch task %s: %v", sr.MemoID, err)
			continue
		}

		// CRITICAL: Truncate to prevent token overflow
		safeContent := truncateText(memoTask.Content, MaxCharsPerTask)

		contextBuilder.WriteString(fmt.Sprintf("-- Task %d (Độ phù hợp: %.0f%%, Link: %s) --\n%s\n\n",
			i+1, sr.Score*100, memoTask.MemoURL, safeContent))
	}

	// 🆕 Log self-healing stats
	if len(zombieVectors) > 0 {
		uc.l.Infof(ctx, "AnswerQuery: Self-healing cleaned up %d zombie vectors", len(zombieVectors))
	}

	// Step 3: Build prompt
	loc, err := time.LoadLocation(uc.timezone)
	if err != nil {
		loc = time.UTC
	}
	currentTime := time.Now().In(loc)
	dateContext := fmt.Sprintf(
		"Hôm nay là %s, ngày %s. Timezone: %s.",
		currentTime.Weekday().String(),
		currentTime.Format("02/01/2006 15:04:05"),
		currentTime.Location().String(),
	)

	prompt := fmt.Sprintf(`%s

Thời gian hiện tại của hệ thống: %s

Nhiệm vụ: Trả lời câu hỏi sau dựa trên ngữ cảnh được cung cấp.
- Phân tích và đối chiếu các mốc thời gian (ví dụ: ngày mai, tuần sau) dựa trên Thời gian hiện tại.
- Nếu ngữ cảnh không có đề cập thông tin để trả lời, tuyệt đối không bịa ra nội dung, hãy nói rõ là không biết.
- Luôn đính kèm link phần mềm (MemoURL) ở mỗi tác vụ khi trích dẫn.
- Trả lời bằng tiếng Việt, ngắn gọn và mạch lạc.

Câu hỏi: "%s"`, contextBuilder.String(), dateContext, input.Query)

	// Step 4: Call LLM
	req := &llmprovider.Request{
		Messages: []llmprovider.Message{
			{
				Role: "user",
				Parts: []llmprovider.Part{
					{Text: prompt},
				},
			},
		},
		Temperature: 0.3, // Lower temperature for factual answers
		MaxTokens:   1024,
	}

	resp, err := uc.llm.GenerateContent(ctx, req)
	if err != nil {
		return task.QueryOutput{}, fmt.Errorf("LLM failed: %w", err)
	}

	if len(resp.Content.Parts) == 0 {
		return task.QueryOutput{}, fmt.Errorf("empty LLM response")
	}

	answerText := resp.Content.Parts[0].Text

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
