package context

import "errors"

var (
	// ErrProfileNotFound is returned when the assistant profile doesn't exist.
	ErrProfileNotFound = errors.New("context: profile not found")

	// ErrMemoryRetrievalFailed is returned when memory retrieval fails.
	ErrMemoryRetrievalFailed = errors.New("context: memory retrieval failed")

	// ErrAssemblyFailed is returned when context assembly fails.
	ErrAssemblyFailed = errors.New("context: assembly failed")
)
