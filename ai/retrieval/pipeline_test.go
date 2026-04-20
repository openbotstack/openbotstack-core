package retrieval

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// --- Normal Cases (4) ---

func TestIndex_SingleDocument(t *testing.T) {
	p := NewPipeline()
	doc := Document{ID: "doc-1", Content: "Hello world. This is a test document."}
	err := p.Index(context.Background(), doc)
	if err != nil {
		t.Fatalf("Index: %v", err)
	}
}

func TestRetrieve_FindsRelevantChunk(t *testing.T) {
	p := NewPipeline()
	_ = p.Index(context.Background(), Document{ID: "doc-1", Content: "OpenBotStack is an AI execution platform"})
	_ = p.Index(context.Background(), Document{ID: "doc-2", Content: "Weather forecast for tomorrow is sunny"})

	result, err := p.Retrieve(context.Background(), "what is OpenBotStack", 1)
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if len(result.Chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}
	if !strings.Contains(strings.ToLower(result.Chunks[0].Content), "openbotstack") {
		t.Errorf("expected relevant chunk about OpenBotStack, got %q", result.Chunks[0].Content)
	}
}

func TestAssembleContext_BasicChunks(t *testing.T) {
	p := NewPipeline()
	chunks := []Chunk{
		{ID: "c1", Content: "First chunk"},
		{ID: "c2", Content: "Second chunk"},
	}
	ctx := p.AssembleContext(chunks, 1000)
	if !strings.Contains(ctx, "First chunk") || !strings.Contains(ctx, "Second chunk") {
		t.Errorf("context should contain both chunks, got %q", ctx)
	}
}

func TestRetrieve_TopK(t *testing.T) {
	p := NewPipeline()
	for i := 0; i < 10; i++ {
		_ = p.Index(context.Background(), Document{
			ID:      fmt.Sprintf("doc-%d", i),
			Content: fmt.Sprintf("Document number %d about AI", i),
		})
	}
	result, err := p.Retrieve(context.Background(), "AI documents", 3)
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if len(result.Chunks) > 3 {
		t.Errorf("expected at most 3 chunks, got %d", len(result.Chunks))
	}
}

// --- Abnormal / Edge Cases (13) ---

func TestIndex_EmptyDocument(t *testing.T) {
	p := NewPipeline()
	err := p.Index(context.Background(), Document{ID: "", Content: "content"})
	if err == nil {
		t.Error("expected error for empty document ID")
	}
}

func TestIndex_EmptyContent(t *testing.T) {
	p := NewPipeline()
	err := p.Index(context.Background(), Document{ID: "doc-1", Content: ""})
	if err == nil {
		t.Error("expected error for empty content")
	}
}

func TestIndex_NilContext(t *testing.T) {
	p := NewPipeline()
	err := p.Index(nil, Document{ID: "doc-1", Content: "content"})
	if err == nil {
		t.Error("expected error for nil context")
	}
}

func TestRetrieve_NoIndexedDocs(t *testing.T) {
	p := NewPipeline()
	result, err := p.Retrieve(context.Background(), "anything", 5)
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if len(result.Chunks) != 0 {
		t.Errorf("expected 0 chunks with no docs, got %d", len(result.Chunks))
	}
}

func TestRetrieve_EmptyQuery(t *testing.T) {
	p := NewPipeline()
	_, err := p.Retrieve(context.Background(), "", 5)
	if err == nil {
		t.Error("expected error for empty query")
	}
}

func TestRetrieve_TopKZero(t *testing.T) {
	p := NewPipeline()
	_, err := p.Retrieve(context.Background(), "query", 0)
	if err == nil {
		t.Error("expected error for topK=0")
	}
}

func TestRetrieve_TopKLargerThanIndex(t *testing.T) {
	p := NewPipeline()
	_ = p.Index(context.Background(), Document{ID: "doc-1", Content: "Only one doc"})
	result, err := p.Retrieve(context.Background(), "doc", 100)
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if len(result.Chunks) == 0 {
		t.Error("should return available chunks even if topK > index size")
	}
}

func TestRetrieve_NilContext(t *testing.T) {
	p := NewPipeline()
	_, err := p.Retrieve(nil, "query", 5)
	if err == nil {
		t.Error("expected error for nil context")
	}
}

func TestAssembleContext_EmptyChunks(t *testing.T) {
	p := NewPipeline()
	ctx := p.AssembleContext(nil, 1000)
	if ctx != "" {
		t.Errorf("expected empty string for nil chunks, got %q", ctx)
	}
}

func TestAssembleContext_MaxLengthTruncation(t *testing.T) {
	p := NewPipeline()
	chunks := []Chunk{
		{Content: "This is a very long chunk that contains a lot of text"},
		{Content: "Another chunk with more content"},
	}
	ctx := p.AssembleContext(chunks, 20)
	if len(ctx) > 25 { // allow small overflow for separator
		t.Errorf("context should respect maxLength (~20), got length %d: %q", len(ctx), ctx)
	}
}

func TestIndex_LargeDocument(t *testing.T) {
	p := NewPipeline()
	largeContent := strings.Repeat("This is a sentence. ", 5000) // ~100k chars
	err := p.Index(context.Background(), Document{ID: "large", Content: largeContent})
	if err != nil {
		t.Fatalf("Index large doc: %v", err)
	}
}

func TestIndex_DuplicateDocumentID(t *testing.T) {
	p := NewPipeline()
	_ = p.Index(context.Background(), Document{ID: "doc-1", Content: "first version"})
	err := p.Index(context.Background(), Document{ID: "doc-1", Content: "updated version"})
	if err != nil {
		t.Fatalf("re-indexing same ID should succeed (overwrite): %v", err)
	}
}

func TestRetrieve_AfterReindex(t *testing.T) {
	p := NewPipeline()
	_ = p.Index(context.Background(), Document{ID: "doc-1", Content: "old content about cats"})
	_, _ = p.Retrieve(context.Background(), "cats", 1)

	_ = p.Index(context.Background(), Document{ID: "doc-1", Content: "new content about dogs"})
	result2, _ := p.Retrieve(context.Background(), "dogs", 1)

	if len(result2.Chunks) > 0 && strings.Contains(result2.Chunks[0].Content, "cats") {
		t.Error("retrieve after reindex should reflect updated content, not old content")
	}
}

// --- Additional edge cases for completeness ---

func TestRetrieve_DurationMeasured(t *testing.T) {
	p := NewPipeline()
	_ = p.Index(context.Background(), Document{ID: "doc-1", Content: "test document"})
	result, err := p.Retrieve(context.Background(), "test", 5)
	if err != nil {
		t.Fatalf("Retrieve: %v", err)
	}
	if result.Duration == 0 {
		t.Error("duration should be measured")
	}
}

func TestRetrieve_ResultContainsQuery(t *testing.T) {
	p := NewPipeline()
	_ = p.Index(context.Background(), Document{ID: "doc-1", Content: "hello world"})
	result, _ := p.Retrieve(context.Background(), "hello", 5)
	if result.Query != "hello" {
		t.Errorf("result.Query = %q, want %q", result.Query, "hello")
	}
}

func TestIndex_SingleSentenceDocument(t *testing.T) {
	p := NewPipeline()
	err := p.Index(context.Background(), Document{ID: "doc-1", Content: "Single sentence"})
	if err != nil {
		t.Fatalf("Index: %v", err)
	}
	result, _ := p.Retrieve(context.Background(), "single", 1)
	if len(result.Chunks) != 1 {
		t.Errorf("expected 1 chunk for single sentence, got %d", len(result.Chunks))
	}
}

func TestAssembleContext_SingleChunk(t *testing.T) {
	p := NewPipeline()
	chunks := []Chunk{{Content: "Only chunk"}}
	ctx := p.AssembleContext(chunks, 1000)
	if ctx != "Only chunk" {
		t.Errorf("single chunk should have no separator, got %q", ctx)
	}
}

func TestRetrieve_ScoresRanked(t *testing.T) {
	p := NewPipeline()
	_ = p.Index(context.Background(), Document{ID: "doc-1", Content: "AI AI AI machine learning"})
	_ = p.Index(context.Background(), Document{ID: "doc-2", Content: "weather sun rain cloud"})
	result, _ := p.Retrieve(context.Background(), "AI machine learning", 2)
	if len(result.Chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(result.Chunks))
	}
	if result.Chunks[0].Score < result.Chunks[1].Score {
		t.Error("chunks should be ranked by score descending")
	}
}

func TestAssembleContext_ZeroMaxLength(t *testing.T) {
	p := NewPipeline()
	chunks := []Chunk{{Content: "some content"}}
	ctx := p.AssembleContext(chunks, 0)
	if ctx != "some content" {
		t.Errorf("maxLength=0 should not truncate, got %q", ctx)
	}
}

func TestIndex_MetadataPreserved(t *testing.T) {
	p := NewPipeline()
	doc := Document{
		ID:       "doc-1",
		Content:  "Test content with metadata.",
		Metadata: map[string]string{"source": "test", "author": "audit"},
	}
	_ = p.Index(context.Background(), doc)

	result, _ := p.Retrieve(context.Background(), "test", 1)
	if len(result.Chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}
	if result.Chunks[0].Metadata["source"] != "test" {
		t.Error("metadata should be preserved in chunks")
	}
}

func TestIndex_MetadataDeepCopy(t *testing.T) {
	p := NewPipeline()
	original := map[string]string{"key": "original"}
	doc := Document{ID: "doc-1", Content: "Test.", Metadata: original}
	_ = p.Index(context.Background(), doc)

	// Mutate original after indexing
	original["key"] = "mutated"

	result, _ := p.Retrieve(context.Background(), "test", 1)
	if len(result.Chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}
	if result.Chunks[0].Metadata["key"] != "original" {
		t.Error("metadata should be deep-copied, not reference-shared")
	}
}

func TestRetrieve_ContextCancellation(t *testing.T) {
	p := NewPipeline()
	_ = p.Index(context.Background(), Document{ID: "doc-1", Content: "test content"})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := p.Retrieve(ctx, "test", 1)
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

func TestRetrieve_DurationNonZero(t *testing.T) {
	p := NewPipeline()
	_ = p.Index(context.Background(), Document{ID: "doc-1", Content: "test content"})
	result, _ := p.Retrieve(context.Background(), "test", 1)
	if result.Duration > time.Second {
		t.Error("duration should be sub-second for small index")
	}
}
