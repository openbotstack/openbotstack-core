// Package model provides core domain models for OpenBotStack.
package assistants

// AssistantProfile defines the identity and configuration of an AI assistant.
// Full definition deferred to future implementation.
type AssistantProfile struct {
	// ID is a unique identifier for this assistant profile.
	ID string

	// Name is the display name of the assistant.
	Name string

	// Description explains what this assistant does.
	Description string

	// SystemPrompt is the base prompt defining assistant behavior.
	SystemPrompt string

	// EnabledSkillIDs lists skills this assistant can use.
	EnabledSkillIDs []string
}
