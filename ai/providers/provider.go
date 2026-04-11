package providers

import (
	"context"
	"github.com/openbotstack/openbotstack-core/control/skills"
)

// ModelProvider abstracts a model backend (Claude, OpenAI, etc.).
//
// Providers are capability-based: skills declare required capabilities,
// and the router selects an appropriate provider.
type ModelProvider interface {
	// ID returns the unique identifier for this provider.
	// Format: "vendor/model" (e.g., "anthropic/claude-3-opus", "openai/gpt-4o")
	ID() string

	// Capabilities returns the list of capabilities this provider supports.
	Capabilities() []skills.CapabilityType

	// Generate performs a model generation call.
	Generate(ctx context.Context, req skills.GenerateRequest) (*skills.GenerateResponse, error)

	// Embed generates embeddings for the given texts.
	// Only available if skills.CapEmbedding is in Capabilities().
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

// ModelRouter selects a provider based on required capabilities and constraints.
type ModelRouter interface {
	// Route selects the best provider for the given requirements.
	// Returns ai.ErrNoMatchingProvider if no provider satisfies the requirements.
	Route(requirements []skills.CapabilityType, constraints skills.ModelConstraints) (ModelProvider, error)

	// Register adds a provider to the router.
	Register(provider ModelProvider) error

	// List returns all registered provider IDs.
	List() []string
}

// StreamingModelProvider extends ModelProvider with streaming support.
// Providers that support streaming should implement this interface.
// Callers can type-assert: if sp, ok := provider.(StreamingModelProvider); ok { ... }
type StreamingModelProvider interface {
	ModelProvider
	// GenerateStream performs a streaming model generation call.
	// Returns a read-only channel that emits StreamChunk values.
	// The channel is closed when the stream ends (either naturally or on error).
	// The caller MUST drain the channel (range over it) to avoid goroutine leaks.
	// Context cancellation closes the channel with an error chunk.
	GenerateStream(ctx context.Context, req skills.GenerateRequest) (<-chan skills.StreamChunk, error)
}
