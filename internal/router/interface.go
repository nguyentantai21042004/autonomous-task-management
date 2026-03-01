package router

import (
	"context"
)

// UseCase defines the business logic interface for the semantic router.
type UseCase interface {
	// Classify determines user intent from message
	Classify(ctx context.Context, message string, conversationHistory []string) (RouterOutput, error)
}
