// Package retrieval provides a RAG retrieval pipeline for OpenBotStack.
//
// Implements document indexing, keyword-based retrieval, and context
// assembly for retrieval-augmented generation. Vector similarity search
// is reserved for the runtime layer.
package retrieval

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Document represents a retrievable document.
type Document struct {
	ID       string
	Content  string
	Metadata map[string]string
}

// Chunk represents a portion of a document.
type Chunk struct {
	ID         string
	DocumentID string
	Content    string
	Embedding  []float32
	Metadata   map[string]string
	Score      float64
}

// RetrievalResult contains retrieval query results.
type RetrievalResult struct {
	Query    string
	Chunks   []Chunk
	Context  string
	Duration time.Duration
}

// Pipeline orchestrates document retrieval for RAG.
type Pipeline struct {
	docs   map[string]Document
	chunks map[string][]Chunk
}

// NewPipeline creates a new retrieval pipeline.
func NewPipeline() *Pipeline {
	return &Pipeline{
		docs:   make(map[string]Document),
		chunks: make(map[string][]Chunk),
	}
}

// Index adds a document to the retrieval index.
// Splits into sentence-based chunks. Overwrites existing document with same ID.
func (p *Pipeline) Index(ctx context.Context, doc Document) error {
	if ctx == nil {
		return fmt.Errorf("retrieval: context is required")
	}
	if doc.ID == "" {
		return fmt.Errorf("retrieval: document ID is required")
	}
	if doc.Content == "" {
		return fmt.Errorf("retrieval: document content is required")
	}

	// Deep-copy metadata to prevent caller mutation from corrupting stored data
	metadata := make(map[string]string, len(doc.Metadata))
	for k, v := range doc.Metadata {
		metadata[k] = v
	}
	doc.Metadata = metadata

	chunks := splitIntoChunks(doc)
	p.docs[doc.ID] = doc
	p.chunks[doc.ID] = chunks

	return nil
}

// Retrieve finds relevant chunks for a query using keyword overlap.
func (p *Pipeline) Retrieve(ctx context.Context, query string, topK int) (*RetrievalResult, error) {
	if ctx == nil {
		return nil, fmt.Errorf("retrieval: context is required")
	}
	if query == "" {
		return nil, fmt.Errorf("retrieval: query is required")
	}
	if topK <= 0 {
		return nil, fmt.Errorf("retrieval: topK must be positive")
	}

	start := time.Now()

	queryTokens := tokenize(query)
	var allChunks []Chunk
	for _, chunks := range p.chunks {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		allChunks = append(allChunks, chunks...)
	}

	for i := range allChunks {
		chunkTokens := tokenize(allChunks[i].Content)
		allChunks[i].Score = overlapScore(queryTokens, chunkTokens)
	}

	sort.Slice(allChunks, func(i, j int) bool {
		return allChunks[i].Score > allChunks[j].Score
	})

	if topK > len(allChunks) {
		topK = len(allChunks)
	}

	result := &RetrievalResult{
		Query:    query,
		Chunks:   allChunks[:topK],
		Duration: time.Since(start),
	}

	return result, nil
}

// AssembleContext builds an LLM-ready context string from chunks.
func (p *Pipeline) AssembleContext(chunks []Chunk, maxLength int) string {
	if len(chunks) == 0 {
		return ""
	}

	var parts []string
	totalLen := 0
	for _, c := range chunks {
		if maxLength > 0 && totalLen+len(c.Content) > maxLength {
			remaining := maxLength - totalLen
			if remaining > 0 {
				parts = append(parts, c.Content[:remaining])
			}
			break
		}
		parts = append(parts, c.Content)
		totalLen += len(c.Content)
	}

	return strings.Join(parts, "\n\n---\n\n")
}

// splitIntoChunks splits a document into sentence-based chunks.
func splitIntoChunks(doc Document) []Chunk {
	sentences := splitSentences(doc.Content)
	chunks := make([]Chunk, 0, len(sentences))
	for i, s := range sentences {
		chunks = append(chunks, Chunk{
			ID:         fmt.Sprintf("%s-chunk-%d", doc.ID, i),
			DocumentID: doc.ID,
			Content:    s,
			Metadata:   doc.Metadata,
		})
	}
	return chunks
}

// splitSentences splits text into sentences.
func splitSentences(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}

	// Split on sentence-ending punctuation followed by space
	replacements := []string{". ", "! ", "? "}
	for _, r := range replacements {
		text = strings.ReplaceAll(text, r, r[:1]+"\n")
	}

	var result []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}

	if len(result) == 0 {
		result = append(result, text)
	}

	return result
}

// tokenize splits text into lowercase tokens.
func tokenize(text string) map[string]bool {
	words := strings.Fields(strings.ToLower(text))
	tokens := make(map[string]bool, len(words))
	for _, w := range words {
		if len(w) > 1 {
			tokens[w] = true
		}
	}
	return tokens
}

// overlapScore computes keyword overlap ratio between query and chunk tokens.
func overlapScore(queryTokens, chunkTokens map[string]bool) float64 {
	if len(queryTokens) == 0 {
		return 0
	}
	overlap := 0
	for q := range queryTokens {
		if chunkTokens[q] {
			overlap++
		}
	}
	return float64(overlap) / float64(len(queryTokens))
}
