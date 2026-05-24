package capability

import (
	"context"
	"testing"

	skills "github.com/openbotstack/openbotstack-core/control/skills"
	"github.com/openbotstack/openbotstack-core/mcp"
)

func TestMCPToolAdapter(t *testing.T) {
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
	adapter := &MCPToolAdapter{Tool: tool, ServerID: "my-server"}

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

func TestMCPToolAdapter_InputSchemaPassthrough(t *testing.T) {
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
	adapter := &MCPToolAdapter{Tool: tool, ServerID: "srv"}

	got := adapter.InputSchema()
	if got != schema {
		t.Errorf("InputSchema should be identity passthrough, got different pointer")
	}
}

func TestMCPToolAdapter_NilInputSchema(t *testing.T) {
	tool := mcp.ClientTool{
		Name:        "ping",
		Description: "Ping",
	}
	adapter := &MCPToolAdapter{Tool: tool, ServerID: "srv"}

	got := adapter.InputSchema()
	if got != nil {
		t.Errorf("expected nil InputSchema, got %+v", got)
	}
}

func TestMCPToolAdapter_InRegistry(t *testing.T) {
	reg := NewMemoryCapabilityRegistry()
	ctx := context.Background()

	tool := mcp.ClientTool{
		Name:        "search",
		Description: "Search",
		InputSchema: &skills.JSONSchema{Type: "object"},
	}
	adapter := &MCPToolAdapter{Tool: tool, ServerID: "srv1"}

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
