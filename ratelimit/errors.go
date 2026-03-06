package ratelimit

import "errors"

var (
	// ErrRateLimitExceeded is returned when quota is exhausted.
	ErrRateLimitExceeded = errors.New("ratelimit: quota exceeded")

	// ErrQuotaNotFound is returned when no quota config exists.
	ErrQuotaNotFound = errors.New("ratelimit: quota not found")

	// ErrInvalidKey is returned when the rate limit key is malformed.
	ErrInvalidKey = errors.New("ratelimit: invalid key")
)
