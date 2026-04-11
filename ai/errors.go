package ai

import "errors"

var (
	// ErrNoMatchingProvider is returned when no provider matches requirements.
	ErrNoMatchingProvider = errors.New("model: no matching provider for requirements")

	// ErrProviderAlreadyExists is returned when registering a duplicate provider.
	ErrProviderAlreadyExists = errors.New("model: provider already exists")

	// ErrCapabilityNotSupported is returned when requesting an unsupported capability.
	ErrCapabilityNotSupported = errors.New("model: capability not supported")

	// ErrGenerationFailed is returned when model generation fails.
	ErrGenerationFailed = errors.New("model: generation failed")

	// ErrRateLimited is returned when the provider rate limits the request.
	ErrRateLimited = errors.New("model: rate limited")

	// ErrContextCanceled is returned when the context is canceled.
	ErrContextCanceled = errors.New("model: context canceled")

	// ErrProviderUnavailable indicates the LLM provider returned a server error (5xx).
	ErrProviderUnavailable = errors.New("model: provider unavailable")

	// ErrBadRequest indicates the LLM provider rejected the request (4xx, not 401/403/429).
	ErrBadRequest = errors.New("model: bad request")

	// ErrAuthenticationFailed indicates authentication with the LLM provider failed.
	ErrAuthenticationFailed = errors.New("model: authentication failed")
)
