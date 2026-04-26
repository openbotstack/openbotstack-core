package assistant

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/openbotstack/openbotstack-core/control/skills"
)

// ============================================================================
// SessionMemory tests
// ============================================================================

func TestSessionMemory_Get_MissingKey(t *testing.T) {
	mem := NewSessionMemory()
	ctx := context.Background()

	_, err := mem.Get(ctx, "nonexistent")
	if !errors.Is(err, ErrMemoryNotFound) {
		t.Errorf("Get(nonexistent) error = %v, want ErrMemoryNotFound", err)
	}
}

func TestSessionMemory_SetThenGet(t *testing.T) {
	mem := NewSessionMemory()
	ctx := context.Background()

	key := "test-key"
	want := []byte("test-value")

	if err := mem.Set(ctx, key, want); err != nil {
		t.Fatalf("Set(%q, %q) error: %v", key, want, err)
	}

	got, err := mem.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get(%q) error: %v", key, err)
	}
	if string(got) != string(want) {
		t.Errorf("Get(%q) = %q, want %q", key, got, want)
	}
}

func TestSessionMemory_SetOverwrites(t *testing.T) {
	mem := NewSessionMemory()
	ctx := context.Background()

	key := "overwrite-key"
	_ = mem.Set(ctx, key, []byte("first"))
	_ = mem.Set(ctx, key, []byte("second"))

	got, err := mem.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get(%q) error: %v", key, err)
	}
	if string(got) != "second" {
		t.Errorf("Get(%q) = %q, want %q after overwrite", key, got, "second")
	}
}

func TestSessionMemory_Delete(t *testing.T) {
	mem := NewSessionMemory()
	ctx := context.Background()

	key := "delete-me"
	_ = mem.Set(ctx, key, []byte("value"))

	if err := mem.Delete(ctx, key); err != nil {
		t.Fatalf("Delete(%q) error: %v", key, err)
	}

	_, err := mem.Get(ctx, key)
	if !errors.Is(err, ErrMemoryNotFound) {
		t.Errorf("Get(%q) after Delete: error = %v, want ErrMemoryNotFound", key, err)
	}
}

func TestSessionMemory_Delete_NonexistentKey(t *testing.T) {
	mem := NewSessionMemory()
	ctx := context.Background()

	// Deleting a key that never existed should not error.
	err := mem.Delete(ctx, "never-existed")
	if err != nil {
		t.Errorf("Delete(nonexistent) error = %v, want nil", err)
	}
}

func TestSessionMemory_EmptyValue(t *testing.T) {
	mem := NewSessionMemory()
	ctx := context.Background()

	key := "empty-val"
	if err := mem.Set(ctx, key, []byte{}); err != nil {
		t.Fatalf("Set(%q, []) error: %v", key, err)
	}

	got, err := mem.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get(%q) error: %v", key, err)
	}
	if len(got) != 0 {
		t.Errorf("Get(%q) = %v, want empty byte slice", key, got)
	}
}

func TestSessionMemory_NilValue(t *testing.T) {
	mem := NewSessionMemory()
	ctx := context.Background()

	key := "nil-val"
	if err := mem.Set(ctx, key, nil); err != nil {
		t.Fatalf("Set(%q, nil) error: %v", key, err)
	}

	got, err := mem.Get(ctx, key)
	if err != nil {
		t.Fatalf("Get(%q) error: %v", key, err)
	}
	if got != nil {
		t.Errorf("Get(%q) = %v, want nil", key, got)
	}
}

func TestSessionMemory_MultipleKeys(t *testing.T) {
	mem := NewSessionMemory()
	ctx := context.Background()

	entries := map[string]string{
		"key-a": "value-a",
		"key-b": "value-b",
		"key-c": "value-c",
	}
	for k, v := range entries {
		_ = mem.Set(ctx, k, []byte(v))
	}

	for k, want := range entries {
		got, err := mem.Get(ctx, k)
		if err != nil {
			t.Errorf("Get(%q) error: %v", k, err)
			continue
		}
		if string(got) != want {
			t.Errorf("Get(%q) = %q, want %q", k, got, want)
		}
	}
}

func TestSessionMemory_Search_ReturnsNil(t *testing.T) {
	mem := NewSessionMemory()
	ctx := context.Background()

	results, err := mem.Search(ctx, "anything", 10)
	if err != nil {
		t.Errorf("Search() error = %v, want nil", err)
	}
	if results != nil {
		t.Errorf("Search() = %v, want nil (session memory does not support search)", results)
	}
}

func TestSessionMemory_ConcurrentReadWrite(t *testing.T) {
	mem := NewSessionMemory()
	ctx := context.Background()

	const writers = 10
	const readers = 20
	const iterations = 100

	var wg sync.WaitGroup

	// Writers: each goroutine writes its own set of keys.
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				key := strings.Join([]string{"writer", string(rune('A' + id)), string(rune('0' + i%10))}, "-")
				_ = mem.Set(ctx, key, []byte(key))
			}
		}(w)
	}

	// Readers: read keys that writers are writing.
	for r := 0; r < readers; r++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				key := strings.Join([]string{"writer", string(rune('A' + id%writers)), string(rune('0' + i%10))}, "-")
				// Get may or may not find the key depending on scheduling;
				// we just need to verify no data race is triggered.
				_, _ = mem.Get(ctx, key)
			}
		}(r)
	}

	wg.Wait()
}

func TestSessionMemory_ConcurrentMixedOperations(t *testing.T) {
	mem := NewSessionMemory()
	ctx := context.Background()

	const goroutines = 20
	const iterations = 50

	var wg sync.WaitGroup

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := strings.Join([]string{"key", string(rune('0' + id%10))}, "-")

			for i := 0; i < iterations; i++ {
				switch i % 3 {
				case 0:
					_ = mem.Set(ctx, key, []byte{byte(id)})
				case 1:
					_, _ = mem.Get(ctx, key)
				case 2:
					_ = mem.Delete(ctx, key)
				}
			}
		}(g)
	}

	wg.Wait()
}

// ============================================================================
// AssistantRuntime construction tests
// ============================================================================

func TestAssistantRuntime_Construction(t *testing.T) {
	soul := AssistantSoul{
		SystemPrompt: "You are a test assistant.",
		Personality:  "Testing personality.",
		Instructions: "Follow test instructions.",
		AllowedSkills: []string{"skill-1", "skill-2"},
		AllowedTools:  []string{"tool-1"},
	}

	rt := AssistantRuntime{
		AssistantID:     "assistant-123",
		TenantID:        "tenant-456",
		Soul:            soul,
		Memory:          NewSessionMemory(),
		Skills:          []string{"skill-1", "skill-2"},
		Policies:        []string{"policy-a"},
		MemoryScope:     "session",
		ToolPermissions: []string{"read", "write"},
	}

	if rt.AssistantID != "assistant-123" {
		t.Errorf("AssistantID = %q, want %q", rt.AssistantID, "assistant-123")
	}
	if rt.TenantID != "tenant-456" {
		t.Errorf("TenantID = %q, want %q", rt.TenantID, "tenant-456")
	}
	if rt.Soul.SystemPrompt != "You are a test assistant." {
		t.Errorf("Soul.SystemPrompt = %q, want %q", rt.Soul.SystemPrompt, "You are a test assistant.")
	}
	if rt.Memory == nil {
		t.Error("Memory should not be nil")
	}
	if len(rt.Skills) != 2 {
		t.Errorf("len(Skills) = %d, want 2", len(rt.Skills))
	}
	if len(rt.Policies) != 1 {
		t.Errorf("len(Policies) = %d, want 1", len(rt.Policies))
	}
	if rt.MemoryScope != "session" {
		t.Errorf("MemoryScope = %q, want %q", rt.MemoryScope, "session")
	}
	if len(rt.ToolPermissions) != 2 {
		t.Errorf("len(ToolPermissions) = %d, want 2", len(rt.ToolPermissions))
	}
}

func TestAssistantRuntime_EmptyFields(t *testing.T) {
	rt := AssistantRuntime{}

	if rt.AssistantID != "" {
		t.Errorf("AssistantID = %q, want empty", rt.AssistantID)
	}
	if rt.Soul.SystemPrompt != "" {
		t.Errorf("Soul.SystemPrompt = %q, want empty", rt.Soul.SystemPrompt)
	}
	if rt.Skills != nil {
		t.Errorf("Skills = %v, want nil", rt.Skills)
	}
	if rt.Memory != nil {
		t.Error("Memory should be nil when not set")
	}
}

func TestAssistantConfig_Construction(t *testing.T) {
	soul := DefaultSoul()
	cfg := AssistantConfig{
		AssistantID: "cfg-assistant-1",
		Soul:        soul,
		Skills:      []string{"summarize"},
		Policies:    []string{"default"},
		MemoryScope: "tenant",
		ToolAllowedList: []skills.CapabilityType{
			skills.CapTextGeneration,
			skills.CapToolCalling,
		},
	}

	if cfg.AssistantID != "cfg-assistant-1" {
		t.Errorf("AssistantID = %q, want %q", cfg.AssistantID, "cfg-assistant-1")
	}
	if cfg.Soul.SystemPrompt != soul.SystemPrompt {
		t.Errorf("Soul.SystemPrompt mismatch")
	}
	if len(cfg.ToolAllowedList) != 2 {
		t.Errorf("len(ToolAllowedList) = %d, want 2", len(cfg.ToolAllowedList))
	}
	if cfg.ToolAllowedList[0] != skills.CapTextGeneration {
		t.Errorf("ToolAllowedList[0] = %q, want %q", cfg.ToolAllowedList[0], skills.CapTextGeneration)
	}
}

// ============================================================================
// Soul tests
// ============================================================================

func TestDefaultSoul(t *testing.T) {
	soul := DefaultSoul()

	if soul.SystemPrompt == "" {
		t.Error("DefaultSoul().SystemPrompt should not be empty")
	}
	if soul.Personality == "" {
		t.Error("DefaultSoul().Personality should not be empty")
	}
	if soul.Instructions == "" {
		t.Error("DefaultSoul().Instructions should not be empty")
	}
}

func TestLoadSoulFromMarkdown_ValidFile(t *testing.T) {
	content := `# System Prompt
You are a specialized finance assistant.

## Personality
Analytical and precise. Always cite sources.

## Instructions
1. Always verify calculations.
2. Never fabricate data.
`
	path := writeTempMarkdown(t, content)

	soul, err := LoadSoulFromMarkdown(path)
	if err != nil {
		t.Fatalf("LoadSoulFromMarkdown() error: %v", err)
	}

	if soul.SystemPrompt != "You are a specialized finance assistant." {
		t.Errorf("SystemPrompt = %q, want %q", soul.SystemPrompt, "You are a specialized finance assistant.")
	}
	if soul.Personality != "Analytical and precise. Always cite sources." {
		t.Errorf("Personality = %q, want %q", soul.Personality, "Analytical and precise. Always cite sources.")
	}
	if !strings.Contains(soul.Instructions, "Always verify calculations.") {
		t.Errorf("Instructions = %q, should contain %q", soul.Instructions, "Always verify calculations.")
	}
}

func TestLoadSoulFromMarkdown_NonexistentFile(t *testing.T) {
	_, err := LoadSoulFromMarkdown("/nonexistent/path/soul.md")
	if err == nil {
		t.Error("LoadSoulFromMarkdown(nonexistent) should return an error")
	}
	if !os.IsNotExist(err) {
		t.Errorf("error type = %T, want a file-not-exists error", err)
	}
}

func TestLoadSoulFromMarkdown_EmptyFile(t *testing.T) {
	path := writeTempMarkdown(t, "")

	soul, err := LoadSoulFromMarkdown(path)
	if err != nil {
		t.Fatalf("LoadSoulFromMarkdown(empty) error: %v", err)
	}

	// Empty file should fall back to DefaultSoul values.
	defaultSoul := DefaultSoul()
	if soul.SystemPrompt != defaultSoul.SystemPrompt {
		t.Errorf("SystemPrompt = %q, want default %q", soul.SystemPrompt, defaultSoul.SystemPrompt)
	}
}

func TestLoadSoulFromMarkdown_NoRecognizedSections(t *testing.T) {
	content := `# About
This is about section.

## Notes
Some notes here.
`
	path := writeTempMarkdown(t, content)

	soul, err := LoadSoulFromMarkdown(path)
	if err != nil {
		t.Fatalf("LoadSoulFromMarkdown() error: %v", err)
	}

	// None of the section headers match known names (prompt/personality/instruction),
	// so the soul should retain its default values.
	defaultSoul := DefaultSoul()
	if soul.SystemPrompt != defaultSoul.SystemPrompt {
		t.Errorf("SystemPrompt = %q, want default (unrecognized sections ignored)", soul.SystemPrompt)
	}
}

func TestLoadSoulFromMarkdown_PartialSections(t *testing.T) {
	content := `# system prompt
Custom system prompt only.
`
	path := writeTempMarkdown(t, content)

	soul, err := LoadSoulFromMarkdown(path)
	if err != nil {
		t.Fatalf("LoadSoulFromMarkdown() error: %v", err)
	}

	if soul.SystemPrompt != "Custom system prompt only." {
		t.Errorf("SystemPrompt = %q, want %q", soul.SystemPrompt, "Custom system prompt only.")
	}
	// Personality and Instructions should keep defaults.
	defaultSoul := DefaultSoul()
	if soul.Personality != defaultSoul.Personality {
		t.Errorf("Personality = %q, want default %q", soul.Personality, defaultSoul.Personality)
	}
}

func TestLoadSoulFromMarkdown_MultilineContent(t *testing.T) {
	content := `# personality
Line one.
Line two.
Line three.

## instructions
Step 1: Read.
Step 2: Analyze.
Step 3: Respond.
`
	path := writeTempMarkdown(t, content)

	soul, err := LoadSoulFromMarkdown(path)
	if err != nil {
		t.Fatalf("LoadSoulFromMarkdown() error: %v", err)
	}

	if !strings.Contains(soul.Personality, "Line one.") {
		t.Errorf("Personality = %q, should contain 'Line one.'", soul.Personality)
	}
	if !strings.Contains(soul.Personality, "Line three.") {
		t.Errorf("Personality = %q, should contain 'Line three.'", soul.Personality)
	}
	if !strings.Contains(soul.Instructions, "Step 3: Respond.") {
		t.Errorf("Instructions = %q, should contain 'Step 3: Respond.'", soul.Instructions)
	}
}

func TestLoadSoulFromMarkdown_AllowsSliceFields(t *testing.T) {
	soul := DefaultSoul()
	if soul.AllowedSkills != nil {
		t.Errorf("DefaultSoul().AllowedSkills = %v, want nil", soul.AllowedSkills)
	}
	if soul.AllowedTools != nil {
		t.Errorf("DefaultSoul().AllowedTools = %v, want nil", soul.AllowedTools)
	}
}

// ============================================================================
// AssistantProfile tests
// ============================================================================

func TestAssistantProfile_Construction(t *testing.T) {
	p := AssistantProfile{
		ID:          "profile-1",
		Name:        "Test Assistant",
		Description: "A test assistant profile.",
		Version:     "1.0.0",
	}

	if p.ID != "profile-1" {
		t.Errorf("ID = %q, want %q", p.ID, "profile-1")
	}
	if p.Name != "Test Assistant" {
		t.Errorf("Name = %q, want %q", p.Name, "Test Assistant")
	}
	if p.Description != "A test assistant profile." {
		t.Errorf("Description = %q, want %q", p.Description, "A test assistant profile.")
	}
	if p.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", p.Version, "1.0.0")
	}
}

func TestAssistantProfile_Empty(t *testing.T) {
	p := AssistantProfile{}
	if p.ID != "" {
		t.Errorf("ID = %q, want empty", p.ID)
	}
	if p.Name != "" {
		t.Errorf("Name = %q, want empty", p.Name)
	}
}

// ============================================================================
// AssistantRegistry tests
// ============================================================================

func TestAssistantRegistry_RegisterAndGetProfile(t *testing.T) {
	reg := NewAssistantRegistry()
	profile := AssistantProfile{
		ID:          "asst-1",
		Name:        "Helper",
		Description: "Helpful assistant",
		Version:     "2.0.0",
	}
	cfg := AssistantConfig{
		AssistantID: "asst-1",
		Soul:        DefaultSoul(),
		Skills:      []string{"summarize"},
		Policies:    []string{"default"},
		MemoryScope: "session",
	}

	reg.Register(profile, cfg)

	gotProfile, err := reg.GetProfile("asst-1")
	if err != nil {
		t.Fatalf("GetProfile() error: %v", err)
	}
	if gotProfile.ID != "asst-1" {
		t.Errorf("GetProfile().ID = %q, want %q", gotProfile.ID, "asst-1")
	}
	if gotProfile.Name != "Helper" {
		t.Errorf("GetProfile().Name = %q, want %q", gotProfile.Name, "Helper")
	}
}

func TestAssistantRegistry_GetProfile_NotFound(t *testing.T) {
	reg := NewAssistantRegistry()

	_, err := reg.GetProfile("nonexistent")
	if !errors.Is(err, ErrAssistantNotFound) {
		t.Errorf("GetProfile(nonexistent) error = %v, want ErrAssistantNotFound", err)
	}
}

func TestAssistantRegistry_GetConfig(t *testing.T) {
	reg := NewAssistantRegistry()
	profile := AssistantProfile{ID: "asst-cfg", Name: "CfgTest"}
	cfg := AssistantConfig{
		AssistantID: "asst-cfg",
		Soul:        DefaultSoul(),
		Skills:      []string{"sentiment", "summarize"},
		Policies:    []string{"strict"},
		MemoryScope: "tenant",
	}

	reg.Register(profile, cfg)

	gotCfg, err := reg.GetConfig("asst-cfg")
	if err != nil {
		t.Fatalf("GetConfig() error: %v", err)
	}
	if gotCfg.AssistantID != "asst-cfg" {
		t.Errorf("GetConfig().AssistantID = %q, want %q", gotCfg.AssistantID, "asst-cfg")
	}
	if len(gotCfg.Skills) != 2 {
		t.Errorf("len(GetConfig().Skills) = %d, want 2", len(gotCfg.Skills))
	}
}

func TestAssistantRegistry_GetConfig_NotFound(t *testing.T) {
	reg := NewAssistantRegistry()

	_, err := reg.GetConfig("nonexistent")
	if !errors.Is(err, ErrAssistantNotFound) {
		t.Errorf("GetConfig(nonexistent) error = %v, want ErrAssistantNotFound", err)
	}
}

func TestAssistantRegistry_RegisterOverwrites(t *testing.T) {
	reg := NewAssistantRegistry()

	profileV1 := AssistantProfile{ID: "asst-ow", Name: "V1", Version: "1.0"}
	cfgV1 := AssistantConfig{AssistantID: "asst-ow", MemoryScope: "session"}
	reg.Register(profileV1, cfgV1)

	profileV2 := AssistantProfile{ID: "asst-ow", Name: "V2", Version: "2.0"}
	cfgV2 := AssistantConfig{AssistantID: "asst-ow", MemoryScope: "tenant"}
	reg.Register(profileV2, cfgV2)

	gotProfile, _ := reg.GetProfile("asst-ow")
	if gotProfile.Version != "2.0" {
		t.Errorf("Profile.Version after overwrite = %q, want %q", gotProfile.Version, "2.0")
	}

	gotCfg, _ := reg.GetConfig("asst-ow")
	if gotCfg.MemoryScope != "tenant" {
		t.Errorf("Config.MemoryScope after overwrite = %q, want %q", gotCfg.MemoryScope, "tenant")
	}
}

func TestAssistantRegistry_MultipleAssistants(t *testing.T) {
	reg := NewAssistantRegistry()

	for i := 0; i < 5; i++ {
		id := strings.Join([]string{"asst", string(rune('0' + i))}, "-")
		profile := AssistantProfile{ID: id, Name: id}
		cfg := AssistantConfig{AssistantID: id}
		reg.Register(profile, cfg)
	}

	for i := 0; i < 5; i++ {
		id := strings.Join([]string{"asst", string(rune('0' + i))}, "-")
		p, err := reg.GetProfile(id)
		if err != nil {
			t.Errorf("GetProfile(%q) error: %v", id, err)
			continue
		}
		if p.ID != id {
			t.Errorf("GetProfile(%q).ID = %q, want %q", id, p.ID, id)
		}
	}
}

func TestAssistantRegistry_ConcurrentAccess(t *testing.T) {
	reg := NewAssistantRegistry()
	ctx := context.Background()

	const goroutines = 20
	const iterations = 50

	var wg sync.WaitGroup

	// Concurrent registrations.
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := strings.Join([]string{"asst", string(rune('A' + id%26))}, "-")
			for i := 0; i < iterations; i++ {
				profile := AssistantProfile{ID: key, Name: key}
				cfg := AssistantConfig{
					AssistantID: key,
					Soul:        DefaultSoul(),
				}
				reg.Register(profile, cfg)
			}
		}(g)
	}

	// Concurrent reads.
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := strings.Join([]string{"asst", string(rune('A' + id%26))}, "-")
			for i := 0; i < iterations; i++ {
				_, _ = reg.GetProfile(key)
				_, _ = reg.GetConfig(key)
			}
		}(g)
	}

	// Concurrent SessionMemory operations to ensure no deadlock with the
	// memory stored in config.
	mem := NewSessionMemory()
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := strings.Join([]string{"mem", string(rune('0' + id%10))}, "-")
			for i := 0; i < iterations; i++ {
				_ = mem.Set(ctx, key, []byte{byte(id)})
				_, _ = mem.Get(ctx, key)
			}
		}(g)
	}

	wg.Wait()
}

// ============================================================================
// AssistantMemory interface conformance test
// ============================================================================

func TestSessionMemory_ImplementsAssistantMemory(t *testing.T) {
	// Compile-time interface check.
	var _ AssistantMemory = (*SessionMemory)(nil)
}

// ============================================================================
// Table-driven tests for Soul markdown parsing
// ============================================================================

func TestLoadSoulFromMarkdown_Table(t *testing.T) {
	tests := []struct {
		name              string
		content           string
		wantSystemPrompt  string
		wantPersonality   string
		wantInstructions  string
		wantErr           bool
	}{
		{
			name:             "all_sections",
			content:          "# System Prompt\nPrompt content.\n\n## Personality\nPersonality content.\n\n## Instructions\nInstruction content.",
			wantSystemPrompt: "Prompt content.",
			wantPersonality:  "Personality content.",
			wantInstructions: "Instruction content.",
		},
		{
			name:             "h2_sections",
			content:          "## System Prompt\nH2 prompt.\n\n## Personality\nH2 personality.",
			wantSystemPrompt: "H2 prompt.",
			wantPersonality:  "H2 personality.",
		},
		{
			name:             "case_insensitive_headers",
			content:          "# SYSTEM PROMPT\nUppercase prompt.\n\n## Instructions\nMixed case instructions.",
			wantSystemPrompt: "Uppercase prompt.",
			wantInstructions: "Mixed case instructions.",
		},
		{
			name:    "empty_file",
			content: "",
			// Defaults from DefaultSoul.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTempMarkdown(t, tt.content)
			soul, err := LoadSoulFromMarkdown(path)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantSystemPrompt != "" && soul.SystemPrompt != tt.wantSystemPrompt {
				t.Errorf("SystemPrompt = %q, want %q", soul.SystemPrompt, tt.wantSystemPrompt)
			}
			if tt.wantPersonality != "" && soul.Personality != tt.wantPersonality {
				t.Errorf("Personality = %q, want %q", soul.Personality, tt.wantPersonality)
			}
			if tt.wantInstructions != "" && soul.Instructions != tt.wantInstructions {
				t.Errorf("Instructions = %q, want %q", soul.Instructions, tt.wantInstructions)
			}
		})
	}
}

// ============================================================================
// Table-driven tests for SessionMemory
// ============================================================================

func TestSessionMemory_Table(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name    string
		key     string
		value   []byte
		wantErr error
	}{
		{name: "simple_string", key: "simple", value: []byte("hello")},
		{name: "empty_key", key: "", value: []byte("value-for-empty-key")},
		{name: "binary_data", key: "binary", value: []byte{0x00, 0x01, 0xFF}},
		{name: "large_value", key: "large", value: make([]byte, 4096)},
		{name: "unicode_key", key: "unicode-key-\xc3\xa9", value: []byte("unicode value")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mem := NewSessionMemory()

			err := mem.Set(ctx, tt.key, tt.value)
			if err != nil {
				t.Fatalf("Set() error: %v", err)
			}

			got, err := mem.Get(ctx, tt.key)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Get() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Get() error: %v", err)
			}

			if string(got) != string(tt.value) {
				t.Errorf("Get() = %v, want %v", got, tt.value)
			}
		})
	}
}

// ============================================================================
// Helpers
// ============================================================================

// writeTempMarkdown creates a temporary markdown file with the given content.
func writeTempMarkdown(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "soul.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write temp markdown: %v", err)
	}
	return path
}
