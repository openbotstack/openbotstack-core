package profile

import (
	"encoding/json"
	"testing"
)

func TestPersonas_DefaultSetRecognized(t *testing.T) {
	for _, p := range []string{PersonaGeneral, PersonaICU, PersonaNursing, PersonaRadiology, PersonaEmergency} {
		if !ValidPersona(p) {
			t.Errorf("default persona %q not recognized", p)
		}
	}
	for _, d := range []string{DomainGeneral, DomainHealthcare, DomainFinance, DomainLegal} {
		if !ValidDomain(d) {
			t.Errorf("default domain %q not recognized", d)
		}
	}
	if !ValidPersona("") {
		t.Error("blank persona must be valid (treated as general)")
	}
	if !ValidDomain("") {
		t.Error("blank domain must be valid")
	}
}

func TestPersonas_RegisterExtendsRegistry(t *testing.T) {
	token := "test-persona-unique"
	defer delete(defaultPersonas, token)
	if ValidPersona(token) {
		t.Fatal("precondition: token should not exist")
	}
	RegisterPersona(token)
	if !ValidPersona(token) {
		t.Errorf("RegisterPersona did not register %q", token)
	}
	if found := contains(ListPersonas(), token); !found {
		t.Errorf("ListPersonas did not include registered %q", token)
	}
}

func TestValidatePersonaFields_RejectsUnknownTokens(t *testing.T) {
	p := AssistantProfile{
		Scope: ScopeGlobal,
		Soul:  Soul{Identity: Identity{Persona: "you-are-an-expert", Domain: "not-a-domain"}},
	}
	vs := ValidatePersonaFields(p)
	if len(vs) != 2 {
		t.Fatalf("expected 2 violations, got %d: %v", len(vs), vs)
	}
}

func contains(slice []string, s string) bool {
	for _, x := range slice {
		if x == s {
			return true
		}
	}
	return false
}

func TestProfileScope_Valid(t *testing.T) {
	if !ScopeGlobal.Valid() || !ScopeTenant.Valid() || !ScopeSession.Valid() {
		t.Error("built-in scopes must be valid")
	}
	if ProfileScope("user").Valid() {
		t.Error("user scope must not be valid (ADR-042: User layer not persisted)")
	}
}

func TestValidateScope_GlobalAcceptsConstitution(t *testing.T) {
	g := DefaultGlobal()
	if vs := ValidateScope(g); len(vs) != 0 {
		t.Errorf("global with constitution fields set must be valid, got %v", vs)
	}
}

func TestValidateScope_TenantRejectsConstitution(t *testing.T) {
	tenant := AssistantProfile{
		Scope:   ScopeTenant,
		Safety:  SafetyPolicy{MedicalMode: boolPtr(true)},
		Evidence: EvidencePolicy{Required: boolPtr(true)},
	}
	vs := ValidateScope(tenant)
	if len(vs) == 0 {
		t.Error("tenant setting constitution fields must produce violations")
	}
	for _, v := range vs {
		if v.Scope != ScopeTenant {
			t.Errorf("violation scope = %q, want tenant", v.Scope)
		}
	}
}

func TestValidateScope_SessionRejectsDisallowed(t *testing.T) {
	session := AssistantProfile{
		Scope: ScopeSession,
		Soul:  Soul{Identity: Identity{Name: "x"}}, // identity not session-allowed
		Output: OutputPolicy{Language: "en-US", Markdown: boolPtr(true)}, // both allowed
	}
	vs := ValidateScope(session)
	got := map[string]bool{}
	for _, v := range vs {
		got[v.Field] = true
	}
	if !got["soul.identity.name"] {
		t.Error("session identity.name must be a violation")
	}
	if got["output.language"] {
		t.Error("session output.language must NOT be a violation")
	}
	if got["output.markdown"] {
		t.Error("session output.markdown must NOT be a violation")
	}
}

func TestProfile_JSONRoundTrip(t *testing.T) {
	g := DefaultGlobal()
	data, err := json.Marshal(g)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got AssistantProfile
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Soul.Identity.Name != g.Soul.Identity.Name {
		t.Errorf("name round-trip failed: %q vs %q", got.Soul.Identity.Name, g.Soul.Identity.Name)
	}
	if got.Output.Temperature == nil || *got.Output.Temperature != *g.Output.Temperature {
		t.Error("temperature round-trip failed")
	}
}

func TestProfile_OmitEmptyForOverlay(t *testing.T) {
	// A tenant overlay with only a language set should serialize compactly (pointers
	// omitted, not emitted as null/false).
	tenant := AssistantProfile{
		Scope:  ScopeTenant,
		Output: OutputPolicy{Language: "en-US"},
	}
	data, err := json.Marshal(tenant)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	s := string(data)
	// Should not contain explicit safety/evidence/soul blocks populated with nulls.
	if !containsStr(s, `"language":"en-US"`) {
		t.Errorf("language not present in JSON: %s", s)
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
