package voyage

// EmbedRequest is the request body for the embeddings API.
type EmbedRequest struct {
	Input []string `json:"input"` // Texts to embed
	Model string   `json:"model"` // Model name (e.g., "voyage-3")
}

// EmbedResponse is the response from the embeddings API.
type EmbedResponse struct {
	Object string          `json:"object"` // "list"
	Data   []EmbeddingData `json:"data"`
	Model  string          `json:"model"`
	Usage  UsageInfo       `json:"usage"`
}

// EmbeddingData contains a single embedding vector.
type EmbeddingData struct {
	Object    string    `json:"object"`    // "embedding"
	Embedding []float32 `json:"embedding"` // Vector
	Index     int       `json:"index"`     // Position in input array
}

// UsageInfo contains token usage statistics.
type UsageInfo struct {
	TotalTokens int `json:"total_tokens"`
}

// ErrorResponse is the error response from Voyage API.
type ErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}
