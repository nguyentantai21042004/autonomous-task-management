package qdrant

import (
	"math"
	"sort"
	"strings"
)

// HybridResult represents a fused search result with RRF score.
type HybridResult struct {
	ScoredPoint
	RRFScore float64
}

// ReciprocateRankFusion merges two ranked lists using the RRF algorithm.
// k is the RRF constant (default 60, higher = less sensitive to top ranks).
// Both lists are ranked by score descending before fusion.
func ReciprocateRankFusion(dense, keyword []ScoredPoint, k int) []HybridResult {
	if k <= 0 {
		k = 60
	}

	// Map from point ID → RRF score accumulator
	scores := make(map[string]float64)
	// Map from ID → point (for payload preservation)
	points := make(map[string]ScoredPoint)

	addList := func(list []ScoredPoint) {
		// Sort descending by score
		sorted := make([]ScoredPoint, len(list))
		copy(sorted, list)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Score > sorted[j].Score
		})
		for rank, p := range sorted {
			scores[p.ID] += 1.0 / float64(k+rank+1)
			if _, exists := points[p.ID]; !exists {
				points[p.ID] = p
			}
		}
	}

	addList(dense)
	addList(keyword)

	// Build result slice
	results := make([]HybridResult, 0, len(scores))
	for id, score := range scores {
		results = append(results, HybridResult{
			ScoredPoint: points[id],
			RRFScore:    score,
		})
	}

	// Sort by RRF score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].RRFScore > results[j].RRFScore
	})

	return results
}

// KeywordScore computes a BM25-inspired keyword relevance score.
// Uses term frequency with saturation and IDF approximation.
// Returns a score in [0, 1] range (normalized by max possible score).
func KeywordScore(query, content string) float64 {
	const (
		k1 = 1.5 // term frequency saturation
		b  = 0.75 // length normalization
	)

	// Tokenize
	queryTerms := tokenize(query)
	contentTerms := tokenize(content)

	if len(queryTerms) == 0 || len(contentTerms) == 0 {
		return 0
	}

	// Term frequency in document
	tf := make(map[string]int)
	for _, t := range contentTerms {
		tf[t]++
	}

	docLen := float64(len(contentTerms))
	avgDocLen := 200.0 // assumed average doc length

	// IDF: simplified — count how many query terms appear in content
	score := 0.0
	matchedTerms := 0
	for _, term := range queryTerms {
		freq, ok := tf[term]
		if !ok {
			continue
		}
		matchedTerms++
		// BM25 TF component
		tfNorm := float64(freq) * (k1 + 1) / (float64(freq) + k1*(1-b+b*docLen/avgDocLen))
		// IDF approximation: log(1 + 1/queryLen) per matched term
		idf := math.Log(1 + 1.0/float64(len(queryTerms)))
		score += idf * tfNorm
	}

	if matchedTerms == 0 {
		return 0
	}

	// Normalize to [0,1] by dividing by max possible score
	maxPerTerm := (k1 + 1) * math.Log(1+1.0/float64(len(queryTerms)))
	maxScore := float64(len(queryTerms)) * maxPerTerm
	if maxScore == 0 {
		return 0
	}

	normalized := score / maxScore
	if normalized > 1 {
		normalized = 1
	}
	return normalized
}

// ApplyKeywordReranking scores a list of ScoredPoints by keyword relevance against query.
// Returns a new slice with Score replaced by KeywordScore for use in RRF.
func ApplyKeywordReranking(query string, points []ScoredPoint) []ScoredPoint {
	result := make([]ScoredPoint, len(points))
	for i, p := range points {
		content := extractContent(p.Payload)
		result[i] = ScoredPoint{
			ID:      p.ID,
			Score:   KeywordScore(query, content),
			Payload: p.Payload,
		}
	}
	return result
}

// extractContent pulls "content" string from payload, falling back to empty.
func extractContent(payload map[string]interface{}) string {
	if payload == nil {
		return ""
	}
	v, ok := payload["content"]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// tokenize lowercases and splits text into word tokens.
func tokenize(text string) []string {
	text = strings.ToLower(text)
	// Replace punctuation with spaces
	var b strings.Builder
	for _, r := range text {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r > 127 {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	parts := strings.Fields(b.String())
	// Filter short tokens
	result := parts[:0]
	for _, p := range parts {
		if len([]rune(p)) >= 2 {
			result = append(result, p)
		}
	}
	return result
}
