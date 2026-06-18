package profile

import (
	"fmt"
	"sort"
)

// Violation records an attempt by a non-Global scope to set a field it is not
// permitted to override (ADR-042 §3 permission matrix).
//
// Violations are informational: the offending value is ignored (not applied), and the
// violation is returned so callers (audit layer, Admin API import path) can record it.
// This avoids hard-failing on partially-valid imported JSON.
type Violation struct {
	Scope ProfileScope `json:"scope"` // the scope that attempted the override
	Field string       `json:"field"`  // dotted path of the field, e.g. "safety.medical_mode"
	Reason string      `json:"reason"` // human-readable explanation
}

// String renders a violation for logs and audit entries.
func (v Violation) String() string {
	return fmt.Sprintf("scope %q may not override %s: %s", v.Scope, v.Field, v.Reason)
}

// ValidateScope checks that every field set on p is permitted for p.Scope, returning
// all violations. Fields that are unset (nil pointers, empty strings, nil slices) are
// not checked — they are "inherit" sentinels, not overrides.
//
// This is the write-path validator: Admin PUT/POST and Import call it and reject the
// request (422) when violations are non-empty. The Merge path uses the same field
// permission matrix but silently ignores offending values instead of erroring.
func ValidateScope(p AssistantProfile) []Violation {
	if !p.Scope.Valid() {
		return []Violation{{Scope: p.Scope, Field: "scope", Reason: "unknown scope"}}
	}
	var vs []Violation

	// Safety and Evidence are constitution fields — only Global may set them.
	if p.Scope != ScopeGlobal {
		vs = append(vs, constitutionViolations(p.Scope, "safety", safetySetFields(p.Safety))...)
		vs = append(vs, constitutionViolations(p.Scope, "evidence", evidenceSetFields(p.Evidence))...)
	}

	// Tenant may not set session-only fields (ADR-042 §3). The write path must reject
	// these — mirrors mergeTenant's read-path enforcement — so invalid rows never persist.
	if p.Scope == ScopeTenant {
		vs = append(vs, tenantSessionOnlyViolations(p)...)
	}

	// Session may only touch a small allow-list; anything else it sets is a violation.
	if p.Scope == ScopeSession {
		vs = append(vs, sessionAllowListViolations(p)...)
	}

	return vs
}

// tenantSessionOnlyViolations reports session-only fields that a tenant profile attempted
// to set. Shared by ValidateScope (write path → 422) and mergeTenant (read path → ignore)
// so both paths enforce the identical ADR-042 §3 matrix.
func tenantSessionOnlyViolations(p AssistantProfile) []Violation {
	var vs []Violation
	if p.Reasoning.ShowReasoning != nil {
		vs = append(vs, Violation{Scope: ScopeTenant, Field: "reasoning.show_reasoning",
			Reason: "field not overridable at tenant scope"})
	}
	if p.Presentation.CompactMode != nil {
		vs = append(vs, Violation{Scope: ScopeTenant, Field: "presentation.compact_mode",
			Reason: "field not overridable at tenant scope"})
	}
	if p.Presentation.Theme != "" {
		vs = append(vs, Violation{Scope: ScopeTenant, Field: "presentation.theme",
			Reason: "field not overridable at tenant scope"})
	}
	return vs
}

// constitutionViolations turns a set of "set field leaf names" for a constitution block
// into Violations prefixed with the block path.
func constitutionViolations(scope ProfileScope, block string, leaves []string) []Violation {
	var vs []Violation
	for _, l := range leaves {
		vs = append(vs, Violation{
			Scope:  scope,
			Field:  block + "." + l,
			Reason: "constitution field, only Global scope may set it",
		})
	}
	return vs
}

// safetySetFields returns the leaf names of SafetyPolicy fields that are explicitly set.
func safetySetFields(s SafetyPolicy) []string {
	var out []string
	if s.HallucinationGuard != nil {
		out = append(out, "hallucination_guard")
	}
	if s.MedicalMode != nil {
		out = append(out, "medical_mode")
	}
	if s.ContentFilter != nil {
		out = append(out, "content_filter")
	}
	return out
}

// evidenceSetFields returns the leaf names of EvidencePolicy fields that are explicitly set.
func evidenceSetFields(e EvidencePolicy) []string {
	var out []string
	if e.Required != nil {
		out = append(out, "required")
	}
	if len(e.RequiredFields) > 0 {
		out = append(out, "required_fields")
	}
	if e.CitationRequired != nil {
		out = append(out, "citation_required")
	}
	return out
}

// sessionAllowedFields is the closed allow-list of leaf paths a Session scope may set.
// Everything else Session attempts to set is a violation. Per ADR-042 §3.
var sessionAllowedFields = map[string]bool{
	"soul.behavior.language":     true,
	"reasoning.show_reasoning":   true,
	"output.language":            true,
	"output.markdown":            true,
	"presentation.compact_mode":  true,
	"presentation.theme":         true,
	// Non-field metadata a Session overlay may carry without it counting as a violation:
	"id":          true,
	"name":        true,
	"description": true,
	"scope":       true,
	"tenant_id":   true,
}

// sessionAllowListViolations inspects every set field on a Session-scoped profile and
// reports any that fall outside sessionAllowedFields.
func sessionAllowListViolations(p AssistantProfile) []Violation {
	var vs []Violation
	report := func(path string, set bool) {
		if set && !sessionAllowedFields[path] {
			vs = append(vs, Violation{
				Scope:  ScopeSession,
				Field:  path,
				Reason: "field not overridable at session scope",
			})
		}
	}

	// Soul.Identity — none of these are session-allowed.
	report("soul.identity.name", p.Soul.Identity.Name != "")
	report("soul.identity.description", p.Soul.Identity.Description != "")
	report("soul.identity.persona", p.Soul.Identity.Persona != "")
	report("soul.identity.domain", p.Soul.Identity.Domain != "")
	report("soul.identity.avatar", p.Soul.Identity.Avatar != "")
	// Soul.Behavior
	report("soul.behavior.tone", p.Soul.Behavior.Tone != "")
	report("soul.behavior.language", p.Soul.Behavior.Language != "")
	report("soul.behavior.citations", p.Soul.Behavior.Citations != nil)
	// Reasoning
	report("reasoning.enabled", p.Reasoning.Enabled != nil)
	report("reasoning.show_reasoning", p.Reasoning.ShowReasoning != nil)
	// Output
	report("output.language", p.Output.Language != "")
	report("output.markdown", p.Output.Markdown != nil)
	report("output.citations", p.Output.Citations != nil)
	report("output.temperature", p.Output.Temperature != nil)
	// Presentation
	report("presentation.reasoning_summary", p.Presentation.ReasoningSummary != nil)
	report("presentation.compact_mode", p.Presentation.CompactMode != nil)
	report("presentation.theme", p.Presentation.Theme != "")

	// Sort for deterministic output (audit friendliness).
	sort.Slice(vs, func(i, j int) bool { return vs[i].Field < vs[j].Field })
	return vs
}

// SortedViolations returns a copy of vs sorted by (Scope, Field) for stable comparison.
func SortedViolations(vs []Violation) []Violation {
	out := make([]Violation, len(vs))
	copy(out, vs)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Scope != out[j].Scope {
			return string(out[i].Scope) < string(out[j].Scope)
		}
		return out[i].Field < out[j].Field
	})
	return out
}

