// Package context provides context assembly for OpenBotStack.
//
// The ContextAssembler builds the LLM prompt from:
//   - AssistantProfile (persona, system prompt)
//   - Memory (short-term and long-term)
//   - Current request
package context

import (
	"github.com/openbotstack/openbotstack-core/control/skills"
	"context"

	"github.com/openbotstack/openbotstack-core/memory/abstraction"
)

// AssembledContext is the complete context for an LLM call.
type AssembledContext struct {
	// SystemPrompt is the final system prompt including persona.
	SystemPrompt string

	// Messages is the conversation history with injected memory.
	Messages []skills.Message

	// AvailableTools is the list of tools the model can call.
	AvailableTools []skills.ToolDefinition

	// Constraints limits applied to this request.
	Constraints skills.ModelConstraints

	// RelevantMemories are the memories retrieved for this context.
	RelevantMemories []abstraction.MemoryEntry
}

// AssistantContext provides the assistant's static configuration.
type AssistantContext struct {
	// ProfileID is the assistant profile identifier.
	ProfileID string

	// Persona defines tone, verbosity, domain.
	Persona Persona

	// BaseSystemPrompt is the foundation system prompt.
	BaseSystemPrompt string

	// EnabledSkillIDs lists available skills.
	EnabledSkillIDs []string

	// MaxReflections bounds the reflection loop.
	MaxReflections int
}

// Persona defines the assistant's personality.
type Persona struct {
	Tone      string // "professional", "friendly", "neutral"
	Verbosity string // "low", "medium", "high"
	Domain    string // e.g., "cardiology", "general"
}

// UserRequest is the incoming user message.
type UserRequest struct {
	// Message is the user's input.
	Message string

	// ConversationID links to the ongoing conversation.
	ConversationID string

	// TenantID identifies the tenant.
	TenantID string

	// UserID identifies the user.
	UserID string
}

// ContextAssembler builds the complete context for an LLM request.
type ContextAssembler interface {
	// Assemble builds the context from profile, memory, and request.
	Assemble(
		ctx context.Context,
		assistant AssistantContext,
		request UserRequest,
		conversationHistory []skills.Message,
	) (*AssembledContext, error)
}
