package skills

import (
	"testing"
	"time"

	"github.com/openbotstack/openbotstack-core/ai/types"
)

type promptTestSkill struct {
	id, name, desc string
	prompt         string
}

func (s *promptTestSkill) ID() string                          { return s.id }
func (s *promptTestSkill) Name() string                        { return s.name }
func (s *promptTestSkill) Description() string                 { return s.desc }
func (s *promptTestSkill) InputSchema() *types.JSONSchema     { return nil }
func (s *promptTestSkill) OutputSchema() *types.JSONSchema    { return nil }
func (s *promptTestSkill) RequiredPermissions() []string       { return nil }
func (s *promptTestSkill) Timeout() time.Duration              { return 30 * time.Second }
func (s *promptTestSkill) Validate() error                     { return nil }
func (s *promptTestSkill) Prompt() string                      { return s.prompt }

type noPromptTestSkill struct {
	id, name, desc string
}

func (s *noPromptTestSkill) ID() string                          { return s.id }
func (s *noPromptTestSkill) Name() string                        { return s.name }
func (s *noPromptTestSkill) Description() string                 { return s.desc }
func (s *noPromptTestSkill) InputSchema() *types.JSONSchema     { return nil }
func (s *noPromptTestSkill) OutputSchema() *types.JSONSchema    { return nil }
func (s *noPromptTestSkill) RequiredPermissions() []string       { return nil }
func (s *noPromptTestSkill) Timeout() time.Duration              { return 30 * time.Second }
func (s *noPromptTestSkill) Validate() error                     { return nil }

func TestGetPrompt_WithProvider(t *testing.T) {
	s := &promptTestSkill{id: "test/with", prompt: "You are a helper."}
	got := GetPrompt(s)
	if got != "You are a helper." {
		t.Errorf("expected prompt text, got %q", got)
	}
}

func TestGetPrompt_WithEmptyPrompt(t *testing.T) {
	s := &promptTestSkill{id: "test/empty", prompt: ""}
	got := GetPrompt(s)
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestGetPrompt_WithoutProvider(t *testing.T) {
	s := &noPromptTestSkill{id: "test/no-prompt"}
	got := GetPrompt(s)
	if got != "" {
		t.Errorf("expected empty string for non-provider, got %q", got)
	}
}
