package qdrant

import "time"

const (
	// DefaultTimeout is the default timeout for Qdrant operations.
	DefaultTimeout = 30 * time.Second

	// DefaultPingTimeout is the timeout for initial connection ping in New.
	DefaultPingTimeout = 5 * time.Second

	// DefaultSearchLimit is the default number of results returned when limit is 0.
	DefaultSearchLimit = 10

	// Distance metric names (for GetDistanceMetric and config).
	DistanceCosine    = "cosine"
	DistanceEuclidean = "euclidean"
	DistanceDot       = "dot"
	DistanceManhattan = "manhattan"
)
