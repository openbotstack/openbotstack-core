package profile

// Merge combines a Global profile with optional Tenant and Session overlays into a
// single EffectiveProfile, applying the ADR-042 §3 permission matrix.
//
// Semantics:
//
//   - The effective profile starts as a copy of global.
//   - For each Tenant-set field that Tenant is permitted to override, the Tenant value
//     replaces the effective value. Constitution fields (Safety/Evidence) set by Tenant
//     are ignored and recorded as Violations.
//   - Session is then applied the same way, restricted to the Session allow-list.
//   - A nil overlay pointer (or empty string / nil slice) means "inherit" — it never
//     overwrites the effective value with a zero.
//
// Merge is a pure function: deterministic, no I/O, no side effects. It is safe to call
// concurrently. Violations are returned (never panicked) so imported or partially-valid
// overlays degrade gracefully and leave an audit trail.
//
// Field permission matrix (ADR-042):
//
//	field                          | Global | Tenant | Session
//	-------------------------------|--------|--------|--------
//	soul.identity.*                |  def   |   ✓    |   ✗
//	soul.behavior.tone/citations   |  def   |   ✓    |   ✗
//	soul.behavior.language         |  def   |   ✓    |   ✓
//	reasoning.enabled              |  def   |   ✓    |   ✗
//	reasoning.show_reasoning       |  def   |   ✗     |   ✓
//	safety.*                       |  LOCK  |   ✗    |   ✗
//	evidence.*                     |  LOCK  |   ✗    |   ✗
//	output.language                |  def   |   ✓    |   ✓
//	output.markdown                |  def   |   ✓    |   ✓
//	output.citations               |  def   |   ✓    |   ✗
//	output.temperature             |  def   |   ✓    |   ✗
//	presentation.reasoning_summary |  def   |   ✓    |   ✗
//	presentation.compact_mode      |  def   |   ✗     |   ✓
//	presentation.theme             |  def   |   ✗     |   ✓
func Merge(global AssistantProfile, tenant, session *AssistantProfile) (AssistantProfile, []Violation) {
	var violations []Violation

	// Effective starts as a deep-ish copy of global. Pointers are shared but never
	// mutated here (we only reassign effective's fields, not the pointed-to values),
	// so the caller's global is not modified.
	effective := global
	effective.Scope = ScopeGlobal // effective profile is always reported as global-shaped
	effective.TenantID = ""
	effective.ID = global.ID

	if tenant != nil {
		violations = append(violations, mergeTenant(&effective, *tenant)...)
	}
	if session != nil {
		violations = append(violations, mergeSession(&effective, *session)...)
	}

	return effective, SortedViolations(violations)
}

// mergeTenant applies tenant-allowed overrides onto effective and records violations
// for fields the tenant attempted to set but is not permitted to (constitution +
// session-only). Session-only violations are produced by the shared
// tenantSessionOnlyViolations helper so the read path matches ValidateScope exactly.
func mergeTenant(effective *AssistantProfile, tenant AssistantProfile) []Violation {
	vs := tenantSessionOnlyViolations(tenant)

	// --- Soul.Identity: all tenant-allowed ---
	mergeString(&effective.Soul.Identity.Name, tenant.Soul.Identity.Name)
	mergeString(&effective.Soul.Identity.Description, tenant.Soul.Identity.Description)
	mergeString(&effective.Soul.Identity.Persona, tenant.Soul.Identity.Persona)
	mergeString(&effective.Soul.Identity.Domain, tenant.Soul.Identity.Domain)
	mergeString(&effective.Soul.Identity.Avatar, tenant.Soul.Identity.Avatar)

	// --- Soul.Behavior: tone + citations tenant-allowed; language tenant-allowed ---
	mergeString(&effective.Soul.Behavior.Tone, tenant.Soul.Behavior.Tone)
	mergeString(&effective.Soul.Behavior.Language, tenant.Soul.Behavior.Language)
	mergeBoolPtr(&effective.Soul.Behavior.Citations, tenant.Soul.Behavior.Citations)

	// --- Reasoning: enabled tenant-allowed; show_reasoning session-only (recorded above) ---
	mergeBoolPtr(&effective.Reasoning.Enabled, tenant.Reasoning.Enabled)

	// --- Safety / Evidence: constitution. Any set field is a violation, ignored. ---
	vs = append(vs, constitutionViolations(ScopeTenant, "safety", safetySetFields(tenant.Safety))...)
	vs = append(vs, constitutionViolations(ScopeTenant, "evidence", evidenceSetFields(tenant.Evidence))...)

	// --- Output: language + markdown tenant-allowed; citations + temperature tenant-allowed ---
	mergeString(&effective.Output.Language, tenant.Output.Language)
	mergeBoolPtr(&effective.Output.Markdown, tenant.Output.Markdown)
	mergeBoolPtr(&effective.Output.Citations, tenant.Output.Citations)
	mergeFloatPtr(&effective.Output.Temperature, tenant.Output.Temperature)

	// --- Presentation: reasoning_summary tenant-allowed; compact_mode + theme session-only (above) ---
	mergeBoolPtr(&effective.Presentation.ReasoningSummary, tenant.Presentation.ReasoningSummary)

	return vs
}

// mergeSession applies only session-allowed overrides onto effective.
func mergeSession(effective *AssistantProfile, session AssistantProfile) []Violation {
	// Session set fields outside its allow-list are recorded as violations via
	// ValidateScope semantics; reuse that machinery for consistency.
	vs := ValidateScope(session)

	// Apply only allow-listed fields.
	mergeString(&effective.Soul.Behavior.Language, session.Soul.Behavior.Language)
	mergeBoolPtr(&effective.Reasoning.ShowReasoning, session.Reasoning.ShowReasoning)
	mergeString(&effective.Output.Language, session.Output.Language)
	mergeBoolPtr(&effective.Output.Markdown, session.Output.Markdown)
	mergeBoolPtr(&effective.Presentation.CompactMode, session.Presentation.CompactMode)
	mergeString(&effective.Presentation.Theme, session.Presentation.Theme)
	return vs
}

// mergeString overwrites dst with src only when src is non-empty (the "inherit" sentinel).
func mergeString(dst *string, src string) {
	if src != "" {
		*dst = src
	}
}

// mergeBoolPtr overwrites dst with src only when src is non-nil.
func mergeBoolPtr(dst **bool, src *bool) {
	if src != nil {
		*dst = src
	}
}

// mergeFloatPtr overwrites dst with src only when src is non-nil.
func mergeFloatPtr(dst **float64, src *float64) {
	if src != nil {
		*dst = src
	}
}

// DefaultGlobal returns a sensible seeded Global profile used when no Global profile
// has been configured yet. Callers (runtime) persist this on first boot.
func DefaultGlobal() AssistantProfile {
	t := true
	f := false
	return AssistantProfile{
		ID:          "global",
		Name:        "Default Assistant",
		Description: "OpenBotStack default global assistant profile",
		Scope:       ScopeGlobal,
		Soul: Soul{
			Identity: Identity{
				Name:        "Assistant",
				Description: "OpenBotStack assistant",
				Persona:     PersonaGeneral,
				Domain:      DomainGeneral,
			},
			Behavior: Behavior{
				Tone:     "concise",
				Language: "zh-CN",
			},
		},
		Reasoning: ReasoningPolicy{Enabled: &t, ShowReasoning: &f},
		Safety: SafetyPolicy{
			HallucinationGuard: &t,
			MedicalMode:        &f,
			ContentFilter:      &t,
		},
		Evidence: EvidencePolicy{Required: &t, CitationRequired: &t},
		Output: OutputPolicy{
			Language:    "zh-CN",
			Markdown:    &t,
			Citations:   &t,
			Temperature: floatPtr(0.3),
		},
		Presentation: PresentationPolicy{
			ReasoningSummary: &f,
			CompactMode:      &f,
			Theme:            "light",
		},
	}
}
