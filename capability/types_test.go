package capability

import (
	"testing"

	skills "github.com/openbotstack/openbotstack-core/control/skills"
)

func TestCapabilityKind_Values(t *testing.T) {
	tests := []struct {
		kind CapabilityKind
		want string
	}{
		{CapabilityKindSkill, "skill"},
		{CapabilityKindMCP, "mcp"},
		{CapabilityKindNative, "native"},
		{CapabilityKindExternal, "external"},
	}
	for _, tt := range tests {
		if string(tt.kind) != tt.want {
			t.Errorf("CapabilityKind = %q, want %q", tt.kind, tt.want)
		}
	}
}

func TestCapabilityDescriptor_Fields(t *testing.T) {
	d := CapabilityDescriptor{
		ID:          "mcp.server1.tool1",
		Name:        "tool1",
		Description: "A test tool",
		InputSchema: &skills.JSONSchema{Type: "object"},
		Kind:        string(CapabilityKindMCP),
		SourceID:    "server1",
	}
	if d.ID != "mcp.server1.tool1" {
		t.Errorf("ID = %q", d.ID)
	}
	if d.Kind != string(CapabilityKindMCP) {
		t.Errorf("Kind = %q", d.Kind)
	}
	if d.SourceID != "server1" {
		t.Errorf("SourceID = %q", d.SourceID)
	}
}
