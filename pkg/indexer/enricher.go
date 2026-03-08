// Package indexer cung cap cac utility de lam giau noi dung task truoc khi embed.
//
// Van de voi V1.2: embed raw content "Review PR #123" → vector qua kho chua deadline/tags.
// Query "deadline tuan nay" → MISS vi vector khong co chuoi "tuan nay".
//
// Giai phap: Contextual Enrichment — build enriched text truoc khi embed:
// "Review PR #123 | Deadline: ngay mai (tuan nay) | Tags: #pr/123, #backend"
// → User query "deadline tuan nay" → HIT ✅
package indexer

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// EnrichTaskContent lam giau noi dung task voi temporal context truoc khi embed.
// content: markdown content cua task (da co - **Due:** yyyy-mm-dd)
// tags: danh sach tags cua task
// timezone: timezone cua user de tinh humanized date
func EnrichTaskContent(content string, tags []string, timezone string) string {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}
	now := time.Now().In(loc)

	var parts []string

	// 1. Giu noi dung goc (stripped markdown)
	cleaned := stripMarkdown(content)
	if cleaned != "" {
		parts = append(parts, cleaned)
	}

	// 2. Them temporal context neu co due date trong content
	if dueDate, ok := extractDueDate(content); ok {
		humanized := humanizeDueDate(dueDate, now)
		parts = append(parts, fmt.Sprintf("Deadline: %s (%s)", dueDate.Format("02/01/2006"), humanized))
	}

	// 3. Them tags de vector capture exact keyword matching
	if len(tags) > 0 {
		parts = append(parts, "Tags: "+strings.Join(tags, ", "))
	}

	result := strings.Join(parts, " | ")

	// Gioi han 1200 chars de tranh embedding API limits
	if len(result) > 1200 {
		result = result[:1200]
	}

	return result
}

// extractDueDate phan tich due date tu markdown content.
// Tim dong "- **Due:** 2026-03-15" hoac "Due: 2026-03-15"
func extractDueDate(content string) (time.Time, bool) {
	// Pattern: - **Due:** 2026-03-15 hoac Due: 2026-03-15
	re := regexp.MustCompile(`(?i)due[:\s*]+(\d{4}-\d{2}-\d{2})`)
	matches := re.FindStringSubmatch(content)
	if len(matches) < 2 {
		return time.Time{}, false
	}

	t, err := time.Parse("2006-01-02", matches[1])
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

// humanizeDueDate chuyen due date thanh string de hieu: "ngay mai", "tuan nay", "qua han 3 ngay".
// Day la key insight: vector capture "tuan nay" → query "deadline tuan nay" match.
func humanizeDueDate(due, now time.Time) string {
	// Tinh so ngay chenh lech (bo qua gio)
	dueDay := time.Date(due.Year(), due.Month(), due.Day(), 0, 0, 0, 0, due.Location())
	nowDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	days := int(dueDay.Sub(nowDay).Hours() / 24)

	switch {
	case days < -1:
		return fmt.Sprintf("qua han %d ngay", -days)
	case days == -1:
		return "qua han hom qua"
	case days == 0:
		return "hom nay"
	case days == 1:
		return "ngay mai, tuan nay"
	case days <= 6:
		// Kiem tra co trong tuan nay khong
		daysUntilWeekEnd := int(time.Sunday-now.Weekday()+7) % 7
		if days <= daysUntilWeekEnd {
			return fmt.Sprintf("%d ngay nua, tuan nay", days)
		}
		return fmt.Sprintf("%d ngay nua, tuan sau", days)
	case days <= 13:
		return "tuan sau"
	case days <= 30:
		return fmt.Sprintf("%d ngay nua, thang nay", days)
	default:
		return fmt.Sprintf("%d ngay nua", days)
	}
}

// stripMarkdown loai bo markdown formatting de giu lai text thuan.
func stripMarkdown(text string) string {
	// Xoa code blocks
	re := regexp.MustCompile("(?s)```[a-z]*\\n?.*?\\n?```")
	text = re.ReplaceAllString(text, "")

	// Xoa markdown headings ##, ###
	re = regexp.MustCompile(`(?m)^#{1,6}\s+`)
	text = re.ReplaceAllString(text, "")

	// Xoa bold/italic **text** hoac *text*
	text = strings.ReplaceAll(text, "**", "")
	text = strings.ReplaceAll(text, "*", "")

	// Xoa bullet points
	re = regexp.MustCompile(`(?m)^\s*[-*+]\s+`)
	text = re.ReplaceAllString(text, "")

	// Xoa links [text](url)
	re = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
	text = re.ReplaceAllString(text, "$1")

	// Collapse multiple newlines/spaces
	re = regexp.MustCompile(`\s+`)
	text = re.ReplaceAllString(text, " ")

	return strings.TrimSpace(text)
}
