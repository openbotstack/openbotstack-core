package planner

import "testing"

func TestAssistantSoul_Construction(t *testing.T) {
	soul := AssistantSoul{
		SystemPrompt:  "You are a test assistant.",
		Personality:   "Testing personality.",
		Instructions:  "Follow test instructions.",
		AllowedSkills: []string{"skill-1", "skill-2"},
		AllowedTools:  []string{"tool-1"},
	}

	if soul.SystemPrompt != "You are a test assistant." {
		t.Errorf("SystemPrompt = %q, want %q", soul.SystemPrompt, "You are a test assistant.")
	}
	if len(soul.AllowedSkills) != 2 {
		t.Errorf("len(AllowedSkills) = %d, want 2", len(soul.AllowedSkills))
	}
}

func TestAssistantSoul_Empty(t *testing.T) {
	soul := AssistantSoul{}
	if soul.SystemPrompt != "" {
		t.Errorf("SystemPrompt = %q, want empty", soul.SystemPrompt)
	}
}

func TestSearchResult(t *testing.T) {
	sr := SearchResult{
		Content: []byte("test content"),
		Score:   0.95,
	}
	if string(sr.Content) != "test content" {
		t.Errorf("Content = %q, want %q", string(sr.Content), "test content")
	}
	if sr.Score != 0.95 {
		t.Errorf("Score = %f, want 0.95", sr.Score)
	}
}
