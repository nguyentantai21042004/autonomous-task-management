package qdrant

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"autonomous-task-management/internal/model"
	"autonomous-task-management/internal/task/repository"
	"autonomous-task-management/pkg/indexer"
	pkgLog "autonomous-task-management/pkg/log"
	pkgQdrant "autonomous-task-management/pkg/qdrant"
	"autonomous-task-management/pkg/voyage"
)

// tagRegex matches hashtags like #repo/myproject, #pr/123, #issue/456.
var tagRegex = regexp.MustCompile(`#[a-zA-Z0-9_/]+`)

type implRepository struct {
	client         *pkgQdrant.Client
	embedder       *voyage.Client
	collectionName string
	l              pkgLog.Logger
}

// New creates a new Qdrant repository.
func New(client *pkgQdrant.Client, embedder *voyage.Client, collectionName string, l pkgLog.Logger) repository.VectorRepository {
	return &implRepository{
		client:         client,
		embedder:       embedder,
		collectionName: collectionName,
		l:              l,
	}
}

// EmbedTask generates embedding and stores in Qdrant.
func (r *implRepository) EmbedTask(ctx context.Context, task model.Task) error {
	// Build text to embed: title + tags + summary (NOT full content)
	textToEmbed := buildEmbeddingText(task)

	// Generate embedding
	vectors, err := r.embedder.Embed(ctx, []string{textToEmbed})
	if err != nil || len(vectors) == 0 {
		r.l.Errorf(ctx, "qdrant repository: failed to generate embedding: %v", err)
		return fmt.Errorf("failed to generate embedding: %w", err)
	}
	vector := vectors[0]

	// CRITICAL FIX: Convert Memos ID to UUID for Qdrant
	// Qdrant requires ID to be UUID or uint64, NOT arbitrary string
	qdrantID := memoIDToUUID(task.ID)

	// NEW: Extract tags from content
	tags := extractTags(task.Content)

	// Create point
	point := pkgQdrant.Point{
		ID:     qdrantID, // UUID string
		Vector: vector,
		Payload: map[string]interface{}{
			"memo_id":     task.ID, // Store original Memos ID in payload
			"memo_url":    task.MemoURL,
			"content":     task.Content,
			"tags":        tags,
			"create_time": task.CreateTime,
			"update_time": task.UpdateTime,
		},
	}

	// Upsert to Qdrant
	req := pkgQdrant.UpsertPointsRequest{
		Points: []pkgQdrant.Point{point},
	}

	if err := r.client.UpsertPoints(ctx, r.collectionName, req); err != nil {
		r.l.Errorf(ctx, "qdrant repository: failed to upsert point: %v", err)
		return fmt.Errorf("failed to upsert point: %w", err)
	}

	r.l.Infof(ctx, "qdrant repository: embedded task %s (qdrant_id=%s)", task.ID, qdrantID)
	return nil
}

// SearchTasks performs parallel hybrid search: dense vector + full-text, fused via RRF.
// Cả 2 tracks chạy song song qua goroutine — giảm latency đáng kể so với sequential.
func (r *implRepository) SearchTasks(ctx context.Context, opt repository.SearchTasksOptions) ([]repository.SearchResult, error) {
	// Generate query embedding
	vectors, err := r.embedder.Embed(ctx, []string{opt.Query})
	if err != nil || len(vectors) == 0 {
		r.l.Errorf(ctx, "qdrant repository: failed to generate query embedding: %v", err)
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}
	queryVector := vectors[0]

	// --- Run dense + text search in parallel ---
	type trackResult struct {
		points []pkgQdrant.ScoredPoint
		err    error
	}
	denseCh := make(chan trackResult, 1)
	textCh := make(chan trackResult, 1)

	// Track 1: Dense vector search
	go func() {
		req := pkgQdrant.SearchRequest{
			Vector:      queryVector,
			Limit:       opt.Limit,
			WithPayload: true,
		}
		resp, err := r.client.SearchPoints(ctx, r.collectionName, req)
		if err != nil {
			denseCh <- trackResult{err: err}
			return
		}
		denseCh <- trackResult{points: resp.Result}
	}()

	// Track 2: Full-text scroll search using Qdrant text match filter
	go func() {
		// Build text match filter: match any word in the query against "content" field.
		// Requires a text index on "content" field (created once via CreatePayloadIndex).
		filter := map[string]interface{}{
			"must": []map[string]interface{}{
				{
					"key": "content",
					"match": map[string]interface{}{
						"text": opt.Query,
					},
				},
			},
		}
		req := pkgQdrant.ScrollRequest{
			Filter:      filter,
			Limit:       opt.Limit,
			WithPayload: true,
			WithVector:  false,
		}
		resp, err := r.client.ScrollPoints(ctx, r.collectionName, req)
		if err != nil {
			// Text index may not exist yet — graceful degradation, not a fatal error
			r.l.Warnf(ctx, "qdrant repository: text search failed (index missing?), skipping: %v", err)
			textCh <- trackResult{points: nil}
			return
		}
		textCh <- trackResult{points: resp.Result.Points}
	}()

	denseRes := <-denseCh
	textRes := <-textCh

	if denseRes.err != nil {
		r.l.Errorf(ctx, "qdrant repository: dense search failed: %v", denseRes.err)
		return nil, fmt.Errorf("failed to search: %w", denseRes.err)
	}

	// RRF fusion of both tracks
	fused := pkgQdrant.ReciprocateRankFusion(denseRes.points, textRes.points, 60)

	// Convert fused results → SearchResult, extracting memo_id from payload
	results := make([]repository.SearchResult, 0, len(fused))
	for _, hr := range fused {
		memoIDRaw, exists := hr.Payload["memo_id"]
		if !exists {
			r.l.Errorf(ctx, "qdrant repository: memo_id missing in payload for point %v", hr.ID)
			continue
		}
		memoID, ok := memoIDRaw.(string)
		if !ok {
			r.l.Errorf(ctx, "qdrant repository: memo_id type assertion failed for point %v", hr.ID)
			continue
		}
		results = append(results, repository.SearchResult{
			MemoID:  memoID,
			Score:   hr.RRFScore,
			Payload: hr.Payload,
		})
	}

	r.l.Infof(ctx, "qdrant repository: hybrid search found %d results (dense=%d, text=%d) for query %q",
		len(results), len(denseRes.points), len(textRes.points), opt.Query)
	return results, nil
}

// SearchTasksWithFilter performs semantic search with payload filtering.
func (r *implRepository) SearchTasksWithFilter(ctx context.Context, opt repository.SearchTasksOptions) ([]repository.SearchResult, error) {
	dummyVector := make([]float32, 1024)

	var shouldConditions []map[string]interface{}
	for _, cond := range opt.Filter.Should {
		shouldConditions = append(shouldConditions, map[string]interface{}{
			"key": cond.Key,
			"match": map[string]interface{}{
				"any": cond.Match.Values,
			},
		})
	}

	filter := map[string]interface{}{}
	if len(shouldConditions) > 0 {
		filter["should"] = shouldConditions
	}

	searchReq := pkgQdrant.SearchRequest{
		Vector:      dummyVector,
		Limit:       opt.Limit,
		WithPayload: true,
		Filter:      filter,
	}

	resp, err := r.client.SearchPoints(ctx, r.collectionName, searchReq)
	if err != nil {
		r.l.Errorf(ctx, "qdrant repository: failed to search with filter: %v", err)
		return nil, fmt.Errorf("failed to search with filter: %w", err)
	}

	results := make([]repository.SearchResult, 0, len(resp.Result))
	for _, scored := range resp.Result {
		memoIDRaw, exists := scored.Payload["memo_id"]
		if !exists {
			continue
		}
		memoID, ok := memoIDRaw.(string)
		if !ok {
			continue
		}
		results = append(results, repository.SearchResult{
			MemoID:  memoID,
			Score:   scored.Score,
			Payload: scored.Payload,
		})
	}

	r.l.Infof(ctx, "qdrant repository: found %d results using filter", len(results))
	return results, nil
}

// DeleteTask removes a task from Qdrant.
func (r *implRepository) DeleteTask(ctx context.Context, taskID string) error {
	// Convert Memos ID to UUID
	qdrantID := memoIDToUUID(taskID)

	if err := r.client.DeletePoints(ctx, r.collectionName, []string{qdrantID}); err != nil {
		r.l.Errorf(ctx, "qdrant repository: failed to delete point: %v", err)
		return fmt.Errorf("failed to delete point: %w", err)
	}

	r.l.Infof(ctx, "qdrant repository: deleted task %s (qdrant_id=%s)", taskID, qdrantID)
	return nil
}

// memoIDToUUID converts Memos ID (arbitrary string) to UUID for Qdrant.
// Qdrant requires ID to be UUID or uint64, NOT arbitrary string.
// We use deterministic UUID v5 (namespace + name) to ensure same ID for same memo.
func memoIDToUUID(memoID string) string {
	// Use UUID v5 with a custom namespace
	// This ensures: same memoID → same UUID (deterministic)
	namespace := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8") // DNS namespace
	return uuid.NewSHA1(namespace, []byte(memoID)).String()
}

// buildEmbeddingText constructs enriched text for embedding from task.
// V2.0: Dung Contextual Enrichment thay vi chi embed title+tags.
// Ket qua: vector capture duoc "tuan nay", "ngay mai", "qua han" →
// query "deadline tuan nay" match chinh xac hon.
func buildEmbeddingText(task model.Task) string {
	return indexer.EnrichTaskContent(task.Content, task.Tags, "Asia/Ho_Chi_Minh")
}

// extractTags extracts hashtags from markdown content.
// Matches patterns like: #repo/myproject, #pr/123, #issue/456
func extractTags(content string) []string {
	var tags []string
	seen := make(map[string]bool)

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		matches := tagRegex.FindAllString(line, -1)

		for _, tag := range matches {
			if !seen[tag] {
				tags = append(tags, tag)
				seen[tag] = true
			}
		}
	}

	return tags
}
