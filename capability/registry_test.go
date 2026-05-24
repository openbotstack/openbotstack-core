package capability

import (
	"context"
	"fmt"
	"sync"
	"testing"

	skills "github.com/openbotstack/openbotstack-core/control/skills"
)

// mockCapability is a test Capability implementation.
type mockCapability struct {
	id, name, desc string
	kind           CapabilityKind
	schema         *skills.JSONSchema
	sourceID       string
}

func (m *mockCapability) ID() string                      { return m.id }
func (m *mockCapability) Name() string                    { return m.name }
func (m *mockCapability) Description() string             { return m.desc }
func (m *mockCapability) Kind() CapabilityKind            { return m.kind }
func (m *mockCapability) InputSchema() *skills.JSONSchema { return m.schema }
func (m *mockCapability) SourceID() string                { return m.sourceID }

func TestMemoryCapabilityRegistry_RegisterAndList(t *testing.T) {
	reg := NewMemoryCapabilityRegistry()
	ctx := context.Background()

	err := reg.Register(ctx, &mockCapability{id: "skill.1", name: "test", desc: "test skill", kind: CapabilityKindSkill, sourceID: "1"})
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	descs := reg.List()
	if len(descs) != 1 {
		t.Fatalf("List() = %d, want 1", len(descs))
	}
	if descs[0].ID != "skill.1" {
		t.Errorf("ID = %q, want %q", descs[0].ID, "skill.1")
	}
}

func TestMemoryCapabilityRegistry_Unregister(t *testing.T) {
	reg := NewMemoryCapabilityRegistry()
	ctx := context.Background()

	reg.Register(ctx, &mockCapability{id: "skill.1", name: "test", kind: CapabilityKindSkill})
	reg.Unregister(ctx, "skill.1")

	descs := reg.List()
	if len(descs) != 0 {
		t.Fatalf("after unregister, List() = %d, want 0", len(descs))
	}
}

func TestMemoryCapabilityRegistry_Get(t *testing.T) {
	reg := NewMemoryCapabilityRegistry()
	ctx := context.Background()

	reg.Register(ctx, &mockCapability{id: "skill.1", name: "test", kind: CapabilityKindSkill})

	cap, err := reg.Get("skill.1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if cap.ID() != "skill.1" {
		t.Errorf("ID = %q", cap.ID())
	}

	_, err = reg.Get("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent capability")
	}
}

func TestMemoryCapabilityRegistry_ListByKind(t *testing.T) {
	reg := NewMemoryCapabilityRegistry()
	ctx := context.Background()

	reg.Register(ctx, &mockCapability{id: "skill.1", name: "s1", kind: CapabilityKindSkill})
	reg.Register(ctx, &mockCapability{id: "mcp.1.tool", name: "m1", kind: CapabilityKindMCP})
	reg.Register(ctx, &mockCapability{id: "skill.2", name: "s2", kind: CapabilityKindSkill})

	skillDescs := reg.ListByKind(CapabilityKindSkill)
	if len(skillDescs) != 2 {
		t.Fatalf("ListByKind(skill) = %d, want 2", len(skillDescs))
	}

	mcpDescs := reg.ListByKind(CapabilityKindMCP)
	if len(mcpDescs) != 1 {
		t.Fatalf("ListByKind(mcp) = %d, want 1", len(mcpDescs))
	}
}

func TestMemoryCapabilityRegistry_RegisterNil(t *testing.T) {
	reg := NewMemoryCapabilityRegistry()
	err := reg.Register(context.Background(), nil)
	if err == nil {
		t.Error("expected error for nil capability")
	}
}

func TestMemoryCapabilityRegistry_Concurrent(t *testing.T) {
	reg := NewMemoryCapabilityRegistry()
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			id := fmt.Sprintf("cap-%d", idx)
			reg.Register(ctx, &mockCapability{id: id, name: id, kind: CapabilityKindNative})
		}(i)
	}
	wg.Wait()

	descs := reg.List()
	if len(descs) != 100 {
		t.Errorf("List() = %d, want 100", len(descs))
	}
}
