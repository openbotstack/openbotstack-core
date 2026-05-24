package skills

import (
	"errors"
	"fmt"
	"time"

	"github.com/openbotstack/openbotstack-core/control/skills"
	"gopkg.in/yaml.v3"
)

// ValidRiskLevels are the allowed values for RiskLevel.
var ValidRiskLevels = map[string]bool{
	"info":      true,
	"sensitive": true,
	"clinical":  true,
	"critical":  true,
}

// ExecutionConfig defines how a skill is executed.
type ExecutionConfig struct {
	Mode string `yaml:"mode"` // declarative or wasm (native not yet implemented)
}

// SkillManifest defines a skill's metadata, execution config, and schemas.
type SkillManifest struct {
	ID          string         `yaml:"id"`
	Version     string         `yaml:"version"`
	Name        string         `yaml:"name,omitempty"`
	Description string         `yaml:"description,omitempty"`
	Execution   ExecutionConfig `yaml:"execution"`
	InputSchema *skills.JSONSchema `yaml:"input_schema,omitempty"`
	OutputSchema *skills.JSONSchema `yaml:"output_schema,omitempty"`
	Requires    []string       `yaml:"requires,omitempty"`
	Permissions []string       `yaml:"permissions,omitempty"`
	Timeout     time.Duration  `yaml:"timeout,omitempty"`
	Resources   ResourceLimits `yaml:"resources,omitempty"`
	RiskLevel   string         `yaml:"risk_level,omitempty"` // info, sensitive, clinical, critical
}

// ResourceLimits defines execution resource constraints.
type ResourceLimits struct {
	MaxMemoryMB int64 `yaml:"max_memory_mb,omitempty"`
	MaxCPUMs    int64 `yaml:"max_cpu_ms,omitempty"`
}

// ParseManifest parses a YAML manifest into a SkillManifest.
func ParseManifest(data []byte) (*SkillManifest, error) {
	var manifest SkillManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

// Validate checks if the manifest is valid.
// ID and name are no longer required — they come from SKILL.md frontmatter.
// Only execution.mode is required if a manifest is present.
func (m *SkillManifest) Validate() error {
	if m.Execution.Mode == "" {
		return errors.New("skill: execution.mode is required")
	}
	switch m.Execution.Mode {
	case "declarative", "wasm":
	default:
		return fmt.Errorf("skill: unsupported execution.mode %q (must be declarative or wasm; native is not yet implemented)", m.Execution.Mode)
	}
	if m.RiskLevel != "" && !ValidRiskLevels[m.RiskLevel] {
		return fmt.Errorf("skill: invalid risk_level %q (must be info, sensitive, clinical, or critical)", m.RiskLevel)
	}
	return nil
}
