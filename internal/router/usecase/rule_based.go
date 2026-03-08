package usecase

import (
	"strings"
	"unicode"

	"autonomous-task-management/internal/router"
)

// ruleBasedThreshold: nếu rule-based đạt >= ngưỡng này thì bỏ qua LLM call.
// 80 = "tự tin" — tiết kiệm 1 LLM round-trip cho ~70% messages phổ biến.
const ruleBasedThreshold = 80

// classifyByRules phân loại intent bằng pattern matching không dùng LLM.
// Trả về RouterOutput + bool (true = đủ tự tin, false = cần LLM).
func classifyByRules(message string) (router.RouterOutput, bool) {
	lower := normalize(message)

	// --- CREATE_TASK ---
	createScore := scorePatterns(lower, createSignals)
	if createScore >= ruleBasedThreshold {
		return router.RouterOutput{
			Intent:     router.IntentCreateTask,
			Confidence: createScore,
			Reasoning:  "rule-based: create signals detected",
		}, true
	}

	// --- SEARCH_TASK ---
	searchScore := scorePatterns(lower, searchSignals)
	if searchScore >= ruleBasedThreshold {
		return router.RouterOutput{
			Intent:     router.IntentSearchTask,
			Confidence: searchScore,
			Reasoning:  "rule-based: search signals detected",
		}, true
	}

	// --- MANAGE_CHECKLIST ---
	checklistScore := scorePatterns(lower, checklistSignals)
	if checklistScore >= ruleBasedThreshold {
		return router.RouterOutput{
			Intent:     router.IntentManageChecklist,
			Confidence: checklistScore,
			Reasoning:  "rule-based: checklist signals detected",
		}, true
	}

	// --- CONVERSATION ---
	convScore := scorePatterns(lower, conversationSignals)
	if convScore >= ruleBasedThreshold {
		return router.RouterOutput{
			Intent:     router.IntentConversation,
			Confidence: convScore,
			Reasoning:  "rule-based: conversational signals detected",
		}, true
	}

	// Ambiguous — cần LLM
	return router.RouterOutput{}, false
}

// scorePatterns tính điểm (0–100) dựa trên số pattern match.
// Mỗi pattern strong = 85 điểm (đủ vượt threshold ngay).
// Mỗi pattern medium = 40 điểm; cộng dồn nếu nhiều pattern match.
func scorePatterns(lower string, signals []signal) int {
	score := 0
	for _, s := range signals {
		if containsToken(lower, s.pattern) {
			score += s.weight
			if score >= 100 {
				return 100
			}
		}
	}
	return score
}

type signal struct {
	pattern string
	weight  int
}

// containsToken checks nếu text chứa pattern như một word/token.
// Dùng rune slice để xử lý đúng Vietnamese multibyte characters.
func containsToken(text, pattern string) bool {
	if !strings.Contains(text, pattern) {
		return false
	}
	textRunes := []rune(text)
	patRunes := []rune(pattern)
	patLen := len(patRunes)

	for i := 0; i <= len(textRunes)-patLen; i++ {
		// Check match at rune position i
		match := true
		for j := 0; j < patLen; j++ {
			if textRunes[i+j] != patRunes[j] {
				match = false
				break
			}
		}
		if !match {
			continue
		}
		// Check left boundary
		if i > 0 {
			prev := textRunes[i-1]
			if unicode.IsLetter(prev) || unicode.IsDigit(prev) {
				continue
			}
		}
		// Check right boundary
		end := i + patLen
		if end < len(textRunes) {
			next := textRunes[end]
			if unicode.IsLetter(next) || unicode.IsDigit(next) {
				continue
			}
		}
		return true
	}
	return false
}

// normalize: lowercase + collapse whitespace, giữ dấu tiếng Việt.
func normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// ---------------------------------------------------------------------------
// Signal tables — weight 85 = strong (1 signal đủ), 45 = medium (cần 2)
// ---------------------------------------------------------------------------

var createSignals = []signal{
	// Strong Vietnamese create verbs
	{"tạo task", 85},
	{"thêm task", 85},
	{"tạo công việc", 85},
	{"thêm công việc", 85},
	{"đặt lịch", 85},
	{"nhắc tôi", 85},
	{"nhắc nhở", 85},
	{"tạo nhắc", 85},
	{"lên kế hoạch", 85},
	{"schedule", 85},
	{"tạo việc", 85},
	// Medium signals — cần kết hợp
	{"deadline", 45},
	{"hạn chót", 45},
	{"due", 45},
	{"priority", 45},
	{"tạo", 40},
	{"thêm", 40},
	{"mới", 30},
}

var searchSignals = []signal{
	{"tìm task", 85},
	{"tìm kiếm", 85},
	{"search task", 85},
	{"tìm việc", 85},
	{"xem task", 85},
	{"danh sách task", 85},
	{"list task", 85},
	{"task nào", 85},
	{"công việc nào", 85},
	{"có task", 85},
	// Medium
	{"tìm", 45},
	{"search", 45},
	{"xem", 40},
	{"danh sách", 40},
	{"list", 40},
	{"tra cứu", 45},
	{"kiếm", 40},
}

var checklistSignals = []signal{
	{"đánh dấu hoàn thành", 85},
	{"mark done", 85},
	{"tick", 85},
	{"check off", 85},
	{"hoàn thành task", 85},
	{"done task", 85},
	{"complete task", 85},
	{"uncheck", 85},
	{"/check", 85},
	{"/uncheck", 85},
	{"/complete", 85},
	{"/progress", 85},
	// Medium
	{"checkbox", 45},
	{"hoàn thành", 45},
	{"done", 40},
	{"complete", 40},
	{"check", 40},
}

var conversationSignals = []signal{
	{"xin chào", 85},
	{"hello", 85},
	{"hi bot", 85},
	{"chào bot", 85},
	{"/start", 85},
	{"/help", 85},
	{"giúp tôi hiểu", 85},
	{"bạn là ai", 85},
	{"bot làm được gì", 85},
	{"hướng dẫn", 85},
	{"cảm ơn", 85},
	{"thanks", 85},
	// Medium casual
	{"chào", 45},
	{"hey", 45},
	{"ơi", 40},
}
