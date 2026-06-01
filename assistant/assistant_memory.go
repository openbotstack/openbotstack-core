package assistant

// SearchResult represents a single entry found during a semantic search.
type SearchResult struct {
	Content []byte
	Score   float32
}
