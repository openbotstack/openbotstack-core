// Package profile defines the structured Assistant Profile model for OpenBotStack.
//
// This package establishes the boundary between **persona** (Soul) and **governance**
// (Profile), per ADR-042. An AssistantProfile is the configuration-layer single source
// of truth: who the assistant is (Soul) and how it is governed and presented (Policies).
//
// Scope layering (ADR-042 §3):
//
//	Global  → platform defaults + locked constitution fields (Safety/Evidence).
//	Tenant  → persona / output / presentation overrides.
//	Session → lightweight runtime preferences (not persisted).
//
// All bool fields that participate in scope merging are pointers (*bool) so that
// "not provided" (nil) is distinguishable from "explicitly false". String fields use
// the empty string as the "not provided" sentinel; *float64 uses nil.
//
// This package is pure: no I/O, no side effects, no execution. It defines types,
// a merge algorithm (pure function), a scope permission matrix, and persona validation.
package profile

// ProfileScope identifies the configuration layer a profile belongs to.
type ProfileScope string

const (
	// ScopeGlobal is the platform-wide default. Constitution fields
	// (Safety/Evidence) are only effective at this scope.
	ScopeGlobal ProfileScope = "global"
	// ScopeTenant is a per-tenant override layer.
	ScopeTenant ProfileScope = "tenant"
	// ScopeSession is a per-session runtime preference overlay. It is never persisted.
	ScopeSession ProfileScope = "session"
)

// String returns the canonical string form of the scope.
func (s ProfileScope) String() string { return string(s) }

// Valid reports whether s is a recognized scope.
func (s ProfileScope) Valid() bool {
	switch s {
	case ScopeGlobal, ScopeTenant, ScopeSession:
		return true
	}
	return false
}

// Soul is the persona layer: who the assistant is.
//
// Per ADR-042, Soul ≠ Profile. Soul holds identity and behavior only; governance
// lives in the Policy fields of AssistantProfile. Identity.Persona is a controlled
// vocabulary token (see personas.go), never a free-form "You are..." prompt.
type Soul struct {
	Identity Identity `json:"identity"`
	Behavior Behavior `json:"behavior"`
}

// Identity is the stable identity of an assistant.
type Identity struct {
	Name        string `json:"name"`                  // display name
	Description string `json:"description,omitempty"` // one-line description
	Persona     string `json:"persona,omitempty"`     // controlled vocabulary token
	Domain      string `json:"domain,omitempty"`      // controlled vocabulary token
	Avatar      string `json:"avatar,omitempty"`      // optional avatar URL
}

// Behavior captures surface-level conversational style.
type Behavior struct {
	Tone      string `json:"tone,omitempty"`      // concise/detailed/formal/warm
	Language  string `json:"language,omitempty"`  // BCP-47 tag, e.g. zh-CN/en-US
	Citations *bool  `json:"citations,omitempty"` // whether to cite evidence
}

// ReasoningPolicy governs how reasoning is surfaced. It does NOT carry loop
// boundaries (MaxTurns etc.) — those are platform constants (ADR-042 §2).
type ReasoningPolicy struct {
	Enabled       *bool `json:"enabled,omitempty"`        // whether reasoning steps run
	ShowReasoning *bool `json:"show_reasoning,omitempty"` // whether reasoning is shown to the user
}

// SafetyPolicy is a constitution field: only effective at Global scope.
// Tenant/Session attempts to set it are rejected (422 on write) or ignored (Merge).
//
// This is the governance safety policy (ADR-042). It is distinct from
// core/ai/safety.SafetyPolicy, which is a content-filtering configuration
// (regex-based input/output filtering). The two types live in different packages
// and serve different layers of the platform.
type SafetyPolicy struct {
	HallucinationGuard *bool `json:"hallucination_guard,omitempty"` // force evidence to counter hallucination
	MedicalMode        *bool `json:"medical_mode,omitempty"`        // stricter medical mode
	ContentFilter      *bool `json:"content_filter,omitempty"`      // content filtering
}

// EvidencePolicy is a constitution field: only effective at Global scope.
type EvidencePolicy struct {
	Required         *bool    `json:"required,omitempty"`
	RequiredFields   []string `json:"required_fields,omitempty"`
	CitationRequired *bool    `json:"citation_required,omitempty"`
}

// OutputPolicy governs output shape and language.
type OutputPolicy struct {
	Language    string   `json:"language,omitempty"`
	Markdown    *bool    `json:"markdown,omitempty"`
	Citations   *bool    `json:"citations,omitempty"`
	Temperature *float64 `json:"temperature,omitempty"` // 0–2; pointer distinguishes "unset" from 0
}

// PresentationPolicy governs how results are presented in the client.
type PresentationPolicy struct {
	ReasoningSummary *bool  `json:"reasoning_summary,omitempty"`
	CompactMode      *bool  `json:"compact_mode,omitempty"`
	Theme            string `json:"theme,omitempty"` // light/dark
}

// AssistantProfile is the configuration-layer single source of truth for an assistant.
//
// It combines the persona (Soul) with governance policies. It intentionally does NOT
// contain execution boundaries — LoopBoundary, Audit, Verify, and CompletionCheck are
// platform constants (ADR-042 §2), not configurable profile fields.
//
// A single AssistantProfile value serves two roles depending on Scope:
//   - ScopeGlobal: a fully-populated default (all meaningful fields set).
//   - ScopeTenant / ScopeSession: an overlay where unset fields (nil pointers, empty
//     strings) mean "inherit from the layer above".
type AssistantProfile struct {
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	Description  string             `json:"description,omitempty"`
	Scope        ProfileScope       `json:"scope"`
	TenantID     string             `json:"tenant_id,omitempty"` // set when Scope == ScopeTenant
	Soul         Soul               `json:"soul"`
	Reasoning    ReasoningPolicy    `json:"reasoning"`
	Safety       SafetyPolicy       `json:"safety"`       // effective only at Global scope
	Evidence     EvidencePolicy     `json:"evidence"`     // effective only at Global scope
	Output       OutputPolicy       `json:"output"`
	Presentation PresentationPolicy `json:"presentation"`
}

// boolPtr is a small helper for constructing profiles in tests and seeding defaults.
func boolPtr(b bool) *bool { return &b }

// floatPtr is a small helper for constructing profiles in tests and seeding defaults.
func floatPtr(f float64) *float64 { return &f }
