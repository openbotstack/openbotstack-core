package assistant

import "github.com/openbotstack/openbotstack-core/planner"

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
