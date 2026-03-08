package usecase

import (
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
