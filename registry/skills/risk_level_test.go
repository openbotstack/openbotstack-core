package skills_test

import (
	"testing"
	"time"

	"github.com/openbotstack/openbotstack-core/ai/types"
	"github.com/openbotstack/openbotstack-core/registry/skills"
)

type mockRiskSkill struct {
	id        string
	riskLevel string
}

func (m *mockRiskSkill) ID() string                      { return m.id }
func (m *mockRiskSkill) Name() string                    { return m.id }
func (m *mockRiskSkill) Description() string             { return "" }
func (m *mockRiskSkill) InputSchema() *types.JSONSchema     { return nil }
func (m *mockRiskSkill) OutputSchema() *types.JSONSchema    { return nil }
func (m *mockRiskSkill) RequiredPermissions() []string   { return nil }
func (m *mockRiskSkill) Timeout() time.Duration          { return 0 }
func (m *mockRiskSkill) Validate() error                 { return nil }
func (m *mockRiskSkill) RiskLevel() string               { return m.riskLevel }

type noRiskSkill struct{}

func (n *noRiskSkill) ID() string                      { return "no-risk" }
func (n *noRiskSkill) Name() string                    { return "no-risk" }
func (n *noRiskSkill) Description() string             { return "" }
func (n *noRiskSkill) InputSchema() *types.JSONSchema     { return nil }
func (n *noRiskSkill) OutputSchema() *types.JSONSchema    { return nil }
func (n *noRiskSkill) RequiredPermissions() []string   { return nil }
func (n *noRiskSkill) Timeout() time.Duration          { return 0 }
func (n *noRiskSkill) Validate() error                 { return nil }

func TestGetRiskLevel_WithProvider(t *testing.T) {
	s := &mockRiskSkill{id: "test/clinical", riskLevel: "clinical"}
	got := skills.GetRiskLevel(s)
	if got != "clinical" {
		t.Errorf("GetRiskLevel = %q, want %q", got, "clinical")
	}
}

func TestGetRiskLevel_WithoutProvider(t *testing.T) {
	s := &noRiskSkill{}
	got := skills.GetRiskLevel(s)
	if got != "info" {
		t.Errorf("GetRiskLevel = %q, want default %q", got, "info")
	}
}

func TestGetRiskLevel_EmptyString(t *testing.T) {
	s := &mockRiskSkill{id: "test/empty", riskLevel: ""}
	got := skills.GetRiskLevel(s)
	if got != "info" {
		t.Errorf("GetRiskLevel = %q, want default %q", got, "info")
	}
}
