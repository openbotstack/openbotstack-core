package capability

import (
	"testing"
	"time"

	skills "github.com/openbotstack/openbotstack-core/control/skills"
)

// stubSkill implements registry.Skill for testing.
type stubSkill struct {
	id, name, desc string
	schema         *skills.JSONSchema
}

func (s *stubSkill) ID() string                      { return s.id }
func (s *stubSkill) Name() string                    { return s.name }
func (s *stubSkill) Description() string             { return s.desc }
func (s *stubSkill) InputSchema() *skills.JSONSchema { return s.schema }
func (s *stubSkill) OutputSchema() *skills.JSONSchema { return nil }
func (s *stubSkill) RequiredPermissions() []string    { return nil }
func (s *stubSkill) Timeout() time.Duration           { return 30 * time.Second }
func (s *stubSkill) Validate() error                  { return nil }

func TestSkillAdapter(t *testing.T) {
	s := &stubSkill{id: "core/search", name: "Search", desc: "Search documents", schema: &skills.JSONSchema{Type: "object"}}
	adapter := &SkillAdapter{Skill: s}

	if adapter.ID() != "core/search" {
		t.Errorf("ID = %q", adapter.ID())
	}
	if adapter.Kind() != CapabilityKindSkill {
		t.Errorf("Kind = %q", adapter.Kind())
	}
	if adapter.SourceID() != "core/search" {
		t.Errorf("SourceID = %q", adapter.SourceID())
	}
}

func TestSkillToDescriptor(t *testing.T) {
	s := &stubSkill{id: "core/search", name: "Search", desc: "Search documents"}
	d := SkillToDescriptor(s)

	if d.Kind != CapabilityKindSkill {
		t.Errorf("Kind = %q", d.Kind)
	}
	if d.SourceID != "core/search" {
		t.Errorf("SourceID = %q", d.SourceID)
	}
}
