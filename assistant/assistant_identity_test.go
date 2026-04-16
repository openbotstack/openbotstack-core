package assistant

import (
	"context"
	"os"
	"testing"
)

func TestLoadSoulFromMarkdown(t *testing.T) {
	content := `# Personality
Friendly and helpful coding assistant.

## System Prompt
You are OpenBotStack Assistant.

## Instructions
1. Always be polite.
2. Use markdown for code.
`
	f, err := os.CreateTemp(t.TempDir(), "soul-*.md")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close temp file: %v", err)
	}

	soul, err := LoadSoulFromMarkdown(f.Name())
	if err != nil {
		t.Fatalf("failed to load soul: %v", err)
	}

	if soul.Personality != "Friendly and helpful coding assistant." {
		t.Errorf("expected personality 'Friendly and helpful coding assistant.', got '%s'", soul.Personality)
	}

	if soul.SystemPrompt != "You are OpenBotStack Assistant." {
		t.Errorf("expected system prompt 'You are OpenBotStack Assistant.', got '%s'", soul.SystemPrompt)
	}

	wantInstructions := "1. Always be polite.\n2. Use markdown for code."
	if soul.Instructions != wantInstructions {
		t.Errorf("expected instructions '%s', got '%s'", wantInstructions, soul.Instructions)
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
