package assistant

import (
	"context"
	"testing"
)

func TestLoadSoulFromMarkdown(t *testing.T) {
	path := "/tmp/soul.md"
	
	soul, err := LoadSoulFromMarkdown(path)
	if err != nil {
		t.Fatalf("failed to load soul: %v", err)
	}

	if soul.Personality != "Friendly and helpful coding assistant." {
		t.Errorf("expected personality 'Friendly and helpful coding assistant.', got '%s'", soul.Personality)
	}

	if soul.SystemPrompt != "You are OpenBotStack Assistant." {
		t.Errorf("expected system prompt 'You are OpenBotStack Assistant.', got '%s'", soul.SystemPrompt)
	}

	if soul.Instructions != "1. Always be polite.\n2. Use markdown for code." {
		t.Errorf("expected instructions with two lines, got '%s'", soul.Instructions)
	}
}

func TestSessionMemory(t *testing.T) {
	mem := NewSessionMemory()
	ctx := context.Background()

	err := mem.Set(ctx, "key1", []byte("val1"))
	if err != nil {
		t.Fatalf("failed to set memory: %v", err)
	}

	val, err := mem.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("failed to get memory: %v", err)
	}

	if string(val) != "val1" {
		t.Errorf("expected val1, got %s", string(val))
	}

	_, err = mem.Get(ctx, "nonexistent")
	if err != ErrMemoryNotFound {
		t.Errorf("expected ErrMemoryNotFound, got %v", err)
	}
}
