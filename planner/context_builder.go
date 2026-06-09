package planner

import (
	aitypes "github.com/openbotstack/openbotstack-core/ai/types"
)

// PlannerContext contains the unified state for generating an execution plan.
// The Skills field holds all planner-facing descriptors regardless of source
// (skills, MCP tools, builtin tools). The former separate Capabilities field
// has been unified since CapabilityDescriptor is a type alias for SkillDescriptor.
type PlannerContext struct {
	AssistantID        string
	Soul               AssistantSoul
	MemoryContext      []SearchResult
	Skills             []aitypes.SkillDescriptor
	UserRequest        string
	ProgressFn         ProgressFn
	// ConversationHistory holds prior session messages (user + assistant turns).
	// System-role messages are filtered at each injection site independently
	// (planner, LLMGenerator, ReasoningLoop) because each has independent
	// message construction. Nil/empty = no history (backward compatible).
	// Bounded by maxHistoryMessages (default 50) at load time.
	ConversationHistory []aitypes.Message
}
