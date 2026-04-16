package assistant

import (
	"context"
	"errors"
)

var (
	ErrMemoryNotFound = errors.New("assistant: memory key not found")
)

// SearchResult represents a single entry found during a semantic search.
type SearchResult struct {
	Content []byte
	Score   float32
}

// AssistantMemory defines the contract for storing and searching assistant knowledge.
type AssistantMemory interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte) error
	Search(ctx context.Context, query string, limit int) ([]SearchResult, error)
}

// SessionMemory is an ephemeral, request-scoped memory implementation.
type SessionMemory struct {
	data map[string][]byte
}

func NewSessionMemory() *SessionMemory {
	return &SessionMemory{data: make(map[string][]byte)}
}

func (m *SessionMemory) Get(ctx context.Context, key string) ([]byte, error) {
	val, ok := m.data[key]
	if !ok {
		return nil, ErrMemoryNotFound
	}
	return val, nil
}

func (m *SessionMemory) Set(ctx context.Context, key string, value []byte) error {
	m.data[key] = value
	return nil
}

func (m *SessionMemory) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	// Basic session memory doesn't support semantic search by default.
	return nil, nil
}
