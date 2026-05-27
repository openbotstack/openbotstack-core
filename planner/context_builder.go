package planner

import (
	"github.com/openbotstack/openbotstack-core/assistant"
	"github.com/openbotstack/openbotstack-core/capability"
	skills "github.com/openbotstack/openbotstack-core/control/skills"
)

// PlannerContext contains the unified state for generating an execution plan.
type PlannerContext struct {
	AssistantID   string
	Soul          assistant.AssistantSoul
	MemoryContext []assistant.SearchResult
	Skills        []skills.SkillDescriptor
	Capabilities  []capability.CapabilityDescriptor
	UserRequest   string
	ProgressFn    ProgressFn
}
