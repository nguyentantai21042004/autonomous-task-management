package usecase

import (
	"context"
	"time"

	"autonomous-task-management/internal/agent"
	"autonomous-task-management/pkg/llmprovider"
)

// ClearSession removes conversation history for a user
func (uc *implUseCase) ClearSession(userID string) {
	uc.cacheMutex.Lock()
	defer uc.cacheMutex.Unlock()
	delete(uc.sessionCache, userID)
}

// GetSessionMessages retrieves the conversation history for a user
func (uc *implUseCase) GetSessionMessages(userID string) []llmprovider.Message {
	uc.cacheMutex.RLock()
	defer uc.cacheMutex.RUnlock()

	session, exists := uc.sessionCache[userID]
	if !exists {
		return nil
	}
	return session.Messages
}

// getSession retrieves or creates session for user
func (uc *implUseCase) getSession(userID string) *agent.SessionMemory {
	uc.cacheMutex.Lock()
	defer uc.cacheMutex.Unlock()

	session, exists := uc.sessionCache[userID]
	if !exists || time.Since(session.LastUpdated) > uc.cacheTTL {
		session = &agent.SessionMemory{
			UserID:      userID,
			Messages:    []llmprovider.Message{},
			LastUpdated: time.Now(),
		}
		uc.sessionCache[userID] = session
	}

	return session
}

// cleanupExpiredSessions runs periodically to remove expired sessions
func (uc *implUseCase) cleanupExpiredSessions() {
	ticker := time.NewTicker(SessionCleanupInterval * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			uc.cacheMutex.Lock()

			now := time.Now()
			expiredKeys := make([]string, 0)

			for userID, session := range uc.sessionCache {
				if now.Sub(session.LastUpdated) > uc.cacheTTL {
					expiredKeys = append(expiredKeys, userID)
				}
			}

			for _, userID := range expiredKeys {
				delete(uc.sessionCache, userID)
			}

			uc.cacheMutex.Unlock()

			if len(expiredKeys) > 0 {
				uc.l.Infof(context.Background(),
					"%s: "+LogMsgSessionsCleanedUp, LogPrefixCleanupSessions, len(expiredKeys))
			}
		case <-uc.stopCleanup:
			return
		}
	}
}

// convertToolsToNormalized converts tool registry to normalized llmprovider.Tool format
func (uc *implUseCase) convertToolsToNormalized() []llmprovider.Tool {
	return uc.registry.ToFunctionDefinitions()
}
