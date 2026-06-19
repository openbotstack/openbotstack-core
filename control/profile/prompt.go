package profile

import (
	"fmt"
	"strings"
)

// RenderPrompt synthesizes a minimal system prompt from a structured Soul, replacing
// the traditional free-form "You are a..." paragraph. Every token in the output is
// derived from typed fields (Identity, Behavior) — there is no prose template.
//
// When all fields are zero the result is an empty string, which the planner treats as
// "no system prompt" (the model runs without a persona override).
//
// This is the Phase 2 pivot point: the Planner writes RenderPrompt(soul) into the
// system message instead of injection-point copying a long AssistantSoul.SystemPrompt.
func RenderPrompt(s Soul) string {
	var b strings.Builder

	if s.Identity.Persona != "" {
		b.WriteString("Persona: ")
		b.WriteString(s.Identity.Persona)
		if s.Identity.Name != "" {
			b.WriteString(" (")
			b.WriteString(s.Identity.Name)
			b.WriteByte(')')
		}
	}
	if s.Identity.Domain != "" {
		if b.Len() > 0 {
			b.WriteString(". ")
		}
		fmt.Fprintf(&b, "Domain: %s", s.Identity.Domain)
	}
	if s.Behavior.Tone != "" {
		if b.Len() > 0 {
			b.WriteString(". ")
		}
		fmt.Fprintf(&b, "Tone: %s", s.Behavior.Tone)
	}
	if s.Behavior.Citations != nil && *s.Behavior.Citations {
		if b.Len() > 0 {
			b.WriteString(". ")
		}
		b.WriteString("Cite evidence when reporting factual claims.")
	}
	return strings.TrimSpace(b.String())
}

// RenderPromptFull returns a more descriptive prompt including description and domain
// context, suitable when the caller wants the model to understand its role beyond a
// single-line tag.
func RenderPromptFull(s Soul) string {
	var b strings.Builder
	if s.Identity.Name != "" {
		fmt.Fprintf(&b, "You are acting as: %s", s.Identity.Name)
		if s.Identity.Persona != "" && !strings.EqualFold(s.Identity.Persona, "general") {
			fmt.Fprintf(&b, " (%s persona)", s.Identity.Persona)
		}
	}
	if s.Identity.Description != "" {
		if b.Len() > 0 {
			b.WriteString(". ")
		}
		b.WriteString(s.Identity.Description)
	}
	if s.Identity.Domain != "" && !strings.EqualFold(s.Identity.Domain, "general") {
		if b.Len() > 0 {
			b.WriteString(" ")
		}
		fmt.Fprintf(&b, "Operating domain: %s.", s.Identity.Domain)
	}
	tag := RenderPrompt(s)
	if tag != "" {
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(tag)
	}
	return strings.TrimSpace(b.String())
}
