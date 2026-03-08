package indexer

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEnrichTaskContent_WithDeadlineTomorrow(t *testing.T) {
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	content := fmt.Sprintf("## Review PR #123\n\n- **Due:** %s\n- **Priority:** #priority/p1", tomorrow)
	tags := []string{"#pr/123", "#priority/p1"}

	result := EnrichTaskContent(content, tags, "Asia/Ho_Chi_Minh")

	assert.Contains(t, result, "Review PR #123")
	assert.Contains(t, result, "ngay mai")
	assert.Contains(t, result, "tuan nay")
}

func TestEnrichTaskContent_WithDeadlineThisWeek(t *testing.T) {
	// Tim ngay cuoi tuan (chu nhat)
	now := time.Now()
	daysUntilSunday := int(time.Sunday-now.Weekday()+7) % 7
	if daysUntilSunday == 0 {
		daysUntilSunday = 7 // neu hom nay la chu nhat, lay chu nhat tuan sau
	}
	thisWeekDay := now.AddDate(0, 0, daysUntilSunday-1) // lay thu bay tuan nay
	deadline := thisWeekDay.Format("2006-01-02")
	content := fmt.Sprintf("## Task tuan nay\n\n- **Due:** %s\n", deadline)

	result := EnrichTaskContent(content, nil, "Asia/Ho_Chi_Minh")

	assert.Contains(t, result, "tuan")
}

func TestEnrichTaskContent_Overdue(t *testing.T) {
	past := time.Now().AddDate(0, 0, -3).Format("2006-01-02")
	content := fmt.Sprintf("## Old Task\n\n- **Due:** %s\n", past)

	result := EnrichTaskContent(content, nil, "Asia/Ho_Chi_Minh")

	assert.Contains(t, result, "qua han")
}

func TestEnrichTaskContent_Today(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	content := fmt.Sprintf("## Task hom nay\n\n- **Due:** %s\n", today)

	result := EnrichTaskContent(content, nil, "Asia/Ho_Chi_Minh")

	assert.Contains(t, result, "hom nay")
}

func TestEnrichTaskContent_NoDeadline(t *testing.T) {
	content := "## Task khong co deadline\n\nChi la mot task binh thuong."

	result := EnrichTaskContent(content, nil, "Asia/Ho_Chi_Minh")

	assert.Contains(t, result, "Task khong co deadline")
	assert.NotContains(t, result, "Deadline:")
}

func TestEnrichTaskContent_WithTags(t *testing.T) {
	content := "## Deploy staging\n\nDeploy service len staging."
	tags := []string{"#project/backend", "#type/deploy", "#priority/p1"}

	result := EnrichTaskContent(content, tags, "Asia/Ho_Chi_Minh")

	assert.Contains(t, result, "#project/backend")
	assert.Contains(t, result, "#type/deploy")
	assert.Contains(t, result, "#priority/p1")
}

func TestEnrichTaskContent_WithoutTags(t *testing.T) {
	content := "## Simple task"

	result := EnrichTaskContent(content, nil, "Asia/Ho_Chi_Minh")

	assert.NotContains(t, result, "Tags:")
}

func TestEnrichTaskContent_LimitsLength(t *testing.T) {
	// Content rat dai
	longContent := strings.Repeat("a", 2000)
	result := EnrichTaskContent(longContent, nil, "Asia/Ho_Chi_Minh")

	assert.LessOrEqual(t, len(result), 1200)
}

func TestEnrichTaskContent_StripMarkdown(t *testing.T) {
	content := "## Task Title\n\n**Bold text** and *italic* and [link](http://example.com)\n\n```go\nsome code\n```"

	result := EnrichTaskContent(content, nil, "Asia/Ho_Chi_Minh")

	assert.NotContains(t, result, "##")
	assert.NotContains(t, result, "**")
	assert.NotContains(t, result, "```")
}

func TestHumanizeDueDate(t *testing.T) {
	now := time.Now()

	tests := []struct {
		days     int
		contains string
	}{
		{-5, "qua han"},
		{-1, "qua han hom qua"},
		{0, "hom nay"},
		{1, "ngay mai"},
		{10, "tuan sau"},
	}

	for _, tt := range tests {
		due := now.AddDate(0, 0, tt.days)
		result := humanizeDueDate(due, now)
		assert.Contains(t, result, tt.contains, "days=%d", tt.days)
	}
}

func TestExtractDueDate(t *testing.T) {
	t.Run("finds due date in markdown", func(t *testing.T) {
		content := "## Task\n\n- **Due:** 2026-12-25\n- **Priority:** p1"
		got, ok := extractDueDate(content)
		assert.True(t, ok)
		assert.Equal(t, 2026, got.Year())
		assert.Equal(t, time.December, got.Month())
		assert.Equal(t, 25, got.Day())
	})

	t.Run("returns false when no due date", func(t *testing.T) {
		content := "## Task without deadline"
		_, ok := extractDueDate(content)
		assert.False(t, ok)
	})

	t.Run("case insensitive", func(t *testing.T) {
		content := "DUE: 2026-06-01"
		_, ok := extractDueDate(content)
		assert.True(t, ok)
	})
}
