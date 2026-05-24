package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// SkillMDFileName is the canonical filename for skill definitions.
const SkillMDFileName = "SKILL.md"

// SkillMD holds the parsed content of a SKILL.md file.
// The file uses YAML frontmatter (between --- delimiters) for metadata
// and a markdown body for LLM instructions.
type SkillMD struct {
	Name        string // from frontmatter
	Description string // from frontmatter
	Body        string // markdown content after frontmatter
}

// frontmatter holds the YAML fields parsed from SKILL.md frontmatter.
type frontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// ParseSkillMD reads a SKILL.md file from the given directory and parses
// its YAML frontmatter and markdown body. Returns (nil, nil) if the file
// does not exist.
func ParseSkillMD(dir string) (*SkillMD, error) {
	data, err := os.ReadFile(filepath.Join(dir, SkillMDFileName))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return ParseSkillMDContent(data)
}

// ParseSkillMDContent parses SKILL.md content from bytes.
// Expects optional YAML frontmatter between --- delimiters followed by markdown body.
func ParseSkillMDContent(data []byte) (*SkillMD, error) {
	content := strings.TrimSpace(string(data))
	// Normalize CRLF to LF for consistent processing
	content = strings.ReplaceAll(content, "\r\n", "\n")
	if content == "" {
		return nil, fmt.Errorf("SKILL.md is empty")
	}

	// Check for frontmatter delimiter: must be exactly "---", not "----" or more.
	// After "---", the next character must NOT be '-' (can be whitespace, newline, or EOF).
	if !strings.HasPrefix(content, "---") || (len(content) > 3 && content[3] == '-') {
		// No frontmatter — entire content is the body
		return &SkillMD{Body: content}, nil
	}

	// Find closing ---
	rest := content[3:] // skip opening ---
	closingIdx := findFrontmatterEnd(rest)
	if closingIdx < 0 {
		return nil, fmt.Errorf("SKILL.md: unclosed frontmatter delimiter")
	}

	fmBytes := []byte(strings.TrimSpace(rest[:closingIdx]))
	body := strings.TrimSpace(rest[closingIdx+3:])

	var fm frontmatter
	if len(fmBytes) > 0 {
		if err := yaml.Unmarshal(fmBytes, &fm); err != nil {
			return nil, fmt.Errorf("SKILL.md: invalid frontmatter YAML: %w", err)
		}
	}

	return &SkillMD{
		Name:        strings.TrimSpace(fm.Name),
		Description: strings.TrimSpace(fm.Description),
		Body:        body,
	}, nil
}

// HasSkillMD checks whether a SKILL.md file exists in the given directory.
func HasSkillMD(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, SkillMDFileName))
	return err == nil
}

// DeriveSkillID derives a skill ID from the directory path.
// Given "skills/core/summarize" it returns "core/summarize".
// Given "skills/report/summary-gen" it returns "report/summary-gen".
// It strips the base "skills" or "skills/" prefix if present.
func DeriveSkillID(skillDir string) string {
	// Clean trailing slashes
	skillDir = strings.TrimRight(skillDir, string(filepath.Separator))
	if skillDir == "" {
		return ""
	}

	name := filepath.Base(skillDir)
	parent := filepath.Base(filepath.Dir(skillDir))

	// If the parent is "skills", this is a top-level skill (no namespace)
	if parent == "skills" || parent == "." || parent == "" {
		return name
	}
	// Otherwise namespace/name
	return parent + "/" + name
}

// findFrontmatterEnd finds the closing --- delimiter in frontmatter content.
// It handles the case where --- appears at the start of a line.
func findFrontmatterEnd(content string) int {
	lines := strings.SplitN(content, "\n", -1)
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" && i > 0 {
			// Calculate byte offset
			offset := 0
			for j := 0; j < i; j++ {
				offset += len(lines[j]) + 1 // +1 for newline
			}
			return offset
		}
		// Also handle --- on first line (empty frontmatter)
		if i == 0 && trimmed == "" {
			continue
		}
		if i == 0 && trimmed == "---" {
			return 0
		}
	}
	return -1
}
