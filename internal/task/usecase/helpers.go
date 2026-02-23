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
	prompt := gemini.BuildTaskParsingPrompt(rawText)

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

// resolveDueDates resolves relative dates from parsed tasks into absolute times.
func (uc *implUseCase) resolveDueDates(parsed []gemini.ParsedTask) []taskWithDate {
	now := time.Now()
	result := make([]taskWithDate, 0, len(parsed))

	for _, p := range parsed {
		absTime, err := uc.dateMath.Parse(p.DueDateRelative, now)
		if err != nil {
			uc.l.Infof(context.Background(), "Failed to parse relative date %q, defaulting to today: %v", p.DueDateRelative, err)
			absTime = uc.dateMath.EndOfDay(now)
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
