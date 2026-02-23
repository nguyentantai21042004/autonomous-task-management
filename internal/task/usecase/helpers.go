package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"autonomous-task-management/pkg/gemini"
)

// parseInputWithLLM sends raw user text to Gemini and returns parsed tasks.
func (uc *implUseCase) parseInputWithLLM(ctx context.Context, rawText string) ([]gemini.ParsedTask, error) {
	loc, err := time.LoadLocation(uc.timezone)
	if err != nil {
		loc = time.UTC
	}
	nowStr := time.Now().In(loc).Format(time.RFC3339)
	prompt := gemini.BuildTaskParsingPrompt(rawText, nowStr)

	req := gemini.GenerateRequest{
		Contents: []gemini.Content{
			{
				Parts: []gemini.Part{
					{Text: prompt},
				},
			},
		},
		GenerationConfig: &gemini.GenerationConfig{
			Temperature:     0.2, // Low temperature for deterministic JSON output
			MaxOutputTokens: 2048,
		},
	}

	resp, err := uc.llm.GenerateContent(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from LLM")
	}

	responseText := resp.Candidates[0].Content.Parts[0].Text
	uc.l.Infof(ctx, "LLM raw response: %s", responseText)

	// Critical fix: sanitize before JSON unmarshal
	cleanedJSON := sanitizeJSONResponse(responseText)

	var tasks []gemini.ParsedTask
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
func (uc *implUseCase) resolveDueDates(parsed []gemini.ParsedTask) []taskWithDate {
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
