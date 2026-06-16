package execution

import (
	"testing"
	"time"
)

// ADR-035 Evidence Model. Evidence is the provenance channel produced by
// Tool/MCP execution; LLM never writes it. These tests lock down the data
// contract and the EvidenceProducer optional interface.

func TestEvidence_Validate_RejectsEmptySource(t *testing.T) {
	// Source is the only hard-required field — without it the evidence's
	// origin cannot be distinguished, which defeats the entire model.
	e := Evidence{Kind: "http", URI: "https://example.com"}
	if err := e.Validate(); err == nil {
		t.Error("Evidence without Source must fail validation")
	}
}

func TestEvidence_Validate_AcceptsSourceOnly(t *testing.T) {
	// ADR-035 builtin.read_file example: {Source, URI, Hash} — no FHIR-style
	// Resource/ID because files/URLs have no business primary key. Validate
	// must accept Source-only evidence (Resource/ID are recommended, not hard).
	e := Evidence{Source: "builtin.read_file", URI: "/etc/hosts", Hash: "abc123"}
	if err := e.Validate(); err != nil {
		t.Errorf("Source-only evidence should be valid (no Resource/ID): %v", err)
	}
}

func TestEvidence_Validate_AcceptsFullMedicalRecord(t *testing.T) {
	// ADR-035 canonical medical case: a FHIR Observation from HIS.
	e := Evidence{
		Source:    "mcp:his",
		Kind:      "fhir:Observation",
		Resource:  "Observation",
		ID:        "obs-123",
		URI:       "mcp://his/Observation/obs-123",
		Timestamp: time.Now().Add(-6 * time.Hour),
		FetchedAt: time.Now(),
		Hash:      "sha256-deadbeef",
	}
	if err := e.Validate(); err != nil {
		t.Errorf("full medical evidence should be valid: %v", err)
	}
}

// mockEvidenceTool proves the EvidenceProducer optional-interface contract:
// any struct exposing a pure ProduceEvidence(input, output) satisfies it, and
// callers detect the capability via type assertion without forcing every tool
// to implement it (builtin.now legitimately produces no evidence).
type mockEvidenceTool struct{}

func (m *mockEvidenceTool) ProduceEvidence(input map[string]any, _ map[string]any) []Evidence {
	return []Evidence{{Source: "mcp:his", URI: input["uri"].(string)}}
}

func TestEvidenceProducer_TypeAssertion(t *testing.T) {
	// A tool that produces evidence is detected via optional interface assertion.
	producing := &mockEvidenceTool{}
	ep, ok := any(producing).(EvidenceProducer)
	if !ok {
		t.Fatal("tool implementing ProduceEvidence must satisfy EvidenceProducer")
	}
	got := ep.ProduceEvidence(map[string]any{"uri": "mcp://his/Observation/1"}, nil)
	if len(got) != 1 || got[0].Source != "mcp:his" || got[0].URI != "mcp://his/Observation/1" {
		t.Errorf("ProduceEvidence returned %v, want one mcp:his evidence with URI", got)
	}

	// A bare struct (builtin.now analog) does NOT satisfy it — no evidence, no error.
	var bare struct{}
	if _, ok := any(bare).(EvidenceProducer); ok {
		t.Error("a non-evidence tool must not satisfy EvidenceProducer")
	}
}
