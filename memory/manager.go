// Package memory provides abstract access to the agent's memory.
//
// "Memory" in OpenBotStack is:
//   - Short-term: Current conversation context
//   - Long-term: Vector-stored knowledge (Milvus abstraction)
//   - Entity: Structured facts about known entities
//
// This package defines interfaces ONLY. The actual storage
// implementation lives in openbotstack-runtime or external infrastructure.
package memory

import (
	"context"
)

// MemoryManager provides abstract access to the agent's memory.
//
// This interface is ONLY an abstraction. The actual storage
// implementation lives in openbotstack-runtime or external infra.
type MemoryManager interface {
	// StoreShortTerm saves conversation-scoped entries.
	// Entry expires after session ends.
	StoreShortTerm(ctx context.Context, entry MemoryEntry) error

	// StoreLongTerm saves entries to vector storage.
	// Entry is embedded and persisted for retrieval.
	StoreLongTerm(ctx context.Context, entry MemoryEntry) error

	// RetrieveSimilar performs semantic search for relevant memories.
	// Returns entries ordered by relevance score (descending).
	// limit <= 0 means use system default.
	RetrieveSimilar(ctx context.Context, query string, limit int) ([]MemoryEntry, error)

	// RetrieveByTag returns memories matching all specified tags.
	RetrieveByTag(ctx context.Context, tags []string, limit int) ([]MemoryEntry, error)

	// Forget removes a specific memory entry.
	// Returns ErrMemoryNotFound if ID doesn't exist.
	Forget(ctx context.Context, id string) error

	// Summarize triggers compaction of memories.
	// Used when context window pressure requires aggregation.
	Summarize(ctx context.Context, entries []MemoryEntry) (MemoryEntry, error)
}
