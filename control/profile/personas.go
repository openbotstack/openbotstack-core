package profile

import (
	"errors"
	"sort"
	"sync"
)

// Controlled vocabulary tokens for Identity.Persona and Identity.Domain (ADR-042 §1).
//
// The vocabulary is a single mutable registry seeded with platform defaults. Tokens
// can be registered at runtime (e.g. an admin entering an out-of-list value) and any
// token can be deleted EXCEPT "general", which is the protected base. Deletion safety
// against dangling profile references is enforced at the runtime layer (the store
// rejects deleting a token still referenced by a stored profile). A blank Persona/
// Domain is always permitted and is equivalent to "general".

var (
	// ErrProtectedVocabulary is returned when attempting to delete "general".
	ErrProtectedVocabulary = errors.New("profile: 'general' is protected and cannot be deleted")
	// ErrUnknownVocabulary is returned when deleting a token that is not registered.
	ErrUnknownVocabulary = errors.New("profile: unknown vocabulary token")
)

const (
	PersonaGeneral   = "general"
	PersonaICU       = "icu"
	PersonaNursing   = "nursing"
	PersonaRadiology = "radiology"
	PersonaEmergency = "emergency"
)

const (
	DomainGeneral    = "general"
	DomainHealthcare = "healthcare"
	DomainFinance    = "finance"
	DomainLegal      = "legal"
)

// VocabEntry is a persona or domain token. Deletable is false only for "general".
type VocabEntry struct {
	Token     string `json:"token"`
	Deletable bool   `json:"deletable"`
}

var (
	vocabMu sync.RWMutex
	// Single mutable registry per kind, seeded with defaults. "general" is protected.
	personaRegistry = map[string]bool{
		PersonaGeneral: true, PersonaICU: true, PersonaNursing: true,
		PersonaRadiology: true, PersonaEmergency: true,
	}
	domainRegistry = map[string]bool{
		DomainGeneral: true, DomainHealthcare: true, DomainFinance: true, DomainLegal: true,
	}
)

// RegisterPersona adds a custom persona token. Empty tokens are no-ops. Returns the
// token for chaining.
func RegisterPersona(token string) string {
	if token == "" {
		return token
	}
	vocabMu.Lock()
	personaRegistry[token] = true
	vocabMu.Unlock()
	return token
}

// RegisterDomain adds a custom domain token.
func RegisterDomain(token string) string {
	if token == "" {
		return token
	}
	vocabMu.Lock()
	domainRegistry[token] = true
	vocabMu.Unlock()
	return token
}

// DeletePersona removes a persona token. "general" is protected. Unknown tokens
// return ErrUnknownVocabulary. Dangling-reference checks are the caller's (runtime)
// responsibility.
func DeletePersona(token string) error {
	if token == PersonaGeneral {
		return ErrProtectedVocabulary
	}
	vocabMu.Lock()
	defer vocabMu.Unlock()
	if !personaRegistry[token] {
		return ErrUnknownVocabulary
	}
	delete(personaRegistry, token)
	return nil
}

// DeleteDomain removes a domain token. "general" is protected.
func DeleteDomain(token string) error {
	if token == DomainGeneral {
		return ErrProtectedVocabulary
	}
	vocabMu.Lock()
	defer vocabMu.Unlock()
	if !domainRegistry[token] {
		return ErrUnknownVocabulary
	}
	delete(domainRegistry, token)
	return nil
}

// ValidPersona reports whether a persona token is registered (or blank).
func ValidPersona(token string) bool {
	if token == "" {
		return true
	}
	vocabMu.RLock()
	defer vocabMu.RUnlock()
	return personaRegistry[token]
}

// ValidDomain reports whether a domain token is registered (or blank).
func ValidDomain(token string) bool {
	if token == "" {
		return true
	}
	vocabMu.RLock()
	defer vocabMu.RUnlock()
	return domainRegistry[token]
}

// ListPersonas returns all registered persona tokens, marked Deletable (false for
// "general" only).
func ListPersonas() []VocabEntry {
	vocabMu.RLock()
	defer vocabMu.RUnlock()
	return vocabEntries(personaRegistry)
}

// ListDomains returns all registered domain tokens, marked Deletable.
func ListDomains() []VocabEntry {
	vocabMu.RLock()
	defer vocabMu.RUnlock()
	return vocabEntries(domainRegistry)
}

func vocabEntries(reg map[string]bool) []VocabEntry {
	out := make([]VocabEntry, 0, len(reg))
	for k := range reg {
		out = append(out, VocabEntry{Token: k, Deletable: k != PersonaGeneral})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Token < out[j].Token })
	return out
}

// ValidatePersonaFields validates the Identity persona/domain tokens of a profile,
// returning violations for any unrecognized token. A blank token is never a violation.
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
