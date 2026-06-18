package planning

import (
	aitypes "github.com/openbotstack/openbotstack-core/ai/types"
	"github.com/openbotstack/openbotstack-core/control/profile"
)

// PlannerContext contains the unified state for generating an execution plan.
// The Skills field holds all planner-facing descriptors regardless of source
// (skills, MCP tools, builtin tools). The former separate Capabilities field
// has been unified since CapabilityDescriptor is a type alias for SkillDescriptor.
type PlannerContext struct {
	AssistantID string
	Soul        AssistantSoul
	// ProfileSoul, when non-nil, replaces the legacy Soul.SystemPrompt injection
	// with a minimal prompt synthesised from the structured Soul fields (ADR-042
	// Phase 2). Keeping it optional (nil) allows the planner to fall back to the
	// existing SystemPrompt string, preserving backward compatibility. The runtime
	// populates this field from the profile.ProfileStore (Global scope only in
	// Phase 2; Tenant/Session merge deferred to Phase 3).
	ProfileSoul   *profile.Soul
	MemoryContext []SearchResult
	Skills        []aitypes.SkillDescriptor
	UserRequest   string
	ProgressFn    ProgressFn
	// ConversationHistory holds prior session messages (user + assistant turns).
	// System-role messages are filtered at each injection site independently
	// (planner, LLMGenerator, ReasoningLoop) because each has independent
	// message construction. Nil/empty = no history (backward compatible).
	// Bounded by maxHistoryMessages (default 50) at load time.
	ConversationHistory []aitypes.Message
	// TurnResults carries structured tool execution results from previous
	// reasoning turns. Used by TurnPlanner to replace legacy XML <observation>
	// injection. Nil/empty = first turn or no previous results.
	TurnResults []TurnToolResult
}

// WithRequest returns a copy of the context with UserRequest replaced by msg.
// All other fields are preserved unchanged.
//
// The copy is shallow: slice fields (MemoryContext, Skills,
// ConversationHistory, TurnResults) share their backing arrays with the
// original. This is intentional and safe for the planner's read-only use —
// callers must not mutate the shared slices in place. Deep-copy the relevant
// slice if mutation is required.
//
// The original context is never mutated.
func (c PlannerContext) WithRequest(msg string) *PlannerContext {
	cp := c // struct copy (shallow: slice headers copied by value)
	cp.UserRequest = msg
	return &cp
}
