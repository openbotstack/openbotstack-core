package skills

import (
	"errors"
	"time"

	"gopkg.in/yaml.v3"
)

// SkillManifest defines a skill's metadata and requirements.
type SkillManifest struct {
	ID          string         `yaml:"id"`
	Version     string         `yaml:"version"`
	Name        string         `yaml:"name,omitempty"`
	Description string         `yaml:"description,omitempty"`
	Requires    []string       `yaml:"requires,omitempty"`
	Permissions []string       `yaml:"permissions,omitempty"`
	Timeout     time.Duration  `yaml:"timeout,omitempty"`
	Resources   ResourceLimits `yaml:"resources,omitempty"`
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
func (m *SkillManifest) Validate() error {
	if m.ID == "" {
		return errors.New("skill: id is required")
	}
	if m.Version == "" {
		return errors.New("skill: version is required")
	}
	return nil
}
