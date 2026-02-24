package automation

import (
	"context"
	"fmt"
	"strings"

	"autonomous-task-management/internal/model"
	"autonomous-task-management/internal/task/repository"
	pkgLog "autonomous-task-management/pkg/log"
)

// TaskMatcher matches webhook events to tasks
type TaskMatcher struct {
	memosRepo  repository.MemosRepository
	vectorRepo repository.VectorRepository
	l          pkgLog.Logger
}

func NewTaskMatcher(memosRepo repository.MemosRepository, vectorRepo repository.VectorRepository, l pkgLog.Logger) *TaskMatcher {
	return &TaskMatcher{
		memosRepo:  memosRepo,
		vectorRepo: vectorRepo,
		l:          l,
	}
}

// FindMatchingTasks finds tasks that match the webhook event
func (m *TaskMatcher) FindMatchingTasks(ctx context.Context, event model.WebhookEvent) ([]TaskMatch, error) {
	criteria := m.buildMatchCriteria(event)

	// Strategy 1: Tag-based search (fast, precise)
	tagMatches, err := m.searchByTags(ctx, criteria)
	if err != nil {
		m.l.Warnf(ctx, "Tag-based search failed: %v", err)
	}

	// Strategy 2: Keyword-based search (flexible, broader)
	keywordMatches, err := m.searchByKeywords(ctx, criteria)
	if err != nil {
		m.l.Warnf(ctx, "Keyword-based search failed: %v", err)
	}

	// Merge and deduplicate results
	matches := m.mergeMatches(tagMatches, keywordMatches)

	m.l.Infof(ctx, "Found %d matching tasks for event %s", len(matches), event.EventType)
	return matches, nil
}

// buildMatchCriteria builds search criteria from webhook event
func (m *TaskMatcher) buildMatchCriteria(event model.WebhookEvent) MatchCriteria {
	criteria := MatchCriteria{
		Repository: event.Repository,
		Tags:       []string{},
		Keywords:   []string{},
	}

	// Add repository tag
	if event.Repository != "" {
		// Convert "user/repo" → "#repo/repo"
		parts := strings.Split(event.Repository, "/")
		if len(parts) == 2 {
			criteria.Tags = append(criteria.Tags, "#repo/"+parts[1])
		}
	}

	// Add PR/Issue tag
	if event.PRNumber > 0 {
		criteria.Tags = append(criteria.Tags, fmt.Sprintf("#pr/%d", event.PRNumber))
		criteria.Keywords = append(criteria.Keywords, fmt.Sprintf("PR #%d", event.PRNumber))
		criteria.Keywords = append(criteria.Keywords, fmt.Sprintf("#%d", event.PRNumber))
	}

	if event.IssueNumber > 0 {
		criteria.Tags = append(criteria.Tags, fmt.Sprintf("#issue/%d", event.IssueNumber))
		criteria.Keywords = append(criteria.Keywords, fmt.Sprintf("Issue #%d", event.IssueNumber))
		criteria.Keywords = append(criteria.Keywords, fmt.Sprintf("#%d", event.IssueNumber))
	}

	// Add branch keyword
	if event.Branch != "" {
		criteria.Keywords = append(criteria.Keywords, event.Branch)
	}

	return criteria
}

// searchByTags searches tasks by EXACT tag match using Qdrant filter
func (m *TaskMatcher) searchByTags(ctx context.Context, criteria MatchCriteria) ([]TaskMatch, error) {
	if len(criteria.Tags) == 0 {
		return nil, nil
	}

	m.l.Infof(ctx, "Searching by exact tags: %v", criteria.Tags)

	// Use Qdrant payload filter for EXACT tag matching
	// This prevents false positives like #pr/123 matching #pr/124
	results, err := m.vectorRepo.SearchTasksWithFilter(ctx, repository.SearchTasksOptions{
		Filter: repository.PayloadFilter{
			// Must match ANY of the tags (OR condition)
			Should: []repository.Condition{
				{
					Key:   "tags",
					Match: repository.MatchAny{Values: criteria.Tags},
				},
			},
		},
		Limit: 10,
	})
	if err != nil {
		return nil, err
	}

	matches := make([]TaskMatch, 0, len(results))
	for _, result := range results {
		// Fetch full task content
		task, err := m.memosRepo.GetTask(ctx, result.MemoID)
		if err != nil {
			m.l.Warnf(ctx, "Failed to fetch task %s: %v", result.MemoID, err)
			continue
		}

		matches = append(matches, TaskMatch{
			TaskID:      result.MemoID,
			Content:     task.Content,
			MatchScore:  1.0, // Exact match = 100%
			MatchReason: fmt.Sprintf("exact-tag: %v", criteria.Tags),
		})
	}

	return matches, nil
}

// searchByKeywords searches tasks by keywords using SEMANTIC search
// This is OK to use vector search because we want fuzzy matching for keywords
func (m *TaskMatcher) searchByKeywords(ctx context.Context, criteria MatchCriteria) ([]TaskMatch, error) {
	if len(criteria.Keywords) == 0 {
		return nil, nil
	}

	// Build search query from keywords
	query := strings.Join(criteria.Keywords, " OR ")

	m.l.Infof(ctx, "Searching by semantic keywords: %s", query)

	results, err := m.vectorRepo.SearchTasks(ctx, repository.SearchTasksOptions{
		Query: query,
		Limit: 10,
	})
	if err != nil {
		return nil, err
	}

	matches := make([]TaskMatch, 0, len(results))
	for _, result := range results {
		// Fetch full task content
		task, err := m.memosRepo.GetTask(ctx, result.MemoID)
		if err != nil {
			m.l.Warnf(ctx, "Failed to fetch task %s: %v", result.MemoID, err)
			continue
		}

		// Verify keyword actually exists in content (avoid false positives)
		contentLower := strings.ToLower(task.Content)
		matched := false
		for _, keyword := range criteria.Keywords {
			if strings.Contains(contentLower, strings.ToLower(keyword)) {
				matched = true
				break
			}
		}

		if !matched {
			continue
		}

		matches = append(matches, TaskMatch{
			TaskID:      result.MemoID,
			Content:     task.Content,
			MatchScore:  result.Score,
			MatchReason: "semantic-keyword",
		})
	}

	return matches, nil
}

// mergeMatches merges and deduplicates task matches
func (m *TaskMatcher) mergeMatches(tagMatches, keywordMatches []TaskMatch) []TaskMatch {
	seen := make(map[string]bool)
	merged := make([]TaskMatch, 0)

	// Add tag matches first (higher priority - exact match)
	for _, match := range tagMatches {
		if !seen[match.TaskID] {
			merged = append(merged, match)
			seen[match.TaskID] = true
		}
	}

	// Add keyword matches (lower priority - semantic match)
	for _, match := range keywordMatches {
		if !seen[match.TaskID] {
			merged = append(merged, match)
			seen[match.TaskID] = true
		}
	}

	return merged
}
