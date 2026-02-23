package qdrant

import (
	"time"

	pb "github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
)

// Config is an alias for QdrantConfig (backward compatibility with config layer).
type Config = QdrantConfig

// QdrantConfig holds Qdrant configuration.
type QdrantConfig struct {
	Host    string
	Port    int
	UseTLS  bool
	APIKey  string
	Timeout time.Duration
}

// qdrantImpl implements IQdrant and wraps the Qdrant gRPC client.
type qdrantImpl struct {
	conn              *grpc.ClientConn
	pointsClient      pb.PointsClient
	collectionsClient pb.CollectionsClient
	defaultTimeout    time.Duration
}

// Point represents a vector point in Qdrant
type Point struct {
	ID      string
	Vector  []float32
	Payload map[string]interface{}
}

// SearchResult represents a search result from Qdrant
type SearchResult struct {
	ID      string
	Score   float32
	Payload map[string]interface{}
}

// CollectionInfo represents collection metadata
type CollectionInfo struct {
	Name        string
	VectorSize  uint64
	Distance    string
	PointsCount uint64
	Status      string
}

// GroupResult represents a group of search results
type GroupResult struct {
	ID    interface{} // Group Key (string or int)
	Hits  []SearchResult
	Count uint64
}

// FacetResult represents a facet value and its count
type FacetResult struct {
	Value interface{}
	Count uint64
}
