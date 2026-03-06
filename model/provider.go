package model

import "context"

// ModelProvider abstracts a model backend (Claude, OpenAI, etc.).
//
// Providers are capability-based: skills declare required capabilities,
// and the router selects an appropriate provider.
type ModelProvider interface {
	// ID returns the unique identifier for this provider.
	// Format: "vendor/model" (e.g., "anthropic/claude-3-opus", "openai/gpt-4o")
	ID() string

	// Capabilities returns the list of capabilities this provider supports.
	Capabilities() []CapabilityType

	// Generate performs a model generation call.
	Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error)

	// Embed generates embeddings for the given texts.
	// Only available if CapEmbedding is in Capabilities().
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

// ModelRouter selects a provider based on required capabilities and constraints.
type ModelRouter interface {
	// Route selects the best provider for the given requirements.
	// Returns ErrNoMatchingProvider if no provider satisfies the requirements.
	Route(requirements []CapabilityType, constraints ModelConstraints) (ModelProvider, error)

	// Register adds a provider to the router.
	Register(provider ModelProvider) error

	// List returns all registered provider IDs.
	List() []string
}
