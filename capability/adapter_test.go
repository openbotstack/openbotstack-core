package capability

import (
	"context"
	"testing"
	"time"

	skills "github.com/openbotstack/openbotstack-core/control/skills"
	"github.com/openbotstack/openbotstack-core/mcp"
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

// --- Skill adapter tests ---

func TestNewFromSkill(t *testing.T) {
	s := &stubSkill{id: "core/search", name: "Search", desc: "Search documents", schema: &skills.JSONSchema{Type: "object"}}
	adapter := NewFromSkill(s)

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

// --- MCP adapter tests ---

func TestNewFromMCP(t *testing.T) {
	tool := mcp.ClientTool{
		Name:        "search",
		Description: "Search documents",
		InputSchema: &skills.JSONSchema{
			Type: "object",
			Properties: map[string]*skills.JSONSchema{
				"query": {Type: "string"},
			},
			Required: []string{"query"},
		},
	}
	adapter := NewFromMCP("my-server", tool)

	if adapter.ID() != "mcp.my-server.search" {
		t.Errorf("ID = %q, want %q", adapter.ID(), "mcp.my-server.search")
	}
	if adapter.Kind() != CapabilityKindMCP {
		t.Errorf("Kind = %q", adapter.Kind())
	}
	if adapter.SourceID() != "my-server" {
		t.Errorf("SourceID = %q", adapter.SourceID())
	}
}

func TestNewFromMCP_InputSchemaPassthrough(t *testing.T) {
	schema := &skills.JSONSchema{
		Type: "object",
		Properties: map[string]*skills.JSONSchema{
			"expression": {Type: "string"},
		},
	}
	tool := mcp.ClientTool{
		Name:        "calc",
		Description: "Calculate",
		InputSchema: schema,
	}
	adapter := NewFromMCP("srv", tool)

	got := adapter.InputSchema()
	if got != schema {
		t.Errorf("InputSchema should be identity passthrough, got different pointer")
	}
}

func TestNewFromMCP_NilInputSchema(t *testing.T) {
	tool := mcp.ClientTool{
		Name:        "ping",
		Description: "Ping",
	}
	adapter := NewFromMCP("srv", tool)

	got := adapter.InputSchema()
	if got != nil {
		t.Errorf("expected nil InputSchema, got %+v", got)
	}
}

func TestNewFromMCP_InRegistry(t *testing.T) {
	reg := NewMemoryCapabilityRegistry()
	ctx := context.Background()

	tool := mcp.ClientTool{
		Name:        "search",
		Description: "Search",
		InputSchema: &skills.JSONSchema{Type: "object"},
	}
	adapter := NewFromMCP("srv1", tool)

	if err := reg.Register(ctx, adapter); err != nil {
		t.Fatalf("Register: %v", err)
	}

	descs := reg.ListByKind(CapabilityKindMCP)
	if len(descs) != 1 {
		t.Fatalf("ListByKind(mcp) = %d, want 1", len(descs))
	}
	if descs[0].ID != "mcp.srv1.search" {
		t.Errorf("ID = %q", descs[0].ID)
	}
}

// --- Native adapter tests ---

func TestNewFromNative(t *testing.T) {
	schema := &skills.JSONSchema{Type: "object"}
	adapter := NewFromNative("builtin.now", "now", "Current UTC timestamp", schema)

	if adapter.ID() != "builtin.now" {
		t.Errorf("ID = %q, want %q", adapter.ID(), "builtin.now")
	}
	if adapter.Name() != "now" {
		t.Errorf("Name = %q, want %q", adapter.Name(), "now")
	}
	if adapter.Description() != "Current UTC timestamp" {
		t.Errorf("Description = %q", adapter.Description())
	}
	if adapter.Kind() != CapabilityKindNative {
		t.Errorf("Kind = %q, want %q", adapter.Kind(), CapabilityKindNative)
	}
	if adapter.SourceID() != "builtin" {
		t.Errorf("SourceID = %q, want %q", adapter.SourceID(), "builtin")
	}
	if adapter.InputSchema() != schema {
		t.Errorf("InputSchema should be identity passthrough")
	}
}

func TestNewFromNative_NilSchema(t *testing.T) {
	adapter := NewFromNative("builtin.ping", "ping", "Ping", nil)

	if adapter.InputSchema() != nil {
		t.Errorf("expected nil InputSchema, got %+v", adapter.InputSchema())
	}
}
