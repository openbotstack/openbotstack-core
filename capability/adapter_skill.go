package capability

import (
	skills "github.com/openbotstack/openbotstack-core/control/skills"
	registry "github.com/openbotstack/openbotstack-core/registry/skills"
)

// SkillAdapter wraps a registry.Skill to satisfy the Capability interface.
type SkillAdapter struct {
	Skill registry.Skill
}

func (a *SkillAdapter) ID() string                      { return a.Skill.ID() }
func (a *SkillAdapter) Name() string                    { return a.Skill.Name() }
func (a *SkillAdapter) Description() string             { return a.Skill.Description() }
func (a *SkillAdapter) Kind() CapabilityKind            { return CapabilityKindSkill }
func (a *SkillAdapter) InputSchema() *skills.JSONSchema { return a.Skill.InputSchema() }
func (a *SkillAdapter) SourceID() string                { return a.Skill.ID() }

// SkillToDescriptor converts a Skill directly to a CapabilityDescriptor.
func SkillToDescriptor(s registry.Skill) CapabilityDescriptor {
	return CapabilityDescriptor{
		ID:          s.ID(),
		Name:        s.Name(),
		Description: s.Description(),
		InputSchema: s.InputSchema(),
		Kind:        CapabilityKindSkill,
		SourceID:    s.ID(),
	}
}
