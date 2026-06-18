package profile

import (
	"reflect"
	"testing"
)

// boolVal dereferences a *bool for test comparison, returning false for nil.
func boolVal(b *bool) bool { return b != nil && *b }

func TestMerge_GlobalOnly_EffectiveEqualsGlobal(t *testing.T) {
	g := DefaultGlobal()
	eff, vs := Merge(g, nil, nil)
	if len(vs) != 0 {
		t.Fatalf("expected no violations, got %d: %v", len(vs), vs)
	}
	if eff.Soul.Identity.Name != g.Soul.Identity.Name {
		t.Errorf("effective identity name = %q, want %q", eff.Soul.Identity.Name, g.Soul.Identity.Name)
	}
	if eff.Scope != ScopeGlobal {
		t.Errorf("effective scope = %q, want global", eff.Scope)
	}
	if eff.TenantID != "" {
		t.Errorf("effective tenant_id should be empty, got %q", eff.TenantID)
	}
}

func TestMerge_TenantOverridesAllowedFields(t *testing.T) {
	g := DefaultGlobal()
	tenant := &AssistantProfile{
		Scope: ScopeTenant,
		Soul: Soul{Identity: Identity{Name: "ICU Bot", Persona: PersonaICU, Domain: DomainHealthcare},
			Behavior: Behavior{Tone: "detailed", Language: "en-US"}},
		Output: OutputPolicy{Language: "en-US"},
	}
	eff, vs := Merge(g, tenant, nil)
	if len(vs) != 0 {
		t.Fatalf("expected no violations, got %v", vs)
	}
	if eff.Soul.Identity.Name != "ICU Bot" {
		t.Errorf("tenant name override failed: %q", eff.Soul.Identity.Name)
	}
	if eff.Soul.Identity.Persona != PersonaICU {
		t.Errorf("tenant persona override failed: %q", eff.Soul.Identity.Persona)
	}
	if eff.Soul.Behavior.Tone != "detailed" {
		t.Errorf("tenant tone override failed: %q", eff.Soul.Behavior.Tone)
	}
	if eff.Soul.Behavior.Language != "en-US" {
		t.Errorf("tenant behavior language override failed: %q", eff.Soul.Behavior.Language)
	}
	if eff.Output.Language != "en-US" {
		t.Errorf("tenant output language override failed: %q", eff.Output.Language)
	}
}

func TestMerge_TenantCannotOverrideConstitution(t *testing.T) {
	g := DefaultGlobal()
	originalSafety := g.Safety
	tenant := &AssistantProfile{
		Scope:   ScopeTenant,
		Safety:  SafetyPolicy{HallucinationGuard: boolPtr(false), MedicalMode: boolPtr(false)},
		Evidence: EvidencePolicy{Required: boolPtr(false)},
	}
	eff, vs := Merge(g, tenant, nil)

	// Constitution fields must be ignored AND recorded as violations.
	if !reflect.DeepEqual(eff.Safety, originalSafety) {
		t.Errorf("tenant safety leaked into effective: %+v", eff.Safety)
	}
	wantFields := map[string]bool{
		"safety.hallucination_guard": true,
		"safety.medical_mode":        true,
		"evidence.required":          true,
	}
	gotFields := map[string]bool{}
	for _, v := range vs {
		if v.Scope == ScopeTenant {
			gotFields[v.Field] = true
		}
	}
	for f := range wantFields {
		if !gotFields[f] {
			t.Errorf("expected violation for %s, got %v", f, gotFields)
		}
	}
}

func TestMerge_TenantCannotOverrideSessionOnlyFields(t *testing.T) {
	g := DefaultGlobal()
	tenant := &AssistantProfile{
		Scope:         ScopeTenant,
		Reasoning:     ReasoningPolicy{ShowReasoning: boolPtr(true)},
		Presentation:  PresentationPolicy{CompactMode: boolPtr(true), Theme: "dark"},
	}
	eff, vs := Merge(g, tenant, nil)

	// ShowReasoning should stay at global default (false), compact_mode/theme untouched.
	if boolVal(eff.Reasoning.ShowReasoning) != false {
		t.Errorf("tenant overrode session-only reasoning.show_reasoning")
	}
	if boolVal(eff.Presentation.CompactMode) != false {
		t.Errorf("tenant overrode session-only presentation.compact_mode")
	}
	if eff.Presentation.Theme != "light" {
		t.Errorf("tenant overrode session-only presentation.theme: %q", eff.Presentation.Theme)
	}
	want := []string{"reasoning.show_reasoning", "presentation.compact_mode", "presentation.theme"}
	got := map[string]bool{}
	for _, v := range vs {
		got[v.Field] = true
	}
	for _, w := range want {
		if !got[w] {
			t.Errorf("missing violation for %s", w)
		}
	}
}

func TestMerge_SessionOverridesAllowedFields(t *testing.T) {
	g := DefaultGlobal()
	session := &AssistantProfile{
		Scope:         ScopeSession,
		Soul:          Soul{Behavior: Behavior{Language: "en-US"}},
		Reasoning:     ReasoningPolicy{ShowReasoning: boolPtr(true)},
		Output:        OutputPolicy{Language: "en-US", Markdown: boolPtr(false)},
		Presentation:  PresentationPolicy{CompactMode: boolPtr(true), Theme: "dark"},
	}
	eff, vs := Merge(g, nil, session)
	if len(vs) != 0 {
		t.Fatalf("session-only override should not violate, got %v", vs)
	}
	if eff.Soul.Behavior.Language != "en-US" {
		t.Errorf("session behavior.language failed: %q", eff.Soul.Behavior.Language)
	}
	if !boolVal(eff.Reasoning.ShowReasoning) {
		t.Errorf("session reasoning.show_reasoning failed")
	}
	if eff.Output.Language != "en-US" {
		t.Errorf("session output.language failed: %q", eff.Output.Language)
	}
	if boolVal(eff.Output.Markdown) != false {
		t.Errorf("session output.markdown failed")
	}
	if !boolVal(eff.Presentation.CompactMode) {
		t.Errorf("session presentation.compact_mode failed")
	}
	if eff.Presentation.Theme != "dark" {
		t.Errorf("session presentation.theme failed: %q", eff.Presentation.Theme)
	}
}

func TestMerge_SessionCannotOverrideDisallowedFields(t *testing.T) {
	g := DefaultGlobal()
	session := &AssistantProfile{
		Scope:       ScopeSession,
		Soul:        Soul{Identity: Identity{Name: "Hacked"}, Behavior: Behavior{Tone: "warm"}},
		Reasoning:   ReasoningPolicy{Enabled: boolPtr(false)},
		Output:      OutputPolicy{Citations: boolPtr(false), Temperature: floatPtr(2.0)},
		Presentation: PresentationPolicy{ReasoningSummary: boolPtr(true)},
	}
	eff, vs := Merge(g, nil, session)

	// None of the disallowed fields should take effect.
	if eff.Soul.Identity.Name != g.Soul.Identity.Name {
		t.Errorf("session overrode identity.name: %q", eff.Soul.Identity.Name)
	}
	if boolVal(eff.Reasoning.Enabled) != boolVal(g.Reasoning.Enabled) {
		t.Errorf("session overrode reasoning.enabled")
	}
	if boolVal(eff.Output.Citations) != boolVal(g.Output.Citations) {
		t.Errorf("session overrode output.citations")
	}
	// And every disallowed set field must be a violation.
	wantFields := []string{
		"soul.identity.name", "soul.behavior.tone", "reasoning.enabled",
		"output.citations", "output.temperature", "presentation.reasoning_summary",
	}
	got := map[string]bool{}
	for _, v := range vs {
		got[v.Field] = true
	}
	for _, w := range wantFields {
		if !got[w] {
			t.Errorf("missing session violation for %s (got %v)", w, got)
		}
	}
}

func TestMerge_NilPointersInherit(t *testing.T) {
	g := DefaultGlobal()
	// Tenant provides only the language string; all pointers nil.
	tenant := &AssistantProfile{
		Scope:  ScopeTenant,
		Output: OutputPolicy{Language: "en-US"},
	}
	eff, vs := Merge(g, tenant, nil)
	if len(vs) != 0 {
		t.Fatalf("unexpected violations: %v", vs)
	}
	// Citations/Temperature must remain the global values, not zero.
	if !boolVal(eff.Output.Citations) {
		t.Errorf("global output.citations lost during merge (inherit failed)")
	}
	if eff.Output.Temperature == nil || *eff.Output.Temperature != 0.3 {
		t.Errorf("global output.temperature lost during merge")
	}
}

func TestMerge_SessionOverlaysOnTenant(t *testing.T) {
	g := DefaultGlobal()
	tenant := &AssistantProfile{Scope: ScopeTenant, Output: OutputPolicy{Language: "en-US"}}
	session := &AssistantProfile{Scope: ScopeSession, Output: OutputPolicy{Language: "ja-JP"}}
	eff, vs := Merge(g, tenant, session)
	if len(vs) != 0 {
		t.Fatalf("unexpected violations: %v", vs)
	}
	// Session language should win over tenant.
	if eff.Output.Language != "ja-JP" {
		t.Errorf("session language did not overlay tenant: %q", eff.Output.Language)
	}
}

func TestMerge_ViolationsSortedAndStable(t *testing.T) {
	g := DefaultGlobal()
	tenant := &AssistantProfile{
		Scope:        ScopeTenant,
		Safety:       SafetyPolicy{MedicalMode: boolPtr(true)},
		Reasoning:    ReasoningPolicy{ShowReasoning: boolPtr(true)},
		Evidence:     EvidencePolicy{Required: boolPtr(false)},
	}
	_, vs := Merge(g, tenant, nil)
	// Must be sorted by field for stable audit output.
	for i := 1; i < len(vs); i++ {
		if vs[i-1].Field > vs[i].Field {
			t.Errorf("violations not sorted: %q before %q", vs[i-1].Field, vs[i].Field)
		}
	}
}

func TestDefaultGlobal_HasNoNilPointers(t *testing.T) {
	g := DefaultGlobal()
	checks := []struct {
		name string
		ptr  *bool
	}{
		{"reasoning.enabled", g.Reasoning.Enabled},
		{"reasoning.show_reasoning", g.Reasoning.ShowReasoning},
		{"safety.hallucination_guard", g.Safety.HallucinationGuard},
		{"safety.medical_mode", g.Safety.MedicalMode},
		{"safety.content_filter", g.Safety.ContentFilter},
		{"evidence.required", g.Evidence.Required},
		{"evidence.citation_required", g.Evidence.CitationRequired},
		{"output.markdown", g.Output.Markdown},
		{"output.citations", g.Output.Citations},
		{"presentation.reasoning_summary", g.Presentation.ReasoningSummary},
		{"presentation.compact_mode", g.Presentation.CompactMode},
	}
	for _, c := range checks {
		if c.ptr == nil {
			t.Errorf("DefaultGlobal has nil pointer at %s", c.name)
		}
	}
	if g.Output.Temperature == nil {
		t.Error("DefaultGlobal has nil output.temperature")
	}
}
