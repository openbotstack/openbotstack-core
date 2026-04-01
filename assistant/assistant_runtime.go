package assistant

import (
	"github.com/openbotstack/openbotstack-core/control/skills"
)

// AssistantRuntime represents the active, request-scoped state of an assistant.
// It governs what the assistant can do and what data it can access.
type AssistantRuntime struct {
	AssistantID     string
	TenantID        string
	
	// Soul defines the personality and behavioral instructions.
	Soul            AssistantSoul
	
	// Memory provides access to ephemeral and persistent knowledge.
	Memory          AssistantMemory
	
	// Skills available to this specific assistant instance.
	Skills          []string
	
	// Policies enforce security and governance boundaries.
	Policies        []string
	
	// MemoryScope defines the visibility and persistence of memory.
	MemoryScope     string
	
	// ToolPermissions define which tools the assistant can invoke.
	ToolPermissions []string
}

// AssistantConfig holds the configuration needed to bootstrap an AssistantRuntime.
type AssistantConfig struct {
	AssistantID     string
	Soul            AssistantSoul
	Skills          []string
	Policies        []string
	MemoryScope     string
	ToolAllowedList []skills.CapabilityType
}
