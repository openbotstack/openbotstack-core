package assistant

import "github.com/openbotstack/openbotstack-core/planner"

// defaultSystemPrompt is used when no Soul.SystemPrompt is configured.
const defaultSystemPrompt = "You are a helpful AI assistant."

// AssistantRuntime represents the active, request-scoped state of an assistant.
// It governs what the assistant can do and what data it can access.
type AssistantRuntime struct {
	AssistantID string
	TenantID    string

	// Soul defines the personality and behavioral instructions.
	Soul planner.AssistantSoul

	// Skills available to this specific assistant instance.
	Skills []string

	// Policies enforce security and governance boundaries.
	Policies []string

	// MemoryScope defines the visibility and persistence of memory.
	MemoryScope string

	// ToolPermissions define which tools the assistant can invoke.
	ToolPermissions []string
}

// EffectiveSystemPrompt returns the system prompt, falling back to a default.
func (r *AssistantRuntime) EffectiveSystemPrompt() string {
	if r.Soul.SystemPrompt != "" {
		return r.Soul.SystemPrompt
	}
	return defaultSystemPrompt
}

// EffectivePersonality returns the personality description, or empty string.
func (r *AssistantRuntime) EffectivePersonality() string {
	return r.Soul.Personality
}

// EffectiveInstructions returns the behavioral instructions, or empty string.
func (r *AssistantRuntime) EffectiveInstructions() string {
	return r.Soul.Instructions
}

// AllowedSkills returns the skills this assistant is allowed to use.
// Returns nil if no restrictions are configured.
func (r *AssistantRuntime) AllowedSkills() []string {
	return r.Soul.AllowedSkills
}

// AllowedTools returns the tools this assistant is allowed to invoke.
// Returns nil if no restrictions are configured.
func (r *AssistantRuntime) AllowedTools() []string {
	return r.Soul.AllowedTools
}
