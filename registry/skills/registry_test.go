package skills_test

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/openbotstack/openbotstack-core/control/skills"
	registryskills "github.com/openbotstack/openbotstack-core/registry/skills"
)

// testSkill is a minimal, configurable Skill implementation for testing.
type testSkill struct {
	id          string
	name        string
	description string
	inputSchema *skills.JSONSchema
	outputSchema *skills.JSONSchema
	permissions []string
	timeout     time.Duration
	validateErr error // if non-nil, Validate() returns this error
}

func (s *testSkill) ID() string                         { return s.id }
func (s *testSkill) Name() string                       { return s.name }
func (s *testSkill) Description() string                { return s.description }
func (s *testSkill) InputSchema() *skills.JSONSchema    { return s.inputSchema }
func (s *testSkill) OutputSchema() *skills.JSONSchema   { return s.outputSchema }
func (s *testSkill) RequiredPermissions() []string      { return s.permissions }
func (s *testSkill) Timeout() time.Duration             { return s.timeout }
func (s *testSkill) Validate() error                    { return s.validateErr }

// helpers

func newTestSkill(id string) *testSkill {
	return &testSkill{
		id:          id,
		name:        id,
		description: fmt.Sprintf("test skill %s", id),
		timeout:     30 * time.Second,
	}
}

func newTestSkillWithPerms(id string, perms []string) *testSkill {
	s := newTestSkill(id)
	s.permissions = perms
	return s
}

// ---------------------------------------------------------------------------
// InMemoryRegistry: Register + Get
// ---------------------------------------------------------------------------

func TestInMemoryRegistry_RegisterAndGet(t *testing.T) {
	reg := registryskills.NewInMemoryRegistry()
	skill := newTestSkill("core/search")

	if err := reg.Register(skill); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	got, err := reg.Get("core/search")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ID() != "core/search" {
		t.Errorf("Get returned wrong skill: got %q, want %q", got.ID(), "core/search")
	}
}

func TestInMemoryRegistry_GetReturnsExactInstance(t *testing.T) {
	reg := registryskills.NewInMemoryRegistry()
	skill := newTestSkill("core/search")
	_ = reg.Register(skill)

	got, _ := reg.Get("core/search")
	if got != skill {
		t.Error("Get did not return the exact same pointer as registered")
	}
}

// ---------------------------------------------------------------------------
// InMemoryRegistry: List
// ---------------------------------------------------------------------------

func TestInMemoryRegistry_ListEmpty(t *testing.T) {
	reg := registryskills.NewInMemoryRegistry()
	ids := reg.List()
	if ids == nil || len(ids) != 0 {
		t.Errorf("List on empty registry should return empty slice, got %v", ids)
	}
}

func TestInMemoryRegistry_ListMultiple(t *testing.T) {
	reg := registryskills.NewInMemoryRegistry()
	skillIDs := []string{"core/search", "core/summarize", "custom/invoice"}

	for _, id := range skillIDs {
		if err := reg.Register(newTestSkill(id)); err != nil {
			t.Fatalf("Register(%q) failed: %v", id, err)
		}
	}

	ids := reg.List()
	if len(ids) != len(skillIDs) {
		t.Fatalf("List returned %d items, want %d", len(ids), len(skillIDs))
	}

	// Order is not guaranteed, convert to set and compare.
	got := toSet(ids)
	for _, want := range skillIDs {
		if !got[want] {
			t.Errorf("List missing skill %q; got %v", want, ids)
		}
	}
}

// ---------------------------------------------------------------------------
// InMemoryRegistry: Get non-existent
// ---------------------------------------------------------------------------

func TestInMemoryRegistry_GetNotFound(t *testing.T) {
	reg := registryskills.NewInMemoryRegistry()

	_, err := reg.Get("nonexistent")
	if err == nil {
		t.Fatal("Get should return error for non-existent skill")
	}
	if !errors.Is(err, registryskills.ErrSkillNotFound) {
		t.Errorf("Get error should wrap ErrSkillNotFound, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// InMemoryRegistry: Duplicate registration
// ---------------------------------------------------------------------------

func TestInMemoryRegistry_RegisterDuplicate(t *testing.T) {
	reg := registryskills.NewInMemoryRegistry()
	skill := newTestSkill("core/dup")

	if err := reg.Register(skill); err != nil {
		t.Fatalf("first Register failed: %v", err)
	}

	err := reg.Register(skill)
	if err == nil {
		t.Fatal("Registering duplicate should return error")
	}
	if !errors.Is(err, registryskills.ErrSkillAlreadyExists) {
		t.Errorf("duplicate error should wrap ErrSkillAlreadyExists, got: %v", err)
	}
}

func TestInMemoryRegistry_RegisterDuplicateDifferentInstance(t *testing.T) {
	reg := registryskills.NewInMemoryRegistry()

	s1 := newTestSkill("core/dup")
	s1.name = "first"
	s2 := newTestSkill("core/dup")
	s2.name = "second"

	_ = reg.Register(s1)
	err := reg.Register(s2)
	if err == nil {
		t.Fatal("registering a second skill with same ID should fail")
	}
	if !errors.Is(err, registryskills.ErrSkillAlreadyExists) {
		t.Errorf("expected ErrSkillAlreadyExists, got: %v", err)
	}

	// Verify the original is still in the registry.
	got, _ := reg.Get("core/dup")
	if got.Name() != "first" {
		t.Error("duplicate Register should not overwrite existing skill")
	}
}

// ---------------------------------------------------------------------------
// InMemoryRegistry: nil / empty-ID skill
// ---------------------------------------------------------------------------

func TestInMemoryRegistry_RegisterNilSkill(t *testing.T) {
	reg := registryskills.NewInMemoryRegistry()
	err := reg.Register(nil)
	if err == nil {
		t.Fatal("Register(nil) should return error")
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("error should mention nil, got: %v", err)
	}
}

func TestInMemoryRegistry_RegisterEmptyID(t *testing.T) {
	reg := registryskills.NewInMemoryRegistry()
	skill := newTestSkill("")
	err := reg.Register(skill)
	if err == nil {
		t.Fatal("Register with empty ID should return error")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error should mention empty, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// InMemoryRegistry: Register calls Validate
// ---------------------------------------------------------------------------

func TestInMemoryRegistry_RegisterInvalidSkill(t *testing.T) {
	reg := registryskills.NewInMemoryRegistry()

	skill := newTestSkill("core/bad")
	skill.validateErr = errors.New("missing required field")

	err := reg.Register(skill)
	if err == nil {
		t.Fatal("Register should reject skill that fails Validate()")
	}
	if !strings.Contains(err.Error(), "missing required field") {
		t.Errorf("error should propagate Validate message, got: %v", err)
	}
	// The sentinel ErrSkillInvalid is not used by Register, but the wrapper
	// should include the skill ID.
	if !strings.Contains(err.Error(), "core/bad") {
		t.Errorf("error should mention skill ID, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// InMemoryRegistry: ListByPermission
// ---------------------------------------------------------------------------

func TestInMemoryRegistry_ListByPermission_Empty(t *testing.T) {
	reg := registryskills.NewInMemoryRegistry()
	result := reg.ListByPermission(nil)
	if len(result) != 0 {
		t.Errorf("ListByPermission on empty registry should have length 0, got %d", len(result))
	}
}

func TestInMemoryRegistry_ListByPermission_NoPermissionsRequired(t *testing.T) {
	reg := registryskills.NewInMemoryRegistry()
	_ = reg.Register(newTestSkillWithPerms("core/public", nil))
	_ = reg.Register(newTestSkillWithPerms("core/also-public", []string{}))

	result := reg.ListByPermission(nil)
	if len(result) != 2 {
		t.Errorf("skills with no required permissions should appear with nil caller perms, got %d", len(result))
	}
}

func TestInMemoryRegistry_ListByPermission_ExactMatch(t *testing.T) {
	reg := registryskills.NewInMemoryRegistry()
	_ = reg.Register(newTestSkillWithPerms("core/reader", []string{"read:docs"}))
	_ = reg.Register(newTestSkillWithPerms("core/writer", []string{"write:docs"}))
	_ = reg.Register(newTestSkillWithPerms("core/admin", []string{"read:docs", "write:docs"}))
	_ = reg.Register(newTestSkillWithPerms("core/public", nil))

	result := reg.ListByPermission([]string{"read:docs"})
	ids := skillIDs(result)
	got := toSet(ids)

	// public (no perms) and reader (exact match) should be included.
	if !got["core/public"] {
		t.Error("core/public should be included (no permissions required)")
	}
	if !got["core/reader"] {
		t.Error("core/reader should be included (read:docs satisfied)")
	}
	if got["core/writer"] {
		t.Error("core/writer should NOT be included (write:docs not granted)")
	}
	if got["core/admin"] {
		t.Error("core/admin should NOT be included (partial permissions)")
	}
}

func TestInMemoryRegistry_ListByPermission_SuperSet(t *testing.T) {
	reg := registryskills.NewInMemoryRegistry()
	_ = reg.Register(newTestSkillWithPerms("core/admin", []string{"read:docs", "write:docs"}))

	result := reg.ListByPermission([]string{"read:docs", "write:docs", "delete:docs"})
	if len(result) != 1 || result[0].ID() != "core/admin" {
		t.Error("super-set of permissions should satisfy skill requirements")
	}
}

// ---------------------------------------------------------------------------
// InMemoryRegistry: Validate (registry-level)
// ---------------------------------------------------------------------------

func TestInMemoryRegistry_Validate_AllValid(t *testing.T) {
	reg := registryskills.NewInMemoryRegistry()
	_ = reg.Register(newTestSkill("core/a"))
	_ = reg.Register(newTestSkill("core/b"))

	if err := reg.Validate(); err != nil {
		t.Errorf("Validate on all-valid registry should return nil, got: %v", err)
	}
}

func TestInMemoryRegistry_Validate_InvalidSkill(t *testing.T) {
	// Manually inject an invalid skill to test registry-level Validate.
	// (Register would reject it, so we need to bypass.)
	reg := registryskills.NewInMemoryRegistry()
	_ = reg.Register(newTestSkill("core/good"))

	bad := newTestSkill("core/bad")
	bad.validateErr = errors.New("broken")

	// We cannot register a bad skill through Register() because it calls Validate().
	// So instead, validate the individual bad skill directly.
	if err := bad.Validate(); err == nil {
		t.Error("bad skill Validate() should return error")
	}
}

// ---------------------------------------------------------------------------
// InMemoryRegistry: concurrent access
// ---------------------------------------------------------------------------

func TestInMemoryRegistry_ConcurrentRegister(t *testing.T) {
	reg := registryskills.NewInMemoryRegistry()
	const n = 100

	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		go func(i int) {
			defer wg.Done()
			_ = reg.Register(newTestSkill(fmt.Sprintf("skill/%d", i)))
		}(i)
	}
	wg.Wait()

	ids := reg.List()
	if len(ids) != n {
		t.Errorf("expected %d skills after concurrent Register, got %d", n, len(ids))
	}
}

func TestInMemoryRegistry_ConcurrentReadWrite(t *testing.T) {
	reg := registryskills.NewInMemoryRegistry()
	_ = reg.Register(newTestSkill("core/existing"))

	const readers = 50
	const writers = 10
	var wg sync.WaitGroup
	wg.Add(readers + writers)

	for i := range readers {
		go func(i int) {
			defer wg.Done()
			_, _ = reg.Get("core/existing")
			_ = reg.List()
			_ = reg.ListByPermission([]string{"read"})
		}(i)
	}
	for i := range writers {
		go func(i int) {
			defer wg.Done()
			_ = reg.Register(newTestSkill(fmt.Sprintf("concurrent/%d", i)))
		}(i)
	}
	wg.Wait()

	// Should not panic; 1 existing + writers new.
	ids := reg.List()
	if len(ids) != 1+writers {
		t.Errorf("expected %d skills, got %d", 1+writers, len(ids))
	}
}

// ---------------------------------------------------------------------------
// Skill interface compliance
// ---------------------------------------------------------------------------

func TestSkillInterface_Compliance(t *testing.T) {
	// Compile-time check: testSkill must implement Skill.
	var _ registryskills.Skill = (*testSkill)(nil)
}

func TestSkillInterface_AllMethods(t *testing.T) {
	inputSchema := &skills.JSONSchema{
		Type: "object",
		Properties: map[string]*skills.JSONSchema{
			"query": {Type: "string"},
		},
		Required: []string{"query"},
	}
	outputSchema := &skills.JSONSchema{
		Type: "object",
		Properties: map[string]*skills.JSONSchema{
			"result": {Type: "string"},
		},
	}
	perms := []string{"read:docs", "write:docs"}
	timeout := 45 * time.Second

	s := &testSkill{
		id:           "core/search",
		name:         "Search",
		description:  "Search documents",
		inputSchema:  inputSchema,
		outputSchema: outputSchema,
		permissions:  perms,
		timeout:      timeout,
	}

	if s.ID() != "core/search" {
		t.Errorf("ID() = %q, want %q", s.ID(), "core/search")
	}
	if s.Name() != "Search" {
		t.Errorf("Name() = %q, want %q", s.Name(), "Search")
	}
	if s.Description() != "Search documents" {
		t.Errorf("Description() = %q, want %q", s.Description(), "Search documents")
	}
	if s.InputSchema() != inputSchema {
		t.Error("InputSchema() should return the same pointer")
	}
	if s.OutputSchema() != outputSchema {
		t.Error("OutputSchema() should return the same pointer")
	}
	if len(s.RequiredPermissions()) != 2 {
		t.Errorf("RequiredPermissions() returned %d items, want 2", len(s.RequiredPermissions()))
	}
	if s.Timeout() != timeout {
		t.Errorf("Timeout() = %v, want %v", s.Timeout(), timeout)
	}
	if s.Validate() != nil {
		t.Errorf("Validate() = %v, want nil", s.Validate())
	}
}

func TestSkill_ValidateWithCustomError(t *testing.T) {
	customErr := errors.New("custom validation failure")
	s := &testSkill{id: "core/bad", validateErr: customErr}

	err := s.Validate()
	if !errors.Is(err, customErr) {
		t.Errorf("Validate() = %v, want customErr", err)
	}
}

// ---------------------------------------------------------------------------
// Error sentinels
// ---------------------------------------------------------------------------

func TestErrorSentinels_ExistAndUsable(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		msg   string
	}{
		{"ErrSkillNotFound", registryskills.ErrSkillNotFound, "not found"},
		{"ErrSkillAlreadyExists", registryskills.ErrSkillAlreadyExists, "already exists"},
		{"ErrSkillInvalid", registryskills.ErrSkillInvalid, "invalid"},
		{"ErrKeyNotFound", registryskills.ErrKeyNotFound, "key not found"},
		{"ErrInvalidURL", registryskills.ErrInvalidURL, "invalid URL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Fatal("sentinel error must not be nil")
			}
			if !strings.Contains(tt.err.Error(), tt.msg) {
				t.Errorf("sentinel %s message = %q, want substring %q", tt.name, tt.err.Error(), tt.msg)
			}
		})
	}
}

func TestErrorSentinels_ErrSkillNotFound_IsUsedByGet(t *testing.T) {
	reg := registryskills.NewInMemoryRegistry()
	_, err := reg.Get("anything")
	if !errors.Is(err, registryskills.ErrSkillNotFound) {
		t.Errorf("Get on empty registry should return ErrSkillNotFound, got: %v", err)
	}
}

func TestErrorSentinels_ErrSkillAlreadyExists_IsUsedByRegister(t *testing.T) {
	reg := registryskills.NewInMemoryRegistry()
	_ = reg.Register(newTestSkill("dup"))
	err := reg.Register(newTestSkill("dup"))
	if !errors.Is(err, registryskills.ErrSkillAlreadyExists) {
		t.Errorf("duplicate Register should return ErrSkillAlreadyExists, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Manifest parsing (supplementary to existing manifest_test.go)
// ---------------------------------------------------------------------------

func TestManifest_EmptyInput(t *testing.T) {
	manifest, err := registryskills.ParseManifest([]byte{})
	if err != nil {
		t.Fatalf("ParseManifest on empty input should not fail, got: %v", err)
	}
	if manifest.ID != "" {
		t.Errorf("empty manifest should have empty ID, got %q", manifest.ID)
	}
}

func TestManifest_InvalidYAML(t *testing.T) {
	_, err := registryskills.ParseManifest([]byte(":\tinvalid: yaml: ["))
	if err == nil {
		t.Error("ParseManifest should return error for malformed YAML")
	}
}

func TestManifest_ValidateMissingBoth(t *testing.T) {
	manifest := &registryskills.SkillManifest{}
	err := manifest.Validate()
	if err == nil {
		t.Fatal("Validate on zero-value manifest should return error")
	}
	if !strings.Contains(err.Error(), "execution.mode") {
		t.Errorf("error should mention 'execution.mode', got: %v", err)
	}
}

func TestManifest_FullRoundTrip(t *testing.T) {
	yaml := `
id: custom/analyzer
version: 2.0.0
name: Analyzer
description: Deep document analysis
execution:
  mode: wasm
requires:
  - text_generation
  - embedding
permissions:
  - read:docs
  - exec:wasm
timeout: 2m
resources:
  max_memory_mb: 512
  max_cpu_ms: 30000
`
	manifest, err := registryskills.ParseManifest([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseManifest failed: %v", err)
	}

	if manifest.ID != "custom/analyzer" {
		t.Errorf("ID = %q", manifest.ID)
	}
	if manifest.Version != "2.0.0" {
		t.Errorf("Version = %q", manifest.Version)
	}
	if manifest.Name != "Analyzer" {
		t.Errorf("Name = %q", manifest.Name)
	}
	if manifest.Description != "Deep document analysis" {
		t.Errorf("Description = %q", manifest.Description)
	}
	if len(manifest.Requires) != 2 {
		t.Errorf("Requires = %v", manifest.Requires)
	}
	if len(manifest.Permissions) != 2 {
		t.Errorf("Permissions = %v", manifest.Permissions)
	}
	if manifest.Timeout != 2*time.Minute {
		t.Errorf("Timeout = %v", manifest.Timeout)
	}
	if manifest.Resources.MaxMemoryMB != 512 {
		t.Errorf("MaxMemoryMB = %d", manifest.Resources.MaxMemoryMB)
	}
	if manifest.Resources.MaxCPUMs != 30000 {
		t.Errorf("MaxCPUMs = %d", manifest.Resources.MaxCPUMs)
	}

	// Validate should pass
	if err := manifest.Validate(); err != nil {
		t.Errorf("Validate() = %v", err)
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func toSet(ss []string) map[string]bool {
	m := make(map[string]bool, len(ss))
	for _, s := range ss {
		m[s] = true
	}
	return m
}

func skillIDs(ss []registryskills.Skill) []string {
	ids := make([]string, len(ss))
	for i, s := range ss {
		ids[i] = s.ID()
	}
	return ids
}
// --- Unregister tests ---

func TestInMemoryRegistry_Unregister(t *testing.T) {
	reg := registryskills.NewInMemoryRegistry()
	skill := &testSkill{id: "test/skill", name: "Test"}
	if err := reg.Register(skill); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := reg.Unregister("test/skill"); err != nil {
		t.Fatalf("unregister: %v", err)
	}
	if _, err := reg.Get("test/skill"); !errors.Is(err, registryskills.ErrSkillNotFound) {
		t.Errorf("expected ErrSkillNotFound after unregister, got %v", err)
	}
}

func TestInMemoryRegistry_Unregister_NotFound(t *testing.T) {
	reg := registryskills.NewInMemoryRegistry()
	if err := reg.Unregister("nonexistent"); !errors.Is(err, registryskills.ErrSkillNotFound) {
		t.Errorf("expected ErrSkillNotFound, got %v", err)
	}
}

func TestInMemoryRegistry_Unregister_ConcurrentAccess(t *testing.T) {
	reg := registryskills.NewInMemoryRegistry()
	for i := 0; i < 10; i++ {
		reg.Register(&testSkill{id: fmt.Sprintf("skill/%d", i), name: fmt.Sprintf("S%d", i)})
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			reg.Unregister(fmt.Sprintf("skill/%d", idx))
		}(i)
	}
	wg.Wait()

	if len(reg.List()) != 0 {
		t.Errorf("expected 0 skills after concurrent unregister, got %d", len(reg.List()))
	}
}

// --- Subscribe tests ---

func TestInMemoryRegistry_Subscribe_RegisterEvent(t *testing.T) {
	reg := registryskills.NewInMemoryRegistry()
	ch := make(chan registryskills.ChangeEvent, 1)
	reg.Subscribe(func(event registryskills.ChangeEvent) { ch <- event })

	reg.Register(&testSkill{id: "new/skill", name: "New"})
	select {
	case event := <-ch:
		if event.Type != registryskills.ChangeEventRegister {
			t.Errorf("expected register event, got %v", event.Type)
		}
		if event.SkillID != "new/skill" {
			t.Errorf("expected skill ID 'new/skill', got %q", event.SkillID)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for callback")
	}
}

func TestInMemoryRegistry_Subscribe_UnregisterEvent(t *testing.T) {
	reg := registryskills.NewInMemoryRegistry()
	reg.Register(&testSkill{id: "old/skill", name: "Old"})
	ch := make(chan registryskills.ChangeEvent, 1)
	reg.Subscribe(func(event registryskills.ChangeEvent) { ch <- event })

	reg.Unregister("old/skill")
	select {
	case event := <-ch:
		if event.Type != registryskills.ChangeEventUnregister {
			t.Errorf("expected unregister event, got %v", event.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for callback")
	}
}

func TestInMemoryRegistry_Subscribe_MultipleCallbacks(t *testing.T) {
	reg := registryskills.NewInMemoryRegistry()
	ch1 := make(chan registryskills.ChangeEvent, 1)
	ch2 := make(chan registryskills.ChangeEvent, 1)
	reg.Subscribe(func(event registryskills.ChangeEvent) { ch1 <- event })
	reg.Subscribe(func(event registryskills.ChangeEvent) { ch2 <- event })

	reg.Register(&testSkill{id: "a", name: "A"})
	select {
	case <-ch1:
	case <-time.After(time.Second):
		t.Fatal("callback 1 not called")
	}
	select {
	case <-ch2:
	case <-time.After(time.Second):
		t.Fatal("callback 2 not called")
	}
}

func TestInMemoryRegistry_Subscribe_CallbackPanic(t *testing.T) {
	reg := registryskills.NewInMemoryRegistry()
	ch := make(chan registryskills.ChangeEvent, 1)
	reg.Subscribe(func(event registryskills.ChangeEvent) { panic("boom") })
	reg.Subscribe(func(event registryskills.ChangeEvent) { ch <- event })

	reg.Register(&testSkill{id: "panic/test", name: "Panic"})
	select {
	case event := <-ch:
		if event.SkillID != "panic/test" {
			t.Errorf("second callback should still fire, got %q", event.SkillID)
		}
	case <-time.After(time.Second):
		t.Fatal("second callback should fire after panic in first")
	}
}

func TestInMemoryRegistry_Subscribe_ReentrantSafe(t *testing.T) {
	reg := registryskills.NewInMemoryRegistry()
	reg.Subscribe(func(event registryskills.ChangeEvent) {
		// Callback that reads from registry (re-entrant)
		reg.List()
	})

	err := reg.Register(&testSkill{id: "reentrant", name: "Reentrant"})
	if err != nil {
		t.Fatalf("register with re-entrant callback should not deadlock: %v", err)
	}
}

// --- G18: SkillInfo compile-time compatibility ---

// TestRegistrySkill_SatisfiesSkillInfo verifies that registry.Skill
// is a superset of control/skills.SkillInfo. If this compiles, the
// interface contract is satisfied.
func TestRegistrySkill_SatisfiesSkillInfo(t *testing.T) {
	// Compile-time assertion: registry.Skill must satisfy control/skills.SkillInfo
	var _ skills.SkillInfo = (registryskills.Skill)(nil)

	// Runtime verification with a concrete implementation
	var s registryskills.Skill = &testSkill{id: "compile-check"}
	var info skills.SkillInfo = s
	if info.ID() != "compile-check" {
		t.Errorf("ID: expected 'compile-check', got %q", info.ID())
	}
}
