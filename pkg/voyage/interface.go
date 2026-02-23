package voyage

import (
	"context"
)

// IVoyage defines the interface for Voyage AI embeddings.
// Implementations are safe for concurrent use.
type IVoyage interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}
