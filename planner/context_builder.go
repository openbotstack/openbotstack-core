package planner

import (
	"context"
	"fmt"

	"github.com/openbotstack/openbotstack-core/assistant"
	registry "github.com/openbotstack/openbotstack-core/registry/skills"
)

// PlannerContext contains the unified state for generating an execution plan.
type PlannerContext struct {
	AssistantID   string
	Soul          assistant.AssistantSoul
	MemoryContext []assistant.SearchResult
	Skills        []SkillDescriptor
	UserRequest   string
}

// ContextBuilder orchestrates the assembly of context from various subsystems.
type ContextBuilder struct {
	skillRegistry registry.SkillRegistry
}

// NewContextBuilder creates a new ContextBuilder.
func NewContextBuilder(reg registry.SkillRegistry) *ContextBuilder {
	return &ContextBuilder{
		skillRegistry: reg,
	}
}

// Build assembles the PlannerContext for a specific request.
func (b *ContextBuilder) Build(ctx context.Context, runtime *assistant.AssistantRuntime, userRequest string) (*PlannerContext, error) {
	if runtime == nil {
		return nil, fmt.Errorf("context_builder: runtime is nil")
	}

	// 1. Fetch relevant memories (semantic search)
	memories, err := runtime.Memory.Search(ctx, userRequest, 5) // Default limit of 5
	if err != nil {
		// Log error but continue with empty memory context
		memories = nil
	}

	// 2. Fetch available skills
	// In V2, we filter by permissions. For now, we take what's in the runtime.
	availableSkills := make([]SkillDescriptor, 0)
	for _, skillID := range runtime.Skills {
		skill, err := b.skillRegistry.Get(skillID)
		if err != nil {
			continue
		}
		availableSkills = append(availableSkills, SkillDescriptor{
			ID:          skill.ID(),
			Name:        skill.Name(),
			Description: skill.Description(),
			InputSchema: skill.InputSchema(),
		})
	}

	return &PlannerContext{
		AssistantID:   runtime.AssistantID,
		Soul:          runtime.Soul,
		MemoryContext: memories,
		Skills:        availableSkills,
		UserRequest:   userRequest,
	}, nil
}
