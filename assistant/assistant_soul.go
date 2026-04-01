package assistant

import (
	"os"
	"strings"
)

// AssistantSoul defines the behavioral parameters of an assistant.
// It acts as the "inner logic" and "personality" that guides the LLM.
type AssistantSoul struct {
	SystemPrompt    string   `json:"system_prompt"`
	Personality     string   `json:"personality"`
	Instructions    string   `json:"instructions"`
	AllowedSkills   []string `json:"allowed_skills"`
	AllowedTools    []string `json:"allowed_tools"`
}

// DefaultSoul returns a generic baseline soul.
func DefaultSoul() AssistantSoul {
	return AssistantSoul{
		SystemPrompt: "You are a helpful and efficient AI assistant.",
		Personality:  "Professional, concise, and helpful.",
		Instructions: "Follow the user's instructions strictly. Use tools when necessary.",
	}
}

// LoadSoulFromMarkdown attempts to populate an AssistantSoul from a markdown file.
// It expects specific headers or sections to identify personality and instructions.
func LoadSoulFromMarkdown(path string) (AssistantSoul, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return AssistantSoul{}, err
	}

	lines := strings.Split(string(content), "\n")
	soul := DefaultSoul()
	
	currentSection := ""
	var sectionContent strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") || strings.HasPrefix(trimmed, "## ") {
			// Save previous section
			if currentSection != "" {
				saveSection(&soul, currentSection, sectionContent.String())
				sectionContent.Reset()
			}
			currentSection = strings.ToLower(strings.TrimLeft(trimmed, "# "))
			continue
		}
		
		if currentSection != "" {
			sectionContent.WriteString(line + "\n")
		}
	}
	
	// Save last section
	if currentSection != "" {
		saveSection(&soul, currentSection, sectionContent.String())
	}

	return soul, nil
}

func saveSection(soul *AssistantSoul, name string, content string) {
	content = strings.TrimSpace(content)
	switch {
	case strings.Contains(name, "prompt"):
		soul.SystemPrompt = content
	case strings.Contains(name, "personality"):
		soul.Personality = content
	case strings.Contains(name, "instruction"):
		soul.Instructions = content
	}
}
