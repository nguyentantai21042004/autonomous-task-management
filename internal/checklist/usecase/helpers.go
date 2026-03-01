package usecase

import (
	"regexp"
)

// sanitizeContent removes code blocks before checkbox parsing
// Prevents matching fake checkboxes in code examples
func sanitizeContent(content string) string {
	// Remove fenced code blocks (```...```)
	fencedCodeBlockPattern := regexp.MustCompile("(?s)```.*?```")
	sanitized := fencedCodeBlockPattern.ReplaceAllString(content, "")

	// Remove inline code blocks (`...`)
	inlineCodePattern := regexp.MustCompile("`[^`]+`")
	sanitized = inlineCodePattern.ReplaceAllString(sanitized, "")

	return sanitized
}
