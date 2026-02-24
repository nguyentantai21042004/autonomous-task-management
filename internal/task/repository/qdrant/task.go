package qdrant

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"autonomous-task-management/internal/model"
	"autonomous-task-management/internal/task/repository"
	pkgLog "autonomous-task-management/pkg/log"
	pkgQdrant "autonomous-task-management/pkg/qdrant"
	"autonomous-task-management/pkg/voyage"
)

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

// SearchTasks performs semantic search.
func (r *implRepository) SearchTasks(ctx context.Context, opt repository.SearchTasksOptions) ([]repository.SearchResult, error) {
	// Generate query embedding
	vectors, err := r.embedder.Embed(ctx, []string{opt.Query})
	if err != nil || len(vectors) == 0 {
		r.l.Errorf(ctx, "qdrant repository: failed to generate query embedding: %v", err)
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}
	queryVector := vectors[0]

	// Build search request
	searchReq := pkgQdrant.SearchRequest{
		Vector:      queryVector,
		Limit:       opt.Limit,
		WithPayload: true, // CRITICAL: Need payload to get original memo_id
	}

	// Add filters if provided
	if len(opt.Tags) > 0 {
		// Implement tag filtering
	}

	// Search in Qdrant
	resp, err := r.client.SearchPoints(ctx, r.collectionName, searchReq)
	if err != nil {
		r.l.Errorf(ctx, "qdrant repository: failed to search: %v", err)
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	// Convert to SearchResult
	// CRITICAL: Extract memo_id from payload (NOT from Qdrant ID)
	results := make([]repository.SearchResult, 0, len(resp.Result))
	for _, scored := range resp.Result {
		// Safe type assertion with detailed error logging
		// Get original Memos ID from payload
		memoIDRaw, exists := scored.Payload["memo_id"]
		if !exists {
			r.l.Errorf(ctx, "qdrant repository: memo_id missing in payload for point %v, payload: %+v",
				scored.ID, scored.Payload)
			continue
		}

		memoID, ok := memoIDRaw.(string)
		if !ok {
			r.l.Errorf(ctx, "qdrant repository: memo_id type assertion failed for point %v, got type %T, value: %v",
				scored.ID, memoIDRaw, memoIDRaw)
			continue
		}

		results = append(results, repository.SearchResult{
			MemoID:  memoID, // Use original Memos ID, not Qdrant UUID
			Score:   scored.Score,
			Payload: scored.Payload,
		})
	}

	r.l.Infof(ctx, "qdrant repository: found %d results for query %q", len(results), opt.Query)
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

// buildEmbeddingText constructs optimized text for embedding from task.
// OPTIMIZATION: Embed only title + tags + summary, NOT full content.
// Full content dilutes semantic density and reduces search accuracy.
func buildEmbeddingText(task model.Task) string {
	var parts []string

	// NITPICK FIX: Strip markdown code blocks first
	// Prevents code snippets from polluting semantic content
	content := stripMarkdownCodeBlocks(task.Content)

	// Extract title (first non-empty line, remove markdown)
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			// Remove markdown formatting
			title := strings.ReplaceAll(line, "**", "")
			title = strings.ReplaceAll(title, "*", "")
			parts = append(parts, title)
			break
		}
	}

	// Extract tags (lines starting with #)
	var tags []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") {
			tags = append(tags, line)
		}
	}
	if len(tags) > 0 {
		parts = append(parts, strings.Join(tags, " "))
	}

	// Extract first 2-3 sentences as summary (skip title line)
	var summaryLines []string
	skipFirst := true
	sentenceCount := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if skipFirst {
			skipFirst = false
			continue
		}
		summaryLines = append(summaryLines, line)
		// Count sentences (rough approximation)
		sentenceCount += strings.Count(line, ".") + strings.Count(line, "!") + strings.Count(line, "?")
		if sentenceCount >= 2 {
			break
		}
	}
	if len(summaryLines) > 0 {
		parts = append(parts, strings.Join(summaryLines, " "))
	}

	// Combine: title + tags + summary
	result := strings.Join(parts, "\n")

	// Limit to 1000 chars to avoid embedding API limits
	if len(result) > 1000 {
		result = result[:1000]
	}

	return result
}

// stripMarkdownCodeBlocks removes code blocks (```...```) from text.
// NITPICK FIX: Prevents code snippets from polluting embeddings.
func stripMarkdownCodeBlocks(text string) string {
	// Remove code blocks: ```language\n...\n``` or ```\n...\n```
	re := regexp.MustCompile("(?s)```[a-z]*\\n.*?\\n```")
	return re.ReplaceAllString(text, "")
}

// extractTags extracts hashtags from markdown content.
// Matches patterns like: #repo/myproject, #pr/123, #issue/456
func extractTags(content string) []string {
	var tags []string
	seen := make(map[string]bool)

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		re := regexp.MustCompile(`#[a-zA-Z0-9_/]+`)
		matches := re.FindAllString(line, -1)

		for _, tag := range matches {
			if !seen[tag] {
				tags = append(tags, tag)
				seen[tag] = true
			}
		}
	}

	return tags
}
