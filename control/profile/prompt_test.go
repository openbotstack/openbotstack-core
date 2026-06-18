package profile

import (
	"strings"
	"testing"
)

func TestRenderPrompt_EmptySoul_ReturnsEmpty(t *testing.T) {
	s := Soul{}
	if got := RenderPrompt(s); got != "" {
		t.Errorf("empty Soul should render empty prompt, got %q", got)
	}
}

func TestRenderPrompt_DefaultGlobal(t *testing.T) {
	s := DefaultGlobal().Soul
	got := RenderPrompt(s)
	// DefaultGlobal has persona=general, domain=general, tone=concise, language=zh-CN.
	// Citations is nil in DefaultGlobal (not set), so "Cite evidence" should NOT appear.
	if got == "" {
		t.Fatal("expected non-empty prompt for default global Soul")
	}
	for _, want := range []string{"Persona:", "general", "Tone:", "concise", "Language:", "zh-CN"} {
		if !strings.Contains(got, want) {
			t.Errorf("render missing %q in %q", want, got)
		}
	}
}

func TestRenderPrompt_ICUProfile(t *testing.T) {
	tr := true
	s := Soul{
		Identity: Identity{Name: "ICU Bot", Persona: PersonaICU, Domain: DomainHealthcare},
		Behavior: Behavior{Tone: "detailed", Language: "en-US", Citations: &tr},
	}
	got := RenderPrompt(s)
	if !strings.Contains(got, PersonaICU) || !strings.Contains(got, "ICU Bot") ||
		!strings.Contains(got, DomainHealthcare) || !strings.Contains(got, "detailed") ||
		!strings.Contains(got, "en-US") {
		t.Errorf("ICU Soul render incomplete: %q", got)
	}
}

func TestRenderPrompt_NoFreeFormYouAre(t *testing.T) {
	s := DefaultGlobal().Soul
	got := RenderPrompt(s)
	// Must not contain free-form "You are" prefixed prose.
	if strings.Contains(got, "You are") || strings.Contains(got, "you are") {
		t.Errorf("RenderPrompt must not emit free-form 'You are...' prose: %q", got)
	}
}

func TestRenderPromptFull_UsesNameAndDescription(t *testing.T) {
	s := Soul{
		Identity: Identity{Name: "Radiology Reader", Description: "Analyzes medical images for radiology reports",
			Persona: PersonaRadiology, Domain: DomainHealthcare},
		Behavior: Behavior{Tone: "detailed", Language: "en-US"},
	}
	got := RenderPromptFull(s)
	if !strings.Contains(got, "Radiology Reader") {
		t.Errorf("Full render missing name: %q", got)
	}
	if !strings.Contains(got, "radiology persona") {
		t.Errorf("Full render missing persona: %q", got)
	}
	if !strings.Contains(got, "Operating domain") {
		t.Errorf("Full render missing domain: %q", got)
	}
}
