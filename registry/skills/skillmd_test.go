package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ==================== ParseSkillMDContent Tests ====================

// --- Normal cases (7) ---

func TestParseSkillMDContent_ValidFrontmatterAndBody(t *testing.T) {
	input := []byte(`---
name: summarize
description: Summarizes text into bullet points
---

You are a summarization assistant.

Text:
{{.Input}}
`)
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if smd.Name != "summarize" {
		t.Errorf("Name = %q, want %q", smd.Name, "summarize")
	}
	if smd.Description != "Summarizes text into bullet points" {
		t.Errorf("Description = %q, want %q", smd.Description, "Summarizes text into bullet points")
	}
	if smd.Body != "You are a summarization assistant.\n\nText:\n{{.Input}}" {
		t.Errorf("Body = %q", smd.Body)
	}
}

func TestParseSkillMDContent_NoFrontmatter(t *testing.T) {
	input := []byte("Just a simple prompt with no frontmatter at all.\n{{.Input}}\n")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if smd.Name != "" {
		t.Errorf("Name should be empty, got %q", smd.Name)
	}
	if smd.Description != "" {
		t.Errorf("Description should be empty, got %q", smd.Description)
	}
	if smd.Body != "Just a simple prompt with no frontmatter at all.\n{{.Input}}" {
		t.Errorf("Body = %q", smd.Body)
	}
}

func TestParseSkillMDContent_FrontmatterOnlyNoBody(t *testing.T) {
	input := []byte("---\nname: test\ndescription: A test skill\n---\n")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if smd.Name != "test" {
		t.Errorf("Name = %q, want %q", smd.Name, "test")
	}
	if smd.Body != "" {
		t.Errorf("Body should be empty, got %q", smd.Body)
	}
}

func TestParseSkillMDContent_ExtraFieldsInFrontmatter(t *testing.T) {
	input := []byte("---\nname: test\ndescription: desc\nversion: 1.0\n---\nBody here")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if smd.Name != "test" {
		t.Errorf("Name = %q, want %q", smd.Name, "test")
	}
	if smd.Body != "Body here" {
		t.Errorf("Body = %q", smd.Body)
	}
}

func TestParseSkillMDContent_NameOnlyNoDescription(t *testing.T) {
	input := []byte("---\nname: minimal\n---\nSome body")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if smd.Name != "minimal" {
		t.Errorf("Name = %q, want %q", smd.Name, "minimal")
	}
	if smd.Description != "" {
		t.Errorf("Description should be empty")
	}
}

func TestParseSkillMDContent_MultilineDescription(t *testing.T) {
	input := []byte("---\nname: test\ndescription: |\n  This is a long\n  multi-line description\n---\nBody")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if smd.Name != "test" {
		t.Errorf("Name = %q", smd.Name)
	}
	if !strings.Contains(smd.Description, "multi-line") {
		t.Errorf("Description should contain 'multi-line', got %q", smd.Description)
	}
}

func TestParseSkillMDContent_BodyWithCodeBlocks(t *testing.T) {
	input := []byte("---\nname: code-skill\ndescription: Uses code blocks\n---\n```python\nprint('hello')\n```\nDone")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(smd.Body, "```python") {
		t.Errorf("Body should contain code block, got %q", smd.Body)
	}
}

// --- Abnormal cases (42+) ---

func TestParseSkillMDContent_Empty(t *testing.T) {
	_, err := ParseSkillMDContent([]byte(""))
	if err == nil {
		t.Error("expected error for empty content")
	}
}

func TestParseSkillMDContent_WhitespaceOnly(t *testing.T) {
	_, err := ParseSkillMDContent([]byte("   \n\t\n  "))
	if err == nil {
		t.Error("expected error for whitespace-only content")
	}
}

func TestParseSkillMDContent_UnclosedFrontmatter(t *testing.T) {
	input := []byte("---\nname: test\nthis never closes")
	_, err := ParseSkillMDContent(input)
	if err == nil {
		t.Error("expected error for unclosed frontmatter")
	}
}

func TestParseSkillMDContent_InvalidYAML(t *testing.T) {
	input := []byte("---\nname: [broken yaml\n---\nBody")
	_, err := ParseSkillMDContent(input)
	if err == nil {
		t.Error("expected error for invalid YAML in frontmatter")
	}
}

func TestParseSkillMDContent_FrontmatterDelimiterInBody(t *testing.T) {
	input := []byte("---\nname: test\n---\nBody with --- dashes\nMore text")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(smd.Body, "--- dashes") {
		t.Errorf("Body should contain dashes, got %q", smd.Body)
	}
}

func TestParseSkillMDContent_DoubleDashOnly(t *testing.T) {
	input := []byte("--\nname: test\n---\nBody")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// "--" is not a frontmatter delimiter, treated as body
	if smd.Name != "" {
		t.Errorf("Name should be empty, got %q", smd.Name)
	}
}

func TestParseSkillMDContent_FourDashes(t *testing.T) {
	input := []byte("----\nname: test\n---\nBody")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// "----" starts with "---" so should be parsed as frontmatter
	if smd.Name != "" {
		t.Errorf("Name should be empty since ---- isn't valid frontmatter start, got %q", smd.Name)
	}
}

func TestParseSkillMDContent_FrontmatterWithTrailingSpaces(t *testing.T) {
	input := []byte("---  \nname: test\n---\nBody")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if smd.Name != "test" {
		t.Errorf("Name = %q, want %q", smd.Name, "test")
	}
}

func TestParseSkillMDContent_EmptyFrontmatter(t *testing.T) {
	input := []byte("---\n---\nBody content")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if smd.Name != "" {
		t.Errorf("Name should be empty")
	}
	if smd.Body != "Body content" {
		t.Errorf("Body = %q", smd.Body)
	}
}

func TestParseSkillMDContent_NullBytes(t *testing.T) {
	input := []byte("---\nname: test\n---\nBody with \x00 null")
	// Null bytes should not cause a panic; either parse or error is acceptable
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Logf("ParseSkillMDContent returned error for null bytes: %v", err)
		return
	}
	if smd == nil {
		t.Fatal("expected non-nil result when no error returned")
	}
	if !strings.Contains(smd.Body, "null") {
		t.Errorf("Body should contain 'null', got %q", smd.Body)
	}
}

func TestParseSkillMDContent_BOMPrefix(t *testing.T) {
	input := append([]byte{0xEF, 0xBB, 0xBF}, []byte("---\nname: test\n---\nBody")...)
	// BOM before --- should prevent frontmatter parsing
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// BOM means no frontmatter detected
	if smd.Name != "" {
		t.Errorf("Name should be empty with BOM prefix, got %q", smd.Name)
	}
}

func TestParseSkillMDContent_CRLFLineEndings(t *testing.T) {
	input := []byte("---\r\nname: test\r\ndescription: desc\r\n---\r\nBody\r\n")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if smd.Name != "test" {
		t.Errorf("Name = %q, want %q", smd.Name, "test")
	}
	if smd.Description != "desc" {
		t.Errorf("Description = %q, want %q", smd.Description, "desc")
	}
	if !strings.Contains(smd.Body, "Body") {
		t.Errorf("Body should contain 'Body', got %q", smd.Body)
	}
}

func TestParseSkillMDContent_TabIndentation(t *testing.T) {
	input := []byte("---\nname:\ttest\ndescription:\tA test\n---\nBody")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if smd.Name != "test" {
		t.Errorf("Name = %q, want %q", smd.Name, "test")
	}
}

func TestParseSkillMDContent_UnicodeName(t *testing.T) {
	input := []byte("---\nname: 护理交接\ndescription: SBAR 护理交接文书\n---\n指令内容")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if smd.Name != "护理交接" {
		t.Errorf("Name = %q, want %q", smd.Name, "护理交接")
	}
}

func TestParseSkillMDContent_EmojiInDescription(t *testing.T) {
	input := []byte("---\nname: emoji\ndescription: Skill with 🎉 emojis 🚀\n---\nBody")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(smd.Description, "🎉") {
		t.Errorf("Description = %q", smd.Description)
	}
}

func TestParseSkillMDContent_VeryLongBody(t *testing.T) {
	longBody := strings.Repeat("This is a line in the body.\n", 10000)
	input := []byte("---\nname: long\ndescription: Long skill\n---\n" + longBody)
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(smd.Body) < 100000 {
		t.Errorf("Body too short: %d bytes", len(smd.Body))
	}
}

func TestParseSkillMDContent_VeryLongDescription(t *testing.T) {
	longDesc := strings.Repeat("This is part of the description. ", 1000)
	input := []byte("---\nname: long-desc\ndescription: " + longDesc + "\n---\nBody")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(smd.Description) < 1000 {
		t.Errorf("Description too short: %d", len(smd.Description))
	}
}

func TestParseSkillMDContent_EmptyName(t *testing.T) {
	input := []byte("---\nname: \"\"\ndescription: desc\n---\nBody")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if smd.Name != "" {
		t.Errorf("Name should be empty string, got %q", smd.Name)
	}
}

func TestParseSkillMDContent_NameWithSpecialChars(t *testing.T) {
	input := []byte("---\nname: my-skill_v2.0\ndescription: desc\n---\nBody")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if smd.Name != "my-skill_v2.0" {
		t.Errorf("Name = %q, want %q", smd.Name, "my-skill_v2.0")
	}
}

func TestParseSkillMDContent_NameWithSpaces(t *testing.T) {
	input := []byte("---\nname: \"My Skill Name\"\ndescription: desc\n---\nBody")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if smd.Name != "My Skill Name" {
		t.Errorf("Name = %q, want %q", smd.Name, "My Skill Name")
	}
}

func TestParseSkillMDContent_YAMLAnchor(t *testing.T) {
	input := []byte("---\nname: &anchor test\ndescription: *anchor\n---\nBody")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if smd.Name != "test" {
		t.Errorf("Name = %q, want %q", smd.Name, "test")
	}
}

func TestParseSkillMDContent_MultipleFrontmatterBlocks(t *testing.T) {
	// Only first --- pair is frontmatter
	input := []byte("---\nname: first\n---\nBody\n---\nname: second\n---\nMore")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if smd.Name != "first" {
		t.Errorf("Name = %q, want %q from first block", smd.Name, "first")
	}
	if !strings.Contains(smd.Body, "---") {
		t.Errorf("Body should contain second --- block")
	}
}

func TestParseSkillMDContent_DashInBodyNotDelimiter(t *testing.T) {
	input := []byte("---\nname: test\n---\nStep 1:\n- Do this\n- Do that\nDone")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(smd.Body, "Do this") {
		t.Errorf("Body should contain list items, got %q", smd.Body)
	}
}

func TestParseSkillMDContent_ThreeDashesInBodyOnOwnLine(t *testing.T) {
	input := []byte("---\nname: test\n---\nBody line 1\n---\nBody line 2")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The second --- in body is not a frontmatter closer (already closed)
	if !strings.Contains(smd.Body, "Body line 1") {
		t.Errorf("Body should contain both lines, got %q", smd.Body)
	}
}

func TestParseSkillMDContent_DescriptionMultilineFolded(t *testing.T) {
	input := []byte("---\nname: test\ndescription: >\n  This is a folded\n  description\n---\nBody")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if smd.Description == "" {
		t.Error("Description should not be empty")
	}
}

func TestParseSkillMDContent_BinaryGarbage(t *testing.T) {
	input := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	// PNG header — should not be valid frontmatter
	// Should be treated as body (no --- prefix)
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if smd == nil {
		t.Error("should return non-nil for non-frontmatter content")
	}
}

func TestParseSkillMDContent_OnlyDashes(t *testing.T) {
	input := []byte("---")
	_, err := ParseSkillMDContent(input)
	if err == nil {
		t.Error("expected error for only opening delimiter")
	}
}

func TestParseSkillMDContent_FrontmatterNoNewline(t *testing.T) {
	input := []byte("---\nname: test---\nBody")
	// No closing --- on its own line
	_, err := ParseSkillMDContent(input)
	if err == nil {
		t.Error("expected error when closing --- not on own line")
	}
}

func TestParseSkillMDContent_NestedYAML(t *testing.T) {
	input := []byte("---\nname: test\ndescription: desc\nmeta:\n  key: value\n---\nBody")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if smd.Name != "test" {
		t.Errorf("Name = %q", smd.Name)
	}
}

func TestParseSkillMDContent_YAMLListValue(t *testing.T) {
	input := []byte("---\nname: test\ndescription: desc\ntags:\n  - tag1\n  - tag2\n---\nBody")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if smd.Name != "test" {
		t.Errorf("Name = %q", smd.Name)
	}
}

func TestParseSkillMDContent_DescriptionWithQuotes(t *testing.T) {
	input := []byte("---\nname: test\ndescription: 'Contains \"quotes\"'\n---\nBody")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(smd.Description, "quotes") {
		t.Errorf("Description = %q", smd.Description)
	}
}

func TestParseSkillMDContent_DescriptionWithColon(t *testing.T) {
	input := []byte("---\nname: test\ndescription: \"A: B\"\n---\nBody")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(smd.Description, "A: B") {
		t.Errorf("Description = %q", smd.Description)
	}
}

func TestParseSkillMDContent_NameWithHash(t *testing.T) {
	input := []byte("---\nname: \"skill #1\"\ndescription: desc\n---\nBody")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if smd.Name != "skill #1" {
		t.Errorf("Name = %q, want %q", smd.Name, "skill #1")
	}
}

func TestParseSkillMDContent_DescriptionAsNumber(t *testing.T) {
	input := []byte("---\nname: test\ndescription: 12345\n---\nBody")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if smd.Description != "12345" {
		t.Errorf("Description = %q, want %q", smd.Description, "12345")
	}
}

func TestParseSkillMDContent_NameAsBoolean(t *testing.T) {
	input := []byte("---\nname: true\ndescription: desc\n---\nBody")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// YAML parses true as boolean, but we store as string
	if smd.Name == "" {
		t.Errorf("Name should not be empty for 'true'")
	}
}

func TestParseSkillMDContent_WhitespaceAroundDelimiters(t *testing.T) {
	input := []byte("\n\n---\nname: test\n---\nBody")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if smd.Name != "test" {
		t.Errorf("Name = %q, want %q", smd.Name, "test")
	}
}

func TestParseSkillMDContent_TemplatePlaceholdersOnly(t *testing.T) {
	input := []byte("---\nname: tpl\ndescription: tpl\n---\n{{.Input}}\n{{.Output}}")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(smd.Body, "{{.Input}}") {
		t.Errorf("Body should contain template placeholders")
	}
}

func TestParseSkillMDContent_SingleDashBody(t *testing.T) {
	input := []byte("---\nname: test\n---\n-")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if smd.Body != "-" {
		t.Errorf("Body = %q, want %q", smd.Body, "-")
	}
}

func TestParseSkillMDContent_FrontmatterWithComments(t *testing.T) {
	input := []byte("---\n# This is a comment\nname: test\n# Another comment\ndescription: desc\n---\nBody")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if smd.Name != "test" {
		t.Errorf("Name = %q, want %q", smd.Name, "test")
	}
}

func TestParseSkillMDContent_XMLStyleFrontmatter(t *testing.T) {
	input := []byte("<!-- not yaml -->\nBody")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if smd.Name != "" {
		t.Errorf("Name should be empty for non-YAML frontmatter")
	}
}

func TestParseSkillMDContent_BodyWithYAMLBlock(t *testing.T) {
	input := []byte("---\nname: test\n---\nSome text\n\n```yaml\nkey: value\n```\nDone")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(smd.Body, "key: value") {
		t.Errorf("Body should contain YAML code block")
	}
}

func TestParseSkillMDContent_FrontmatterTrailingWhitespaceOnDelimiter(t *testing.T) {
	input := []byte("---  \nname: test  \n---  \nBody")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if smd.Name != "test" {
		t.Errorf("Name = %q after trim, want %q", smd.Name, "test")
	}
}

func TestParseSkillMDContent_EmptyLinesBetweenFrontmatterFields(t *testing.T) {
	input := []byte("---\nname: test\n\ndescription: desc\n\n---\nBody")
	smd, err := ParseSkillMDContent(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if smd.Name != "test" {
		t.Errorf("Name = %q", smd.Name)
	}
	if smd.Description != "desc" {
		t.Errorf("Description = %q", smd.Description)
	}
}

// ==================== ParseSkillMD (filesystem) Tests ====================

func TestParseSkillMD_FileExists(t *testing.T) {
	dir := t.TempDir()
	content := "---\nname: fs-test\ndescription: from filesystem\n---\nFile body"
	os.WriteFile(filepath.Join(dir, SkillMDFileName), []byte(content), 0644)

	smd, err := ParseSkillMD(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if smd.Name != "fs-test" {
		t.Errorf("Name = %q", smd.Name)
	}
}

func TestParseSkillMD_FileNotExists(t *testing.T) {
	dir := t.TempDir()
	smd, err := ParseSkillMD(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if smd != nil {
		t.Errorf("expected nil for missing file, got %+v", smd)
	}
}

func TestParseSkillMD_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, SkillMDFileName), []byte(""), 0644)

	_, err := ParseSkillMD(dir)
	if err == nil {
		t.Error("expected error for empty file")
	}
}

// ==================== HasSkillMD Tests ====================

func TestHasSkillMD_True(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, SkillMDFileName), []byte("content"), 0644)
	if !HasSkillMD(dir) {
		t.Error("expected true")
	}
}

func TestHasSkillMD_False(t *testing.T) {
	dir := t.TempDir()
	if HasSkillMD(dir) {
		t.Error("expected false")
	}
}

// ==================== DeriveSkillID Tests ====================

func TestDeriveSkillID_NamespacePath(t *testing.T) {
	id := DeriveSkillID("skills/core/summarize")
	if id != "core/summarize" {
		t.Errorf("got %q, want %q", id, "core/summarize")
	}
}

func TestDeriveSkillID_DeepPath(t *testing.T) {
	id := DeriveSkillID("skills/report/summary-gen")
	if id != "report/summary-gen" {
		t.Errorf("got %q, want %q", id, "report/summary-gen")
	}
}

func TestDeriveSkillID_TopLevel(t *testing.T) {
	id := DeriveSkillID("skills/hello-world")
	if id != "hello-world" {
		t.Errorf("got %q, want %q", id, "hello-world")
	}
}

func TestDeriveSkillID_SkillsParentDir(t *testing.T) {
	id := DeriveSkillID("/some/path/skills/hello-world")
	if id != "hello-world" {
		t.Errorf("got %q, want %q", id, "hello-world")
	}
}

func TestDeriveSkillID_AbsolutePath(t *testing.T) {
	id := DeriveSkillID("/opt/openbotstack/skills/core/summarize")
	if id != "core/summarize" {
		t.Errorf("got %q, want %q", id, "core/summarize")
	}
}

func TestDeriveSkillID_TrailingSlash(t *testing.T) {
	id := DeriveSkillID("skills/core/summarize/")
	if id != "core/summarize" {
		t.Errorf("got %q, want %q", id, "core/summarize")
	}
}

func TestDeriveSkillID_Empty(t *testing.T) {
	id := DeriveSkillID("")
	if id != "" {
		t.Errorf("got %q, want empty", id)
	}
}

func TestDeriveSkillID_SingleDir(t *testing.T) {
	id := DeriveSkillID("summarize")
	if id != "summarize" {
		t.Errorf("got %q, want %q", id, "summarize")
	}
}

