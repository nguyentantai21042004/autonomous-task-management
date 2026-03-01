package tools

import (
	"context"
	"fmt"

	"autonomous-task-management/internal/agent"
	"autonomous-task-management/internal/model"
	"autonomous-task-management/internal/task"
	pkgLog "autonomous-task-management/pkg/log"
)

// searchTasksTool implements semantic search over tasks.
type searchTasksTool struct {
	uc task.UseCase
	l  pkgLog.Logger
}

// NewSearchTasksTool creates the search_tasks agent tool for the task domain.
func NewSearchTasksTool(uc task.UseCase, l pkgLog.Logger) agent.Tool {
	return &searchTasksTool{uc: uc, l: l}
}

func (t *searchTasksTool) Name() string {
	return "search_tasks"
}

func (t *searchTasksTool) Description() string {
	return "Search for tasks using natural language query. Returns relevant tasks with similarity scores."
}

func (t *searchTasksTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "Natural language search query",
			},
			"limit": map[string]interface{}{
				"type":        "integer",
				"description": "Maximum number of results (default 10)",
			},
		},
		"required": []string{"query"},
	}
}

func (t *searchTasksTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query parameter is required")
	}

	limit := 10
	if l, ok := params["limit"].(float64); ok {
		limit = int(l)
	}

	sc := model.Scope{UserID: "agent"}
	output, err := t.uc.Search(ctx, sc, task.SearchInput{
		Query: query,
		Limit: limit,
	})
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	results := make([]map[string]interface{}, 0, len(output.Results))
	for _, r := range output.Results {
		results = append(results, map[string]interface{}{
			"memo_id":  r.MemoID,
			"memo_url": r.MemoURL,
			"content":  r.Content,
			"score":    r.Score,
		})
	}

	return map[string]interface{}{
		"count":   output.Count,
		"results": results,
	}, nil
}

var _ agent.Tool = (*searchTasksTool)(nil)
