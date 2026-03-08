package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"autonomous-task-management/internal/task/repository"
	"autonomous-task-management/pkg/llmprovider"
)

// taskParsingSystemPrompt is the system instruction sent to LLM for task parsing.
const taskParsingSystemPrompt = `You are a task parsing assistant. Your job is to extract structured tasks from user input.

RULES:
1. Parse the input text and extract all individual tasks.
2. For each task, identify:
   - title: Short, clear task description (required)
   - description: Additional details (can be empty string)
   - due_date_absolute: Absolute ISO8601 (RFC3339) date-time string (e.g., "2026-02-24T09:00:00+07:00"). If a specific time is mentioned (e.g., "9h sáng", "3h chiều"), use it. If only a date is mentioned and no specific time, default to 23:59:59 of that target day.
   - priority: MUST be exactly one of: "p0", "p1", "p2", "p3"
   - tags: Array of tag strings following the format #category/value
   - estimated_duration_minutes: Integer number of minutes (minimum 15, default 60)

3. Return ONLY a valid JSON array. No markdown, no code blocks, no explanation text.
4. If no specific date mentioned at all, default due_date_absolute to today's 23:59:59.
5. If no priority mentioned, default to "p2".
6. Infer relevant tags from context (domain, project, type).

EXAMPLE INPUT:
"Finish SMAP report by tomorrow, review code for Ahamove project today p1, prepare presentation next Monday"

EXAMPLE OUTPUT:
[
  {
    "title": "Finish SMAP report",
    "description": "",
    "due_date_absolute": "2026-02-24T23:59:59+07:00",
    "priority": "p2",
    "tags": ["#project/smap", "#type/research"],
    "estimated_duration_minutes": 120
  },
  {
    "title": "Review code for Ahamove project",
    "description": "",
    "due_date_absolute": "2026-02-23T23:59:59+07:00",
    "priority": "p1",
    "tags": ["#domain/ahamove", "#type/review"],
    "estimated_duration_minutes": 60
  },
  {
    "title": "Prepare presentation",
    "description": "",
    "due_date_absolute": "2026-03-02T23:59:59+07:00",
    "priority": "p2",
    "tags": ["#type/meeting"],
    "estimated_duration_minutes": 90
  }
]

Now parse the following input and return ONLY the JSON array:`

// buildTaskParsingPrompt builds the full prompt for task parsing.
func buildTaskParsingPrompt(userInput string, currentTime string) string {
	return taskParsingSystemPrompt + "\n\nCURRENT MOCK CONTEXT (USE FOR RELATIVE DATE/TIME RESOLUTION):\n" + currentTime + "\n\nNow parse the following input and return ONLY the JSON array:\n" + userInput
}

// parseInputWithLLM sends raw user text to LLM and returns parsed tasks.
func (uc *implUseCase) parseInputWithLLM(ctx context.Context, rawText string) ([]ParsedTask, error) {
	loc, err := time.LoadLocation(uc.timezone)
	if err != nil {
		loc = time.UTC
	}
	nowStr := time.Now().In(loc).Format(time.RFC3339)
	prompt := buildTaskParsingPrompt(rawText, nowStr)

	req := &llmprovider.Request{
		Messages: []llmprovider.Message{
			{
				Role: "user",
				Parts: []llmprovider.Part{
					{Text: prompt},
				},
			},
		},
		Temperature: 0.2, // Low temperature for deterministic JSON output
		MaxTokens:   2048,
	}

	resp, err := uc.llm.GenerateContent(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	if len(resp.Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from LLM")
	}

	responseText := resp.Content.Parts[0].Text
	uc.l.Infof(ctx, "LLM raw response: %s", responseText)

	// Critical fix: sanitize before JSON unmarshal
	cleanedJSON := sanitizeJSONResponse(responseText)

	var tasks []ParsedTask
	if err := json.Unmarshal([]byte(cleanedJSON), &tasks); err != nil {
		uc.l.Errorf(ctx, "Failed to parse LLM response. Raw=%q Cleaned=%q", responseText, cleanedJSON)
		return nil, fmt.Errorf("failed to parse LLM JSON response: %w", err)
	}

	return tasks, nil
}

// sanitizeJSONResponse removes markdown code fences and leading/trailing prose
// that LLMs often add around JSON output.
func sanitizeJSONResponse(text string) string {
	// Remove ```json ... ``` or ``` ... ``` blocks
	re := regexp.MustCompile("(?s)```(?:json)?\\s*(.+?)\\s*```")
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// No code block: find first [ or { and last ] or }
	start := strings.IndexAny(text, "[{")
	if start == -1 {
		return text
	}
	end := strings.LastIndexAny(text, "]}")
	if end == -1 || end < start {
		return text
	}
	return strings.TrimSpace(text[start : end+1])
}

// resolveDueDates resolves absolute dates from parsed tasks into time.Time.
func (uc *implUseCase) resolveDueDates(parsed []ParsedTask) []taskWithDate {
	now := time.Now()
	if loc, err := time.LoadLocation(uc.timezone); err == nil {
		now = now.In(loc)
	}

	result := make([]taskWithDate, 0, len(parsed))

	for _, p := range parsed {
		absTime, err := time.Parse(time.RFC3339, p.DueDateAbsolute)
		if err != nil {
			uc.l.Infof(context.Background(), "Failed to parse absolute date %q from LLM, defaulting to end of today: %v", p.DueDateAbsolute, err)
			todayStart, _ := uc.dateMath.Parse("today", now)
			absTime = uc.dateMath.EndOfDay(todayStart)
		}

		result = append(result, taskWithDate{
			Title:                    p.Title,
			Description:              p.Description,
			DueDateAbsolute:          absTime,
			Priority:                 p.Priority,
			Tags:                     p.Tags,
			EstimatedDurationMinutes: p.EstimatedDurationMinutes,
		})
	}
	return result
}

// buildMarkdownContent builds the full Markdown body for a task memo.
func buildMarkdownContent(t taskWithDate) string {
	var sb strings.Builder

	// Title as heading
	sb.WriteString(fmt.Sprintf("## %s\n\n", t.Title))

	// Description block
	if t.Description != "" {
		sb.WriteString(t.Description)
		sb.WriteString("\n\n")
	}

	// Metadata block
	sb.WriteString(fmt.Sprintf("- **Due:** %s\n", t.DueDateAbsolute.Format("2006-01-02")))
	sb.WriteString(fmt.Sprintf("- **Priority:** #priority/%s\n", t.Priority))

	if t.EstimatedDurationMinutes > 0 {
		sb.WriteString(fmt.Sprintf("- **Estimated:** %d min\n", t.EstimatedDurationMinutes))
	}

	return sb.String()
}

// priorityToTag maps priority string to full tag representation.
func priorityTag(priority string) string {
	return fmt.Sprintf("#priority/%s", priority)
}

// allTags returns all tags for a task including the priority tag.
func allTags(t taskWithDate) []string {
	tags := make([]string, 0, len(t.Tags)+1)
	tags = append(tags, priorityTag(t.Priority))
	tags = append(tags, t.Tags...)
	return tags
}

// hybridRerank optionally calls Voyage cross-encoder reranker on pre-fused results from repo layer,
// then returns top MaxTasksInContext results.
// NOTE: BM25+RRF fusion is already done at the repository level (qdrant/task.go SearchTasks),
// so we skip the redundant second round here to avoid score distortion.
func (uc *implUseCase) hybridRerank(ctx context.Context, query string, denseResults []repository.SearchResult) []repository.SearchResult {
	if len(denseResults) == 0 {
		return denseResults
	}

	// Limit candidates to over-fetch pool
	limit := MaxTasksInContext * overFetchMultiplier
	if len(denseResults) < limit {
		limit = len(denseResults)
	}
	candidates := denseResults[:limit]

	// Optional: Voyage cross-encoder rerank
	if uc.reranker != nil && len(candidates) > 0 {
		docs := make([]string, len(candidates))
		for i, c := range candidates {
			if c.Payload != nil {
				if content, ok := c.Payload["content"].(string); ok {
					docs[i] = content
					continue
				}
			}
			docs[i] = c.MemoID
		}

		rerankResults, err := uc.reranker.Rerank(ctx, query, docs, MaxTasksInContext)
		if err != nil {
			uc.l.Warnf(ctx, "hybridRerank: reranker failed, using original order: %v", err)
		} else {
			reranked := make([]repository.SearchResult, 0, len(rerankResults))
			for _, rr := range rerankResults {
				if rr.Index < len(candidates) {
					c := candidates[rr.Index]
					reranked = append(reranked, repository.SearchResult{
						MemoID:  c.MemoID,
						Score:   rr.RelevanceScore,
						Payload: c.Payload,
					})
				}
			}
			return reranked
		}
	}

	// No reranker: just return top-K from pre-fused results
	top := MaxTasksInContext
	if len(candidates) < top {
		top = len(candidates)
	}
	return candidates[:top]
}
