package audit

import "testing"

func TestAuditEvent_ExplicitSource(t *testing.T) {
	e := AuditEvent{
		ID:     "evt-1",
		Action: "skills.execute",
		Source: SourceExecutor,
	}
	if e.Source != SourceExecutor {
		t.Errorf("Source = %q, want %q", e.Source, SourceExecutor)
	}
}

func TestAuditEvent_EmptySource(t *testing.T) {
	e := AuditEvent{ID: "evt-2", Action: "test"}
	if e.Source != "" {
		t.Errorf("empty Source should be zero value, got %q", e.Source)
	}
}

func TestToEnvelope_UsesExplicitSource(t *testing.T) {
	e := AuditEvent{
		ID:     "evt-3",
		Action: "skills.execute",
		Source: SourceAdmin,
	}
	env := e.ToEnvelope()
	if env.Source != SourceAdmin {
		t.Errorf("ToEnvelope Source = %q, want explicit %q", env.Source, SourceAdmin)
	}
}

func TestToEnvelope_FallsBackToInferred(t *testing.T) {
	e := AuditEvent{
		ID:     "evt-4",
		Action: "policy.enforce",
	}
	env := e.ToEnvelope()
	if env.Source != SourcePolicy {
		t.Errorf("ToEnvelope inferred Source = %q, want %q", env.Source, SourcePolicy)
	}
}

func TestSource_Constants(t *testing.T) {
	sources := map[Source]bool{
		SourceExecutor:  true,
		SourcePolicy:    true,
		SourceAdmin:     true,
		SourceReasoning: true,
		SourceSystem:    true,
	}
	for _, s := range []Source{SourceExecutor, SourcePolicy, SourceAdmin, SourceReasoning, SourceSystem} {
		if !sources[s] {
			t.Errorf("expected source %q in known sources", s)
		}
	}
}
