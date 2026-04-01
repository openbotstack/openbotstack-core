package abstraction

import "time"

// MemoryEntry represents a single unit of memory.
type MemoryEntry struct {
	// ID is a unique identifier for this memory entry.
	ID string

	// Content is the raw text content.
	Content string

	// Embedding is the vector embedding (may be nil if not yet computed).
	Embedding []float32

	// Tags are categorical labels for structured retrieval.
	Tags []string

	// Metadata contains arbitrary key-value pairs.
	Metadata map[string]string

	// CreatedAt is when this entry was created.
	CreatedAt time.Time

	// TTL is the time-to-live; nil means no expiry.
	TTL *time.Duration
}
