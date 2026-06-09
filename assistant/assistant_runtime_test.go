package assistant

import (
	"testing"

	"github.com/openbotstack/openbotstack-core/planner"
)

func TestAssistantRuntime_EffectiveSystemPrompt(t *testing.T) {
	tests := []struct {
		name string
		rt   *AssistantRuntime
		want string
	}{
		{
			name: "custom system prompt",
			rt: &AssistantRuntime{
				AssistantID: "medical",
				Soul:        planner.AssistantSoul{SystemPrompt: "You are a medical assistant."},
			},
			want: "You are a medical assistant.",
		},
		{
			name: "default when empty",
			rt: &AssistantRuntime{
				AssistantID: "default",
			},
			want: "You are a helpful AI assistant.",
		},
		{
			name: "default when nil soul",
			rt: &AssistantRuntime{
				AssistantID: "default",
			},
			want: "You are a helpful AI assistant.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.rt.EffectiveSystemPrompt()
			if got != tt.want {
				t.Errorf("EffectiveSystemPrompt() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAssistantRuntime_EffectivePersonality(t *testing.T) {
	rt := &AssistantRuntime{
		Soul: planner.AssistantSoul{Personality: "professional and concise"},
	}
	if got := rt.EffectivePersonality(); got != "professional and concise" {
		t.Errorf("EffectivePersonality() = %q, want %q", got, "professional and concise")
	}

	rtEmpty := &AssistantRuntime{}
	if got := rtEmpty.EffectivePersonality(); got != "" {
		t.Errorf("EffectivePersonality() = %q, want empty", got)
	}
}

func TestAssistantRuntime_EffectiveInstructions(t *testing.T) {
	rt := &AssistantRuntime{
		Soul: planner.AssistantSoul{Instructions: "Always verify facts."},
	}
	if got := rt.EffectiveInstructions(); got != "Always verify facts." {
		t.Errorf("EffectiveInstructions() = %q, want %q", got, "Always verify facts.")
	}

	rtEmpty := &AssistantRuntime{}
	if got := rtEmpty.EffectiveInstructions(); got != "" {
		t.Errorf("EffectiveInstructions() = %q, want empty", got)
	}
}

func TestAssistantRuntime_AllowedSkillsAndTools(t *testing.T) {
	rt := &AssistantRuntime{
		Soul: planner.AssistantSoul{
			AllowedSkills: []string{"summarize", "classify"},
			AllowedTools:  []string{"builtin.web_fetch"},
		},
	}

	skills := rt.AllowedSkills()
	if len(skills) != 2 || skills[0] != "summarize" || skills[1] != "classify" {
		t.Errorf("AllowedSkills() = %v, want [summarize classify]", skills)
	}

	tools := rt.AllowedTools()
	if len(tools) != 1 || tools[0] != "builtin.web_fetch" {
		t.Errorf("AllowedTools() = %v, want [builtin.web_fetch]", tools)
	}

	// Nil soul returns nil.
	rtEmpty := &AssistantRuntime{}
	if rtEmpty.AllowedSkills() != nil {
		t.Errorf("AllowedSkills() = %v, want nil", rtEmpty.AllowedSkills())
	}
	if rtEmpty.AllowedTools() != nil {
		t.Errorf("AllowedTools() = %v, want nil", rtEmpty.AllowedTools())
	}
}
