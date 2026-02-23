package qdrant

import (
	"context"
	"fmt"
	"time"

	pb "github.com/qdrant/go-client/qdrant"
)

// Close closes the Qdrant connection.
func (c *qdrantImpl) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Ping checks if Qdrant is reachable.
func (c *qdrantImpl) Ping(ctx context.Context) error {
	_, err := c.collectionsClient.List(ctx, &pb.ListCollectionsRequest{})
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}
	return nil
}

// GetPointsClient returns the underlying points client for advanced operations.
func (c *qdrantImpl) GetPointsClient() pb.PointsClient {
	return c.pointsClient
}

// GetCollectionsClient returns the underlying collections client for advanced operations.
func (c *qdrantImpl) GetCollectionsClient() pb.CollectionsClient {
	return c.collectionsClient
}

// GetDefaultTimeout returns the default timeout for operations.
func (c *qdrantImpl) GetDefaultTimeout() time.Duration {
	return c.defaultTimeout
}

// CreateCollection creates a new collection in Qdrant.
func (c *qdrantImpl) CreateCollection(ctx context.Context, name string, vectorSize uint64, distance pb.Distance) error {
	if name == "" {
		return ErrEmptyCollection
	}
	if vectorSize == 0 {
		return ErrInvalidVectorSize
	}
	_, err := c.collectionsClient.Create(ctx, &pb.CreateCollection{
		CollectionName: name,
		VectorsConfig: &pb.VectorsConfig{
			Config: &pb.VectorsConfig_Params{
				Params: &pb.VectorParams{
					Size:     vectorSize,
					Distance: distance,
				},
			},
		},
	})
	if err != nil {
		return WrapError(err, "failed to create collection")
	}
	return nil
}

// DeleteCollection deletes a collection from Qdrant.
func (c *qdrantImpl) DeleteCollection(ctx context.Context, name string) error {
	if name == "" {
		return ErrEmptyCollection
	}
	_, err := c.collectionsClient.Delete(ctx, &pb.DeleteCollection{CollectionName: name})
	if err != nil {
		return WrapError(err, "failed to delete collection")
	}
	return nil
}

// CollectionExists checks if a collection exists.
func (c *qdrantImpl) CollectionExists(ctx context.Context, name string) (bool, error) {
	if name == "" {
		return false, ErrEmptyCollection
	}
	resp, err := c.collectionsClient.Get(ctx, &pb.GetCollectionInfoRequest{CollectionName: name})
	if err != nil {
		return false, nil
	}
	return resp != nil, nil
}

// GetCollectionInfo retrieves information about a collection.
func (c *qdrantImpl) GetCollectionInfo(ctx context.Context, name string) (*CollectionInfo, error) {
	if name == "" {
		return nil, ErrEmptyCollection
	}
	resp, err := c.collectionsClient.Get(ctx, &pb.GetCollectionInfoRequest{CollectionName: name})
	if err != nil {
		return nil, WrapError(err, "failed to get collection info")
	}
	if resp.Result == nil {
		return nil, ErrCollectionNotFound
	}
	info := &CollectionInfo{
		Name:        name,
		Status:      resp.Result.Status.String(),
		PointsCount: resp.Result.GetPointsCount(),
	}
	if resp.Result.Config != nil && resp.Result.Config.Params != nil {
		if vectorConfig := resp.Result.Config.Params.VectorsConfig; vectorConfig != nil {
			if params := vectorConfig.GetParams(); params != nil {
				info.VectorSize = params.Size
				info.Distance = params.Distance.String()
			}
		}
	}
	return info, nil
}

// ListCollections lists all collections.
func (c *qdrantImpl) ListCollections(ctx context.Context) ([]string, error) {
	resp, err := c.collectionsClient.List(ctx, &pb.ListCollectionsRequest{})
	if err != nil {
		return nil, WrapError(err, "failed to list collections")
	}
	collections := make([]string, 0, len(resp.Collections))
	for _, col := range resp.Collections {
		collections = append(collections, col.Name)
	}
	return collections, nil
}

// UpsertPoint inserts or updates a point in a collection.
func (c *qdrantImpl) UpsertPoint(ctx context.Context, collectionName string, point Point) error {
	if collectionName == "" {
		return ErrEmptyCollection
	}
	if point.ID == "" {
		return ErrInvalidPointID
	}
	if len(point.Vector) == 0 {
		return ErrInvalidVector
	}
	payloadMap, err := pb.TryValueMap(point.Payload)
	if err != nil {
		return WrapError(err, "failed to convert payload")
	}
	qdrantPoint := &pb.PointStruct{
		Id:      &pb.PointId{PointIdOptions: &pb.PointId_Uuid{Uuid: point.ID}},
		Vectors: &pb.Vectors{VectorsOptions: &pb.Vectors_Vector{Vector: &pb.Vector{Data: point.Vector}}},
		Payload: payloadMap,
	}
	_, err = c.pointsClient.Upsert(ctx, &pb.UpsertPoints{
		CollectionName: collectionName,
		Points:         []*pb.PointStruct{qdrantPoint},
	})
	if err != nil {
		return WrapError(err, "failed to upsert point")
	}
	return nil
}

// UpsertPoints inserts or updates multiple points in a collection.
func (c *qdrantImpl) UpsertPoints(ctx context.Context, collectionName string, points []Point) error {
	if collectionName == "" {
		return ErrEmptyCollection
	}
	if len(points) == 0 {
		return nil
	}
	qdrantPoints := make([]*pb.PointStruct, 0, len(points))
	for _, point := range points {
		if point.ID == "" {
			return ErrInvalidPointID
		}
		if len(point.Vector) == 0 {
			return ErrInvalidVector
		}
		payloadMap, err := pb.TryValueMap(point.Payload)
		if err != nil {
			return WrapError(err, "failed to convert payload")
		}
		qdrantPoints = append(qdrantPoints, &pb.PointStruct{
			Id:      &pb.PointId{PointIdOptions: &pb.PointId_Uuid{Uuid: point.ID}},
			Vectors: &pb.Vectors{VectorsOptions: &pb.Vectors_Vector{Vector: &pb.Vector{Data: point.Vector}}},
			Payload: payloadMap,
		})
	}
	_, err := c.pointsClient.Upsert(ctx, &pb.UpsertPoints{
		CollectionName: collectionName,
		Points:         qdrantPoints,
	})
	if err != nil {
		return WrapError(err, "failed to upsert points")
	}
	return nil
}

// DeletePoint deletes a point from a collection.
func (c *qdrantImpl) DeletePoint(ctx context.Context, collectionName string, pointID string) error {
	if collectionName == "" {
		return ErrEmptyCollection
	}
	if pointID == "" {
		return ErrInvalidPointID
	}
	_, err := c.pointsClient.Delete(ctx, &pb.DeletePoints{
		CollectionName: collectionName,
		Points: &pb.PointsSelector{
			PointsSelectorOneOf: &pb.PointsSelector_Points{
				Points: &pb.PointsIdsList{
					Ids: []*pb.PointId{{PointIdOptions: &pb.PointId_Uuid{Uuid: pointID}}},
				},
			},
		},
	})
	if err != nil {
		return WrapError(err, "failed to delete point")
	}
	return nil
}

// GetPoint retrieves a point by ID.
func (c *qdrantImpl) GetPoint(ctx context.Context, collectionName string, pointID string) (*Point, error) {
	if collectionName == "" {
		return nil, ErrEmptyCollection
	}
	if pointID == "" {
		return nil, ErrInvalidPointID
	}
	resp, err := c.pointsClient.Get(ctx, &pb.GetPoints{
		CollectionName: collectionName,
		Ids:            []*pb.PointId{{PointIdOptions: &pb.PointId_Uuid{Uuid: pointID}}},
		WithPayload:    &pb.WithPayloadSelector{SelectorOptions: &pb.WithPayloadSelector_Enable{Enable: true}},
		WithVectors:    &pb.WithVectorsSelector{SelectorOptions: &pb.WithVectorsSelector_Enable{Enable: true}},
	})
	if err != nil {
		return nil, WrapError(err, "failed to get point")
	}
	if len(resp.Result) == 0 {
		return nil, ErrPointNotFound
	}
	result := resp.Result[0]
	var vector []float32
	if vectors := result.Vectors; vectors != nil {
		if v := vectors.GetVector(); v != nil {
			vector = v.Data
		}
	}
	payload := make(map[string]interface{})
	for key, value := range result.Payload {
		payload[key] = valueToInterface(value)
	}
	return &Point{ID: pointID, Vector: vector, Payload: payload}, nil
}

// Search performs a vector similarity search.
func (c *qdrantImpl) Search(ctx context.Context, collectionName string, vector []float32, limit uint64) ([]SearchResult, error) {
	if collectionName == "" {
		return nil, ErrEmptyCollection
	}
	if len(vector) == 0 {
		return nil, ErrInvalidVector
	}
	if limit == 0 {
		limit = DefaultSearchLimit
	}
	resp, err := c.pointsClient.Search(ctx, &pb.SearchPoints{
		CollectionName: collectionName,
		Vector:         vector,
		Limit:          limit,
		WithPayload:    &pb.WithPayloadSelector{SelectorOptions: &pb.WithPayloadSelector_Enable{Enable: true}},
	})
	if err != nil {
		return nil, WrapError(err, "failed to search")
	}
	return c.searchResultsFromHits(resp.Result), nil
}

// SearchWithFilter performs a vector similarity search with payload filter.
func (c *qdrantImpl) SearchWithFilter(ctx context.Context, collectionName string, vector []float32, limit uint64, filter *pb.Filter) ([]SearchResult, error) {
	if collectionName == "" {
		return nil, ErrEmptyCollection
	}
	if len(vector) == 0 {
		return nil, ErrInvalidVector
	}
	if limit == 0 {
		limit = DefaultSearchLimit
	}
	resp, err := c.pointsClient.Search(ctx, &pb.SearchPoints{
		CollectionName: collectionName,
		Vector:         vector,
		Limit:          limit,
		Filter:         filter,
		WithPayload:    &pb.WithPayloadSelector{SelectorOptions: &pb.WithPayloadSelector_Enable{Enable: true}},
	})
	if err != nil {
		return nil, WrapError(err, "failed to search with filter")
	}
	return c.searchResultsFromHits(resp.Result), nil
}

// SearchBatch performs multiple vector searches in a single request.
func (c *qdrantImpl) SearchBatch(ctx context.Context, collectionName string, vectors [][]float32, limit uint64) ([][]SearchResult, error) {
	if collectionName == "" {
		return nil, ErrEmptyCollection
	}
	if len(vectors) == 0 {
		return nil, ErrInvalidVector
	}
	if limit == 0 {
		limit = DefaultSearchLimit
	}
	searches := make([]*pb.SearchPoints, 0, len(vectors))
	for _, vector := range vectors {
		if len(vector) == 0 {
			return nil, ErrInvalidVector
		}
		searches = append(searches, &pb.SearchPoints{
			CollectionName: collectionName,
			Vector:         vector,
			Limit:          limit,
			WithPayload:    &pb.WithPayloadSelector{SelectorOptions: &pb.WithPayloadSelector_Enable{Enable: true}},
		})
	}
	resp, err := c.pointsClient.SearchBatch(ctx, &pb.SearchBatchPoints{
		CollectionName: collectionName,
		SearchPoints:   searches,
	})
	if err != nil {
		return nil, WrapError(err, "failed to batch search")
	}
	allResults := make([][]SearchResult, 0, len(resp.Result))
	for _, batchResult := range resp.Result {
		allResults = append(allResults, c.searchResultsFromHits(batchResult.Result))
	}
	return allResults, nil
}

// CountPoints returns the number of points in a collection.
func (c *qdrantImpl) CountPoints(ctx context.Context, collectionName string) (uint64, error) {
	if collectionName == "" {
		return 0, ErrEmptyCollection
	}
	resp, err := c.pointsClient.Count(ctx, &pb.CountPoints{CollectionName: collectionName})
	if err != nil {
		return 0, WrapError(err, "failed to count points")
	}
	if resp.Result == nil {
		return 0, nil
	}
	return resp.Result.Count, nil
}

// searchResultsFromHits maps Qdrant hit results to SearchResult slice.
func (c *qdrantImpl) searchResultsFromHits(hits []*pb.ScoredPoint) []SearchResult {
	results := make([]SearchResult, 0, len(hits))
	for _, hit := range hits {
		payload := make(map[string]interface{})
		for key, value := range hit.Payload {
			payload[key] = valueToInterface(value)
		}
		var id string
		if hit.Id != nil {
			id = hit.Id.GetUuid()
		}
		results = append(results, SearchResult{ID: id, Score: hit.Score, Payload: payload})
	}
	return results
}
