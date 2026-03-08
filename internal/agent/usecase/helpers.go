package usecase

import (
	"fmt"
	"time"

	"autonomous-task-management/pkg/llmprovider"
)

// ClearSession xoa GraphState cua user khoi cache.
// LRU tu dong quan ly TTL — khong can cleanup goroutine thu cong.
func (uc *implUseCase) ClearSession(userID string) {
	uc.stateCache.Remove(userID)
}

// GetSessionMessages tra ve lich su hoi thoai cua user tu GraphState.
// Tra ve nil neu chua co session.
func (uc *implUseCase) GetSessionMessages(userID string) []llmprovider.Message {
	state, ok := uc.stateCache.Get(userID)
	if !ok {
		return nil
	}
	return state.Messages
}

// convertToolsToNormalized chuyen tool registry sang format llmprovider.Tool.
func (uc *implUseCase) convertToolsToNormalized() []llmprovider.Tool {
	return uc.registry.ToFunctionDefinitions()
}

// buildTimeContext creates a temporal context string for LLM
func buildTimeContext(timezone string) string {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}

	now := time.Now().In(loc)

	// Calculate week boundaries (Monday-Sunday)
	weekday := int(now.Weekday())
	if weekday == 0 { // Sunday
		weekday = 7
	}
	weekStart := now.AddDate(0, 0, -(weekday - 1)) // Monday
	weekEnd := weekStart.AddDate(0, 0, 6)          // Sunday
	tomorrow := now.AddDate(0, 0, 1)

	// Build context string using template from constant.go
	context := fmt.Sprintf(
		TimeContextTemplate,
		now.Format(DateFormatISO),
		now.Weekday().String(),
		weekStart.Format(DateFormatISO),
		weekEnd.Format(DateFormatISO),
		tomorrow.Format(DateFormatISO),
		weekStart.Format(DateFormatISO),
		weekEnd.Format(DateFormatISO),
		tomorrow.Format(DateFormatISO),
	)

	return context
}
