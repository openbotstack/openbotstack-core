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

	"github.com/openbotstack/openbotstack-core/control/skills"
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
	InputSchema() *skills.JSONSchema

	// OutputSchema returns the JSON Schema defining expected outputs.
	// Returns nil if the skill produces no structured output.
	OutputSchema() *skills.JSONSchema

	// RequiredPermissions returns the permission strings this skill requires.
	// Empty slice means no special permissions required.
	RequiredPermissions() []string

	// Timeout returns the maximum allowed execution duration.
	Timeout() time.Duration

	// Validate checks if the skill definition is internally consistent.
	Validate() error
}
