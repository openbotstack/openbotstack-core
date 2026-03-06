package memory

import "errors"

var (
	// ErrMemoryNotFound is returned when a memory ID does not exist.
	ErrMemoryNotFound = errors.New("memory: not found")

	// ErrStoreFailed is returned when storing a memory entry fails.
	ErrStoreFailed = errors.New("memory: store failed")

	// ErrRetrieveFailed is returned when retrieving memories fails.
	ErrRetrieveFailed = errors.New("memory: retrieve failed")

	// ErrSummarizeFailed is returned when memory summarization fails.
	ErrSummarizeFailed = errors.New("memory: summarize failed")
)
