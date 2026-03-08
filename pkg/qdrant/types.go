package qdrant

// CreateCollectionRequest defines the schema for creating a collection.
type CreateCollectionRequest struct {
	Name    string       `json:"-"` // Collection name (in URL)
	Vectors VectorConfig `json:"vectors"`
}

// VectorConfig defines vector dimension and distance metric.
type VectorConfig struct {
	Size     int    `json:"size"`     // Vector dimension (e.g., 768 for Gemini)
	Distance string `json:"distance"` // "Cosine", "Euclid", "Dot"
}

// Point represents a vector with payload (metadata).
// CRITICAL: Qdrant requires ID to be UUID or uint64, NOT arbitrary string!
type Point struct {
	ID      interface{}            `json:"id"`      // UUID string or uint64 (NOT arbitrary string!)
	Vector  []float32              `json:"vector"`  // Embedding vector
	Payload map[string]interface{} `json:"payload"` // Metadata (memo_id, title, tags, etc.)
}

// UpsertPointsRequest is the request to insert/update points.
type UpsertPointsRequest struct {
	Points []Point `json:"points"`
}

// SearchRequest is the request for semantic search.
type SearchRequest struct {
	Vector      []float32              `json:"vector"`           // Query vector
	Limit       int                    `json:"limit"`            // Top-K results
	WithPayload bool                   `json:"with_payload"`     // Include metadata
	Filter      map[string]interface{} `json:"filter,omitempty"` // Optional filters
}

// SearchResponse contains search results.
type SearchResponse struct {
	Result []ScoredPoint `json:"result"`
}

// ScoredPoint is a search result with similarity score.
type ScoredPoint struct {
	ID      string                 `json:"id"`
	Score   float64                `json:"score"`
	Payload map[string]interface{} `json:"payload"`
}

// DeletePointsRequest is the request to delete points.
type DeletePointsRequest struct {
	Points []string `json:"points"`
}

// ScrollRequest fetches points by payload filter (used for keyword/text search).
type ScrollRequest struct {
	Filter      map[string]interface{} `json:"filter,omitempty"`
	Limit       int                    `json:"limit"`
	WithPayload bool                   `json:"with_payload"`
	WithVector  bool                   `json:"with_vector"`
}

// ScrollResponse contains paginated scroll results.
// Qdrant scroll API returns result as an object with "points" array.
type ScrollResponse struct {
	Result ScrollResult `json:"result"`
}

// ScrollResult is the inner result object of a scroll response.
type ScrollResult struct {
	Points         []ScoredPoint `json:"points"`
	NextPageOffset interface{}   `json:"next_page_offset"`
}

// CreatePayloadIndexRequest creates an index on a payload field.
type CreatePayloadIndexRequest struct {
	FieldName   string `json:"field_name"`
	FieldSchema string `json:"field_schema"` // "text", "keyword", "integer", etc.
}
