package qdrant

import (
	"context"

	pb "github.com/qdrant/go-client/qdrant"
)

// SearchGroups performs a search with grouping.
func (c *qdrantImpl) SearchGroups(ctx context.Context, collectionName string, vector []float32, limit uint64, groupBy string, groupLimit uint64, filter *pb.Filter) ([]GroupResult, error) {
	if collectionName == "" {
		return nil, ErrEmptyCollection
	}
	if len(vector) == 0 {
		return nil, ErrInvalidVector
	}
	if groupBy == "" {
		return nil, ErrInvalidVector // Reuse error or create new one
	}

	resp, err := c.pointsClient.SearchGroups(ctx, &pb.SearchPointGroups{
		CollectionName: collectionName,
		Vector:         vector,
		Limit:          uint32(limit), // Limit groups
		GroupBy:        groupBy,
		GroupSize:      uint32(groupLimit),
		Filter:         filter,
		WithPayload:    &pb.WithPayloadSelector{SelectorOptions: &pb.WithPayloadSelector_Enable{Enable: true}},
	})
	if err != nil {
		return nil, WrapError(err, "failed to search groups")
	}

	results := make([]GroupResult, 0, len(resp.Result.Groups))
	for _, group := range resp.Result.Groups {
		hits := c.searchResultsFromHits(group.Hits)

		var id interface{}
		if group.Id.GetStringValue() != "" {
			id = group.Id.GetStringValue()
		} else {
			id = group.Id.GetUnsignedValue()
		}

		results = append(results, GroupResult{
			ID:    id,
			Hits:  hits,
			Count: 0, // Qdrant Group response might not have total count per group readily available in this struct unless configured
		})
	}

	return results, nil
}
