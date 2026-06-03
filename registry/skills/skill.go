// Package skill defines the Skill interface and SkillRegistry for OpenBotStack.
//
// A Skill is a governed, declarative unit of capability. Unlike tools (low-level
// operations like HTTP calls or DB queries), Skills are governed compositions
// with explicit inputs, outputs, permissions, and constraints.
//
// Skills are stateless descriptors. They do NOT execute anything.
// Execution is delegated to openbotstack-runtime.
package skills

import (
	"time"

	aitypes "github.com/openbotstack/openbotstack-core/ai/types"
)

// Skill defines a governed, declarative unit of capability.
//
// A Skill is NOT a tool. Whereas tools are low-level operations
// (HTTP call, DB query), Skills are governed compositions of tools
// with explicit inputs, outputs, permissions, and constraints.
//
// Skills are stateless descriptors. They do NOT execute anything.
// Execution is delegated to openbotstack-runtime.
type Skill interface {
	// ID returns the unique, stable identifier for this skills.
	// Format: "namespace/name" (e.g., "core/search", "custom/invoice-generator")
	ID() string

	// Name returns a human-readable display name.
	Name() string

	// Description returns a concise explanation of what this skill does.
	// This is used by the LLM for skill selection.
	Description() string

	// InputSchema returns the JSON Schema defining expected inputs.
	// Returns nil if the skill takes no inputs.
	InputSchema() *aitypes.JSONSchema

	// OutputSchema returns the JSON Schema defining expected outputs.
	// Returns nil if the skill produces no structured output.
	OutputSchema() *aitypes.JSONSchema

	// RequiredPermissions returns the permission strings this skill requires.
	// Empty slice means no special permissions required.
	RequiredPermissions() []string

	// Timeout returns the maximum allowed execution duration.
	Timeout() time.Duration

	// Validate checks if the skill definition is internally consistent.
	Validate() error
}

// ExecutionModeProvider is an optional interface that skills can implement
// to declare how they should be executed (declarative, wasm, native).
// If not implemented, GetExecutionMode defaults to "declarative".
type ExecutionModeProvider interface {
	ExecutionMode() string
}

// GetExecutionMode returns the execution mode for a skill.
// If the skill implements ExecutionModeProvider and returns a non-empty
// string, that value is used. Otherwise defaults to "declarative".
func GetExecutionMode(s Skill) string {
	if em, ok := s.(ExecutionModeProvider); ok {
		if mode := em.ExecutionMode(); mode != "" {
			return mode
		}
	}
	return "declarative"
}

// PromptProvider is an optional interface that skills can implement
// to provide LLM instruction text for declarative execution.
// For declarative skills, this is loaded from SKILL.md.
// For wasm/native skills, this is typically empty.
type PromptProvider interface {
	Prompt() string
}

// GetPrompt returns the prompt text for a skill.
// If the skill implements PromptProvider and returns a non-empty
// string, that value is used. Otherwise returns empty string.
func GetPrompt(s Skill) string {
	if pp, ok := s.(PromptProvider); ok {
		return pp.Prompt()
	}
	return ""
}

// RiskLevelProvider is an optional interface that skills can implement
// to declare their risk classification (info, sensitive, clinical, critical).
type RiskLevelProvider interface {
	RiskLevel() string
}

// GetRiskLevel returns the risk level for a skill.
// If the skill implements RiskLevelProvider and returns a non-empty
// string, that value is used. Otherwise defaults to "info".
func GetRiskLevel(s Skill) string {
	if rl, ok := s.(RiskLevelProvider); ok {
		if level := rl.RiskLevel(); level != "" {
			return level
		}
	}
	return "info"
}

// DescriptorProvider is an optional interface that skills can implement
// to produce their planner-facing descriptor directly, without field-by-field
// copying through an adapter. This is the canonical way for Skill to become
// a Capability — the adapter layer is bypassed entirely.
type DescriptorProvider interface {
	Descriptor() aitypes.SkillDescriptor
}

// GetDescriptor returns the planner-facing descriptor for a skill.
// If the skill implements DescriptorProvider, that method is called.
// Otherwise a default descriptor is built from the Skill's core fields
// with Kind="skill" and SourceID=Skill.ID().
func GetDescriptor(s Skill) aitypes.SkillDescriptor {
	if dp, ok := s.(DescriptorProvider); ok {
		return dp.Descriptor()
	}
	return aitypes.SkillDescriptor{
		ID:          s.ID(),
		Name:        s.Name(),
		Description: s.Description(),
		InputSchema: s.InputSchema(),
		Kind:        "skill",
		SourceID:    s.ID(),
	}
}
