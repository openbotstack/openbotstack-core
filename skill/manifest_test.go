package skill_test

import (
	"testing"
	"time"

	"github.com/openbotstack/openbotstack-core/skill"
)

func TestSkillManifestParse(t *testing.T) {
	yaml := `
id: core/search
version: 1.0.0
name: Search Skill
description: Search through documents
requires:
  - text_generation
  - tool_calling
permissions:
  - read:documents
timeout: 30s
resources:
  max_memory_mb: 128
  max_cpu_ms: 5000
`
	manifest, err := skill.ParseManifest([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseManifest failed: %v", err)
	}

	if manifest.ID != "core/search" {
		t.Errorf("Expected ID 'core/search', got '%s'", manifest.ID)
	}
	if manifest.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", manifest.Version)
	}
}

func TestSkillManifestRequires(t *testing.T) {
	yaml := `
id: test/skill
version: 1.0.0
requires:
  - text_generation
  - vision
`
	manifest, err := skill.ParseManifest([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseManifest failed: %v", err)
	}

	if len(manifest.Requires) != 2 {
		t.Errorf("Expected 2 requirements, got %d", len(manifest.Requires))
	}
}

func TestSkillManifestTimeout(t *testing.T) {
	yaml := `
id: test/timeout
version: 1.0.0
timeout: 1m30s
`
	manifest, err := skill.ParseManifest([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseManifest failed: %v", err)
	}

	expected := 90 * time.Second
	if manifest.Timeout != expected {
		t.Errorf("Expected timeout %v, got %v", expected, manifest.Timeout)
	}
}

func TestSkillManifestResources(t *testing.T) {
	yaml := `
id: test/resources
version: 1.0.0
resources:
  max_memory_mb: 256
  max_cpu_ms: 10000
`
	manifest, err := skill.ParseManifest([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseManifest failed: %v", err)
	}

	if manifest.Resources.MaxMemoryMB != 256 {
		t.Errorf("Expected max_memory_mb 256, got %d", manifest.Resources.MaxMemoryMB)
	}
	if manifest.Resources.MaxCPUMs != 10000 {
		t.Errorf("Expected max_cpu_ms 10000, got %d", manifest.Resources.MaxCPUMs)
	}
}

func TestSkillManifestValidate(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{
			name: "valid",
			yaml: `
id: valid/skill
version: 1.0.0
`,
			wantErr: false,
		},
		{
			name: "missing id",
			yaml: `
version: 1.0.0
`,
			wantErr: true,
		},
		{
			name: "missing version",
			yaml: `
id: test/skill
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest, err := skill.ParseManifest([]byte(tt.yaml))
			if err != nil {
				if !tt.wantErr {
					t.Errorf("ParseManifest failed: %v", err)
				}
				return
			}

			err = manifest.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSkillManifestPermissions(t *testing.T) {
	yaml := `
id: test/perms
version: 1.0.0
permissions:
  - read:files
  - write:files
  - exec:commands
`
	manifest, err := skill.ParseManifest([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseManifest failed: %v", err)
	}

	if len(manifest.Permissions) != 3 {
		t.Errorf("Expected 3 permissions, got %d", len(manifest.Permissions))
	}
}
