package qdrant

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ---------------------------------------------------------------------------
// ReciprocateRankFusion tests
// ---------------------------------------------------------------------------

func TestRRF_EmptyInputs(t *testing.T) {
	results := ReciprocateRankFusion(nil, nil, 60)
	assert.Empty(t, results)
}

func TestRRF_OnlyDense(t *testing.T) {
	dense := []ScoredPoint{
		{ID: "a", Score: 0.9},
		{ID: "b", Score: 0.7},
	}
	results := ReciprocateRankFusion(dense, nil, 60)
	assert.Len(t, results, 2)
	// "a" ranked 1st in dense → higher RRF score
	assert.Equal(t, "a", results[0].ID)
}

func TestRRF_OnlyKeyword(t *testing.T) {
	keyword := []ScoredPoint{
		{ID: "x", Score: 0.8},
		{ID: "y", Score: 0.5},
	}
	results := ReciprocateRankFusion(nil, keyword, 60)
	assert.Len(t, results, 2)
	assert.Equal(t, "x", results[0].ID)
}

func TestRRF_Fusion_BoostsOverlap(t *testing.T) {
	// "b" appears in both lists → should get boosted score
	dense := []ScoredPoint{
		{ID: "a", Score: 0.95},
		{ID: "b", Score: 0.80},
	}
	keyword := []ScoredPoint{
		{ID: "b", Score: 0.90}, // "b" top keyword result
		{ID: "c", Score: 0.70},
	}
	results := ReciprocateRankFusion(dense, keyword, 60)

	// Find positions
	pos := make(map[string]int)
	for i, r := range results {
		pos[r.ID] = i
	}

	// "b" appears in both → should beat "a" which only appears in dense
	assert.Less(t, pos["b"], pos["a"], "b should rank higher than a due to overlap boost")
}

func TestRRF_DefaultK(t *testing.T) {
	// k=0 should use default k=60
	dense := []ScoredPoint{{ID: "a", Score: 0.9}}
	results := ReciprocateRankFusion(dense, nil, 0)
	assert.NotEmpty(t, results)
	assert.InDelta(t, 1.0/61.0, results[0].RRFScore, 0.001) // 1/(60+1)
}

func TestRRF_PreservesPayload(t *testing.T) {
	payload := map[string]interface{}{"memo_id": "memo_123", "content": "test task"}
	dense := []ScoredPoint{
		{ID: "a", Score: 0.9, Payload: payload},
	}
	results := ReciprocateRankFusion(dense, nil, 60)
	assert.Equal(t, payload, results[0].Payload)
}

// ---------------------------------------------------------------------------
// KeywordScore tests
// ---------------------------------------------------------------------------

func TestKeywordScore_EmptyQuery(t *testing.T) {
	score := KeywordScore("", "some content")
	assert.Equal(t, 0.0, score)
}

func TestKeywordScore_EmptyContent(t *testing.T) {
	score := KeywordScore("query", "")
	assert.Equal(t, 0.0, score)
}

func TestKeywordScore_PerfectMatch(t *testing.T) {
	score := KeywordScore("deadline", "deadline for project")
	assert.Greater(t, score, 0.0)
	assert.LessOrEqual(t, score, 1.0)
}

func TestKeywordScore_NoMatch(t *testing.T) {
	score := KeywordScore("meeting", "deploy staging server")
	assert.Equal(t, 0.0, score)
}

func TestKeywordScore_PartialMatch_LowerThanFullMatch(t *testing.T) {
	full := KeywordScore("review pr deadline", "review pr deadline backend")
	partial := KeywordScore("review pr deadline", "review backend")
	assert.Greater(t, full, partial)
}

func TestKeywordScore_CaseInsensitive(t *testing.T) {
	score1 := KeywordScore("PR Review", "pr review backend")
	score2 := KeywordScore("pr review", "PR Review Backend")
	assert.InDelta(t, score1, score2, 0.001)
}

// ---------------------------------------------------------------------------
// ApplyKeywordReranking tests
// ---------------------------------------------------------------------------

func TestApplyKeywordReranking_ScoresFromContent(t *testing.T) {
	points := []ScoredPoint{
		{ID: "1", Score: 0.9, Payload: map[string]interface{}{"content": "review PR #123 backend deadline"}},
		{ID: "2", Score: 0.85, Payload: map[string]interface{}{"content": "deploy staging server"}},
	}

	result := ApplyKeywordReranking("PR review", points)
	assert.Len(t, result, 2)

	// "PR review" matches first doc better
	scores := make(map[string]float64)
	for _, r := range result {
		scores[r.ID] = r.Score
	}
	assert.Greater(t, scores["1"], scores["2"])
}

func TestApplyKeywordReranking_EmptyPayload(t *testing.T) {
	points := []ScoredPoint{
		{ID: "1", Score: 0.9, Payload: nil},
	}
	result := ApplyKeywordReranking("query", points)
	assert.Len(t, result, 1)
	assert.Equal(t, 0.0, result[0].Score)
}

// ---------------------------------------------------------------------------
// tokenize tests
// ---------------------------------------------------------------------------

func TestTokenize_Basic(t *testing.T) {
	tokens := tokenize("Review PR #123")
	assert.Contains(t, tokens, "review")
	assert.Contains(t, tokens, "123")
	// Short tokens like "PR" should be included (2 chars)
	assert.Contains(t, tokens, "pr")
}

func TestTokenize_PunctuationStripped(t *testing.T) {
	tokens := tokenize("hello, world!")
	assert.Contains(t, tokens, "hello")
	assert.Contains(t, tokens, "world")
}

func TestTokenize_VietnameseSupported(t *testing.T) {
	// Vietnamese chars are > 127 so should be preserved
	tokens := tokenize("deadline ngày mai")
	assert.NotEmpty(t, tokens)
}
