package qdrant

import (
	"context"
	"fmt"

	pb "github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// IQdrant aggregates all Qdrant vector DB operations.
type IQdrant interface {
	CollectionsOps
	PointsOps
	SearchOps
	Close() error
	Ping(ctx context.Context) error
}

// CollectionsOps defines interface for collection-related operations.
type CollectionsOps interface {
	CreateCollection(ctx context.Context, name string, vectorSize uint64, distance pb.Distance) error
	DeleteCollection(ctx context.Context, name string) error
	CollectionExists(ctx context.Context, name string) (bool, error)
	GetCollectionInfo(ctx context.Context, name string) (*CollectionInfo, error)
	ListCollections(ctx context.Context) ([]string, error)
}

// PointsOps defines interface for point-related operations.
type PointsOps interface {
	UpsertPoint(ctx context.Context, colName string, point Point) error
	UpsertPoints(ctx context.Context, colName string, points []Point) error
	DeletePoint(ctx context.Context, colName string, pointID string) error
	GetPoint(ctx context.Context, colName string, pointID string) (*Point, error)
	CountPoints(ctx context.Context, colName string) (uint64, error)
}

// SearchOps defines interface for search operations.
type SearchOps interface {
	Search(ctx context.Context, colName string, vector []float32, limit uint64) ([]SearchResult, error)
	SearchWithFilter(ctx context.Context, colName string, vector []float32, limit uint64, filter *pb.Filter) ([]SearchResult, error)
	SearchBatch(ctx context.Context, colName string, vectors [][]float32, limit uint64) ([][]SearchResult, error)
	SearchGroups(ctx context.Context, colName string, vector []float32, limit uint64, groupBy string, groupLimit uint64, filter *pb.Filter) ([]GroupResult, error)
	Facet(ctx context.Context, colName string, key string, limit uint64, filter *pb.Filter) ([]FacetResult, error)
}

// New creates a new Qdrant client. Returns an implementation of IQdrant.
func NewQdrant(cfg QdrantConfig) (IQdrant, error) {
	// Validate config directly here
	if cfg.Host == "" {
		return nil, fmt.Errorf("invalid config: host is required")
	}
	if cfg.Port == 0 {
		return nil, fmt.Errorf("invalid config: port is required")
	}

	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultTimeout
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	var opts []grpc.DialOption
	if cfg.UseTLS {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Qdrant: %w", err)
	}

	client := &qdrantImpl{
		conn:              conn,
		pointsClient:      pb.NewPointsClient(conn),
		collectionsClient: pb.NewCollectionsClient(conn),
		defaultTimeout:    cfg.Timeout,
	}

	ctx, cancel := context.WithTimeout(context.Background(), DefaultPingTimeout)
	defer cancel()

	if err := client.Ping(ctx); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to ping Qdrant: %w", err)
	}

	return client, nil
}
