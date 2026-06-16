package execution

import (
	"errors"
	"time"
)

// Evidence records the provenance of data produced by a Tool/MCP execution
// (ADR-035 Evidence Model). It is the trust channel that lets the audit trail
// reconstruct where every value came from — distinct from StepResult.Output,
// which the LLM may rewrite. Evidence is filled ONLY by the Tool code path;
// the LLM never produces or modifies it.
//
// Canonical medical example: an LLM may emit "bp: 120/80" in Output, but only
// an HIS Observation query produces the Evidence proving that value came from
// a real FHIR resource. Provenance Verify (ADR-036) checks Output ↔ Evidence.
type Evidence struct {
	// Provenance core. Source is the only hard-required field (see Validate);
	// Kind/Resource/ID are recommended but optional because URLs/files carry
	// no FHIR-style business key.
	Source   string // "mcp:his", "builtin.web_fetch", "skill:lab_ref", "computed"
	Kind     string // "fhir:Observation", "hl7:ORU", "row", "http", "computed"
	Resource string // resource type: "Observation" / "Patient" / table name / URL path
	ID       string // resource business key (e.g. FHIR resource id)

	// Globally unique locator for back-tracing the original resource.
	URI string // MCP resource URI / DB row id / full URL

	// Time — medical requires data-occurrence vs query time to be separate.
	// "This BP was taken 6h ago" (Timestamp) ≠ "I queried it 3min ago" (FetchedAt).
	Timestamp time.Time // when the underlying data was produced/collected (NOT query time)
	FetchedAt time.Time // when this system queried the data

	// Correlation & integrity.
	TraceID string // distributed trace / execution_id link
	Hash    string // raw-payload fingerprint (SHA-256), tamper/reconciliation, optional

	// Optional: which Output field this evidence substantiates (e.g. "bp_systolic").
	OutputField string
}

// Validate enforces the minimal Evidence contract. Source is the only
// hard-required field: without it an evidence's origin cannot be distinguished,
// which defeats the model. Kind/Resource/ID/URI are recommended but not
// enforced, because builtin tools over URLs/files legitimately lack a
// FHIR-style business key (see ADR-035 builtin.read_file example).
func (e Evidence) Validate() error {
	if e.Source == "" {
		return errors.New("evidence: Source is required")
	}
	return nil
}

// EvidenceProducer is an optional capability interface (ADR-035) implemented by
// Tools that can compute provenance evidence for a specific call. Detection is
// via type assertion, so tools that reference no external resource (builtin.now,
// builtin.uuid_generate) simply omit it and produce no evidence — fully backward
// compatible. Mirrors the ADR-033 Replanner optional-interface pattern.
//
// ProduceEvidence is a PURE function of the call's input and output: it holds no
// state. This is deliberate — builtin tools are stateless singletons shared
// across concurrent requests, so a stateful LastEvidence() would race.
// ProduceEvidence is called by the runner adapter immediately after a successful
// Execute, with the real call's arguments and result, so the evidence reflects
// an actual external query (real URL, real body hash). The LLM path never calls
// this — evidence is produced only by the Tool code channel.
type EvidenceProducer interface {
	ProduceEvidence(input map[string]any, output map[string]any) []Evidence
}
