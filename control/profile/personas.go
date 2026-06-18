package profile

import "sort"

// Controlled vocabulary tokens for Identity.Persona and Identity.Domain (ADR-042 §1).
//
// These are intentionally string constants rather than a closed enum so the registry can
// be extended (Admin may register additional personas in future phases) without breaking
// serialized data. A blank Persona is permitted and is equivalent to PersonaGeneral.
//
// Free-form "You are a..." prompts are NOT accepted as Persona values — the whole point
// of ADR-042 is to retire long system prompts.

const (
	PersonaGeneral   = "general"
	PersonaICU       = "icu"
	PersonaNursing   = "nursing"
	PersonaRadiology = "radiology"
	PersonaEmergency = "emergency"
)

const (
	DomainGeneral   = "general"
	DomainHealthcare = "healthcare"
	DomainFinance   = "finance"
	DomainLegal     = "legal"
)

// defaultPersonas is the built-in persona registry. It may be appended to at runtime
// via RegisterPersona; the built-in set is always present and cannot be removed.
var defaultPersonas = map[string]bool{
	PersonaGeneral:   true,
	PersonaICU:       true,
	PersonaNursing:   true,
	PersonaRadiology: true,
	PersonaEmergency: true,
}

// defaultDomains is the built-in domain registry.
var defaultDomains = map[string]bool{
	DomainGeneral:    true,
	DomainHealthcare: true,
	DomainFinance:    true,
	DomainLegal:      true,
}

// RegisterPersona adds a persona token to the runtime registry. It allows controlled
// extension without code changes in future phases. Re-registering an existing token is
// a no-op. Returns the token for chaining.
func RegisterPersona(token string) string {
	if token != "" {
		defaultPersonas[token] = true
	}
	return token
}

// RegisterDomain adds a domain token to the runtime registry.
func RegisterDomain(token string) string {
	if token != "" {
		defaultDomains[token] = true
	}
	return token
}

// ValidPersona reports whether a persona token is recognized. A blank token is valid
// (treated as "general"/unspecified).
func ValidPersona(token string) bool {
	if token == "" {
		return true
	}
	return defaultPersonas[token]
}

// ValidDomain reports whether a domain token is recognized. A blank token is valid.
func ValidDomain(token string) bool {
	if token == "" {
		return true
	}
	return defaultDomains[token]
}

// ListPersonas returns the sorted set of registered persona tokens.
func ListPersonas() []string {
	return sortedKeys(defaultPersonas)
}

// ListDomains returns the sorted set of registered domain tokens.
func ListDomains() []string {
	return sortedKeys(defaultDomains)
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// ValidatePersonaFields validates the Identity persona/domain tokens of a profile,
// returning violations for any unrecognized token. Used by the write path alongside
// ValidateScope. A blank token is never a violation.
func ValidatePersonaFields(p AssistantProfile) []Violation {
	var vs []Violation
	if !ValidPersona(p.Soul.Identity.Persona) {
		vs = append(vs, Violation{Scope: p.Scope, Field: "soul.identity.persona",
			Reason: "unknown persona token: " + p.Soul.Identity.Persona})
	}
	if !ValidDomain(p.Soul.Identity.Domain) {
		vs = append(vs, Violation{Scope: p.Scope, Field: "soul.identity.domain",
			Reason: "unknown domain token: " + p.Soul.Identity.Domain})
	}
	return vs
}
