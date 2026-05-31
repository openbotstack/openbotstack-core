package skills_test

import (
	"testing"

	"github.com/openbotstack/openbotstack-core/ai/types"
	registryskills "github.com/openbotstack/openbotstack-core/registry/skills"
)

// --- ExecutionModeProvider optional interface ---

// testSkillWithMode implements both Skill and ExecutionModeProvider.
type testSkillWithMode struct {
	testSkill
	mode string
}

func (s *testSkillWithMode) ExecutionMode() string {
	return s.mode
}

func TestGetExecutionMode_WithProvider(t *testing.T) {
	s := &testSkillWithMode{
		testSkill: testSkill{id: "core/search", name: "Search"},
		mode:      "declarative",
	}
	got := registryskills.GetExecutionMode(s)
	if got != "declarative" {
		t.Errorf("expected 'declarative', got %q", got)
	}
}

func TestGetExecutionMode_WithProviderEmpty(t *testing.T) {
	s := &testSkillWithMode{
		testSkill: testSkill{id: "core/search", name: "Search"},
		mode:      "",
	}
	got := registryskills.GetExecutionMode(s)
	if got != "declarative" {
		t.Errorf("empty mode should default to 'declarative', got %q", got)
	}
}

func TestGetExecutionMode_WithoutProvider(t *testing.T) {
	s := &testSkill{id: "core/search", name: "Search"}
	got := registryskills.GetExecutionMode(s)
	if got != "declarative" {
		t.Errorf("skill without ExecutionModeProvider should default to 'declarative', got %q", got)
	}
}

func TestGetExecutionMode_ValidModes(t *testing.T) {
	tests := []struct {
		mode string
		want string
	}{
		{"declarative", "declarative"},
		{"wasm", "wasm"},
		{"native", "native"},
	}
	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			s := &testSkillWithMode{
				testSkill: testSkill{id: "test"},
				mode:      tt.mode,
			}
			got := registryskills.GetExecutionMode(s)
			if got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestExecutionModeProvider_InterfaceSatisfaction(t *testing.T) {
	// Compile-time check: testSkillWithMode must implement ExecutionModeProvider.
	var _ registryskills.ExecutionModeProvider = (*testSkillWithMode)(nil)
	// Also verify it still satisfies Skill.
	var _ registryskills.Skill = (*testSkillWithMode)(nil)
}

// --- ExecutionConfig in manifest ---

func TestManifest_ExecutionConfig(t *testing.T) {
	yaml := `
id: core/summarize
version: 1.0.0
name: Summarize
description: Summarizes text
execution:
  mode: declarative
`
	manifest, err := registryskills.ParseManifest([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseManifest failed: %v", err)
	}
	if manifest.Execution.Mode != "declarative" {
		t.Errorf("Execution.Mode = %q, want 'declarative'", manifest.Execution.Mode)
	}
}

func TestManifest_ExecutionConfig_WasmMode(t *testing.T) {
	yaml := `
id: core/tax-calc
version: 1.0.0
name: Tax Calculator
execution:
  mode: wasm
`
	manifest, err := registryskills.ParseManifest([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseManifest failed: %v", err)
	}
	if manifest.Execution.Mode != "wasm" {
		t.Errorf("Execution.Mode = %q, want 'wasm'", manifest.Execution.Mode)
	}
}

func TestManifest_Validate_RequiresExecutionMode(t *testing.T) {
	manifest := &registryskills.SkillManifest{
		ID:      "core/test",
		Version: "1.0.0",
	}
	err := manifest.Validate()
	if err == nil {
		t.Fatal("Validate should require execution.mode")
	}
}

func TestManifest_Validate_ExecutionModeProvided(t *testing.T) {
	manifest := &registryskills.SkillManifest{
		ID:      "core/test",
		Version: "1.0.0",
		Execution: registryskills.ExecutionConfig{
			Mode: "wasm",
		},
	}
	if err := manifest.Validate(); err != nil {
		t.Errorf("Validate should pass with execution.mode: %v", err)
	}
}

// --- Manifest schema fields ---

func TestManifest_InputSchemaField(t *testing.T) {
	yaml := `
id: core/search
version: 1.0.0
execution:
  mode: wasm
input_schema:
  type: object
  properties:
    query:
      type: string
  required:
    - query
`
	manifest, err := registryskills.ParseManifest([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseManifest failed: %v", err)
	}
	if manifest.InputSchema == nil {
		t.Fatal("InputSchema should not be nil")
	}
	if manifest.InputSchema.Type != "object" {
		t.Errorf("InputSchema.Type = %q, want 'object'", manifest.InputSchema.Type)
	}
	if manifest.InputSchema.Properties == nil {
		t.Fatal("InputSchema.Properties should not be nil")
	}
	if prop, ok := manifest.InputSchema.Properties["query"]; !ok || prop.Type != "string" {
		t.Errorf("InputSchema.Properties['query'] missing or wrong type")
	}
}

func TestManifest_OutputSchemaField(t *testing.T) {
	yaml := `
id: core/search
version: 1.0.0
execution:
  mode: wasm
output_schema:
  type: object
  properties:
    results:
      type: array
`
	manifest, err := registryskills.ParseManifest([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseManifest failed: %v", err)
	}
	if manifest.OutputSchema == nil {
		t.Fatal("OutputSchema should not be nil")
	}
	if manifest.OutputSchema.Type != "object" {
		t.Errorf("OutputSchema.Type = %q, want 'object'", manifest.OutputSchema.Type)
	}
}

func TestManifest_MinimalValidManifest(t *testing.T) {
	yaml := `
id: core/hello
version: 1.0.0
execution:
  mode: wasm
`
	manifest, err := registryskills.ParseManifest([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseManifest failed: %v", err)
	}
	if err := manifest.Validate(); err != nil {
		t.Errorf("minimal valid manifest should pass Validate: %v", err)
	}
}

// Verify that the parsed manifest's InputSchema is compatible with types.JSONSchema
func TestManifest_InputSchemaIsJSONSchema(t *testing.T) {
	yaml := `
id: core/test
version: 1.0.0
execution:
  mode: wasm
input_schema:
  type: object
  properties:
    name:
      type: string
  required:
    - name
`
	manifest, err := registryskills.ParseManifest([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseManifest failed: %v", err)
	}
	// Verify it's a *types.JSONSchema
	if manifest.InputSchema == nil {
		t.Fatal("InputSchema should be *types.JSONSchema")
	}
	_ = types.JSONSchema(*manifest.InputSchema)
}
