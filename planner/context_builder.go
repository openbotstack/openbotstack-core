package planner

import (
	"github.com/openbotstack/openbotstack-core/assistant"
	"github.com/openbotstack/openbotstack-core/capability"
)

// PlannerContext contains the unified state for generating an execution plan.
type PlannerContext struct {
	AssistantID   string
	Soul          assistant.AssistantSoul
	MemoryContext []assistant.SearchResult
	Skills        []SkillDescriptor
	Capabilities  []capability.CapabilityDescriptor
	UserRequest   string
	ProgressFn    ProgressFn
}
