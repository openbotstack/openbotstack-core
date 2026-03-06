package agent

import "errors"

var (
	// ErrInvalidTransition is returned when a state transition is not allowed.
	ErrInvalidTransition = errors.New("agent: invalid state transition")

	// ErrMaxReflectionsExceeded is returned when reflection limit is reached.
	ErrMaxReflectionsExceeded = errors.New("agent: max reflections exceeded")

	// ErrSerializationFailed is returned when state serialization fails.
	ErrSerializationFailed = errors.New("agent: serialization failed")
)
