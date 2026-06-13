package template

import (
	"strings"
	"testing"
)

// mustResolve fails the test if Resolve returns an error; returns the value.
func mustResolve(t *testing.T, s string, results map[string]any) any {
	t.Helper()
	got, err := Resolve(s, results)
	if err != nil {
		t.Fatalf("Resolve(%q) returned unexpected error: %v", s, err)
	}
	return got
}

func TestResolve_SimpleName(t *testing.T) {
	results := map[string]any{"math-add": float64(42)}
	got := mustResolve(t, "{{math-add}}", results)
	if got != "42" {
		t.Errorf("Resolve({{math-add}}) = %v, want 42", got)
	}
}

func TestResolve_SimpleNameFieldAccess(t *testing.T) {
	results := map[string]any{
		"math-add": map[string]any{"sum": float64(42)},
	}
	got := mustResolve(t, "{{math-add.sum}}", results)
	if got != "42" {
		t.Errorf("Resolve({{math-add.sum}}) = %v, want 42", got)
	}
}

func TestResolve_MCPDottedStepName(t *testing.T) {
	results := map[string]any{
		"mcp.his.query_patient": map[string]any{"name": "John", "age": 65},
	}
	got := mustResolve(t, "{{mcp.his.query_patient}}", results)
	want := `{"age":65,"name":"John"}`
	if got != want {
		t.Errorf("Resolve({{mcp.his.query_patient}}) = %v, want %v", got, want)
	}
}

func TestResolve_MCPDottedStepNameFieldAccess(t *testing.T) {
	results := map[string]any{
		"mcp.his.query_patient": map[string]any{"name": "John", "age": 65},
	}
	got := mustResolve(t, "{{mcp.his.query_patient.name}}", results)
	if got != "John" {
		t.Errorf("Resolve({{mcp.his.query_patient.name}}) = %v, want John", got)
	}
}

func TestResolve_MCPDottedStepNameNestedFieldAccess(t *testing.T) {
	results := map[string]any{
		"mcp.his.query_patient": map[string]any{
			"data": map[string]any{"field": "value1"},
		},
	}
	got := mustResolve(t, "{{mcp.his.query_patient.data.field}}", results)
	if got != "value1" {
		t.Errorf("Resolve({{mcp.his.query_patient.data.field}}) = %v, want value1", got)
	}
}

func TestResolve_MCPDottedNameEmbedded(t *testing.T) {
	results := map[string]any{
		"mcp.his.query_patient": "John",
	}
	got := mustResolve(t, "Patient: {{mcp.his.query_patient}}", results)
	if got != "Patient: John" {
		t.Errorf("Resolve(embedded) = %v, want Patient: John", got)
	}
}

func TestResolve_MCPDottedNameUnresolved_ReturnsError(t *testing.T) {
	// Unresolved step references MUST return an error — no silent fallback
	// to the original template string.
	results := map[string]any{"other": 1}
	_, err := Resolve("{{mcp.his.query_patient}}", results)
	if err == nil {
		t.Fatal("Resolve(unresolved step) should return an error, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found': %v", err)
	}
}

func TestResolve_FieldNotFound_ReturnsErrorAndListsKeys(t *testing.T) {
	// The headline bug: a field the planner guessed (e.g. "content" when the
	// result has "text") must surface a clear error listing available keys.
	results := map[string]any{
		"resource_read": map[string]any{"text": "hello", "title": "T"},
	}
	_, err := Resolve("{{resource_read.content}}", results)
	if err == nil {
		t.Fatal("Resolve(missing field) should return an error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "content") {
		t.Errorf("error should name the missing field: %v", err)
	}
	if !strings.Contains(msg, "text") || !strings.Contains(msg, "title") {
		t.Errorf("error should list available keys: %v", err)
	}
}

func TestResolve_MCPMultipleDottedRefs(t *testing.T) {
	results := map[string]any{
		"mcp.his.query_patient":  "John",
		"mcp.vitals.get_vitals":  "120/80",
	}
	got := mustResolve(t, "{{mcp.his.query_patient}} has BP {{mcp.vitals.get_vitals}}", results)
	if got != "John has BP 120/80" {
		t.Errorf("Resolve(multiple) = %v, want 'John has BP 120/80'", got)
	}
}

func TestResolve_FullTypePreservation_SimpleName(t *testing.T) {
	results := map[string]any{"step1": float64(42)}
	got := mustResolve(t, "{{step1}}", results)
	if got != "42" {
		t.Errorf("Resolve({{step1}}) = %v (%T), want 42", got, got)
	}
}

func TestResolve_FullTypePreservation_MCPName(t *testing.T) {
	results := map[string]any{
		"mcp.his.query_patient": map[string]any{"name": "John"},
	}
	got := mustResolve(t, "{{mcp.his.query_patient}}", results)
	want := `{"name":"John"}`
	if got != want {
		t.Errorf("Resolve(full MCP) = %v, want %v", got, want)
	}
}

func TestResolve_AmbiguousDots_PrefersExactMatch(t *testing.T) {
	// Both "mcp" and "mcp.his.query_patient" exist in results
	results := map[string]any{
		"mcp":                   map[string]any{"his": "short"},
		"mcp.his.query_patient": map[string]any{"name": "exact"},
	}
	got := mustResolve(t, "{{mcp.his.query_patient.name}}", results)
	if got != "exact" {
		// Should prefer the exact key match, not fall back to mcp.his.query_patient via nested field
		t.Errorf("Resolve(ambiguous) = %v, want exact (from exact key match)", got)
	}
}

func TestResolve_AmbiguousDots_FallsBackToShorterKey(t *testing.T) {
	results := map[string]any{
		"mcp": map[string]any{
			"his": map[string]any{
				"query_patient": "fallback_value",
			},
		},
	}
	got := mustResolve(t, "{{mcp.his.query_patient}}", results)
	if got != "fallback_value" {
		t.Errorf("Resolve(fallback) = %v, want fallback_value", got)
	}
}

func TestResolve_NoTemplates(t *testing.T) {
	got := mustResolve(t, "hello world", map[string]any{"x": 1})
	if got != "hello world" {
		t.Errorf("Resolve(no templates) = %v, want hello world", got)
	}
}

func TestResolve_EmptyResults(t *testing.T) {
	// No prior results (e.g. the first step) → nothing to resolve against.
	// Returns the original string with no error: this is a legitimate
	// "no context yet" state, not a resolution failure.
	got, err := Resolve("{{step1}}", nil)
	if err != nil {
		t.Fatalf("Resolve with nil results should not error: %v", err)
	}
	if got != "{{step1}}" {
		t.Errorf("Resolve(nil results) = %v, want original {{step1}}", got)
	}
}

func TestResolve_NilArguments(t *testing.T) {
	got := mustResolve(t, "", map[string]any{"x": 1})
	if got != "" {
		t.Errorf("Resolve(empty string) = %v, want empty", got)
	}
}
