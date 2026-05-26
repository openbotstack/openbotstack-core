package planner

import (
	"fmt"
	"sort"
	"strings"

	"github.com/openbotstack/openbotstack-core/capability"
)

// ToolSpec is a lightweight representation of a skill's input schema
// for use in LLM planner prompts. Instead of sending the full 20-field
// JSONSchema (which wastes tokens), ToolSpec extracts just the parameter
// names and their type descriptions.
type ToolSpec struct {
	ID          string
	Name        string
	Description string
	Parameters  map[string]string // param name → type description
	Required    []string          // required parameter names
}

// SchemaToToolSpec creates a lightweight ToolSpec from a SkillDescriptor.
func SchemaToToolSpec(desc SkillDescriptor) ToolSpec {
	return descriptorToToolSpec(desc)
}

// CapabilityToToolSpec creates a ToolSpec from a CapabilityDescriptor.
// Since CapabilityDescriptor is now a type alias for SkillDescriptor, this
// delegates to the same implementation.
func CapabilityToToolSpec(desc capability.CapabilityDescriptor) ToolSpec {
	return descriptorToToolSpec(desc)
}

// descriptorToToolSpec is the canonical conversion from SkillDescriptor to ToolSpec.
// Both SchemaToToolSpec and CapabilityToToolSpec delegate here, eliminating the
// previous 80% code duplication.
func descriptorToToolSpec(desc SkillDescriptor) ToolSpec {
	spec := ToolSpec{
		ID:          desc.ID,
		Name:        desc.Name,
		Description: desc.Description,
	}
	if desc.InputSchema != nil && desc.InputSchema.Properties != nil {
		spec.Parameters = make(map[string]string, len(desc.InputSchema.Properties))
		for name, schema := range desc.InputSchema.Properties {
			typeStr := schema.Type
			if typeStr == "" {
				typeStr = "value"
			}
			if schema.Description != "" && schema.Description != typeStr {
				typeStr = typeStr + " (" + schema.Description + ")"
			}
			spec.Parameters[name] = typeStr
		}
	}
	if desc.InputSchema != nil {
		spec.Required = desc.InputSchema.Required
	}
	return spec
}

// FormatToolSpecs formats a slice of ToolSpecs as a compact string for LLM prompts.
func FormatToolSpecs(specs []ToolSpec) string {
	var sb strings.Builder
	for _, spec := range specs {
		fmt.Fprintf(&sb, "- %s (%s): %s", spec.ID, spec.Name, spec.Description)
		if len(spec.Parameters) > 0 {
			requiredSet := make(map[string]bool, len(spec.Required))
			for _, r := range spec.Required {
				requiredSet[r] = true
			}
			names := make([]string, 0, len(spec.Parameters))
			for n := range spec.Parameters {
				names = append(names, n)
			}
			sort.Strings(names)
			sb.WriteString("\n  Params: ")
			for i, n := range names {
				if i > 0 {
					sb.WriteString(", ")
				}
				fmt.Fprintf(&sb, "%s: %s", n, spec.Parameters[n])
				if requiredSet[n] {
					sb.WriteString(" [required]")
				}
			}
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
