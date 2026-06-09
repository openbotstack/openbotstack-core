package prompts

import (
	"bytes"
	"strings"
	"testing"
)

func TestPlanTemplateRenders(t *testing.T) {
	data := PlanData{
		Personality:   "professional",
		Instructions:  "Always be concise.",
		MemoryContext: []string{"User prefers short answers"},
		Skills:        "1. summarize - Summarizes text",
		UserRequest:   "Summarize this document",
	}

	var buf bytes.Buffer
	if err := PlanTemplate.Execute(&buf, data); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	result := buf.String()

	// Verify key sections appear.
	if !strings.Contains(result, "execution planner") {
		t.Error("missing execution planner header")
	}
	if !strings.Contains(result, "professional") {
		t.Error("missing personality")
	}
	if !strings.Contains(result, "Always be concise.") {
		t.Error("missing instructions")
	}
	if !strings.Contains(result, "User prefers short answers") {
		t.Error("missing memory context")
	}
	if !strings.Contains(result, "summarize") {
		t.Error("missing skills")
	}
	if !strings.Contains(result, "<user_request>") {
		t.Error("missing user_request tags")
	}
	if !strings.Contains(result, "Summarize this document") {
		t.Error("missing user request content")
	}
	if !strings.Contains(result, "/no_think") {
		t.Error("missing /no_think directive")
	}
	if !strings.Contains(result, "vision_analyze") {
		t.Error("missing vision_analyze instruction")
	}
}

func TestPlanTemplateMinimal(t *testing.T) {
	data := PlanData{
		Skills:      "1. respond - Direct response",
		UserRequest: "Hello",
	}

	var buf bytes.Buffer
	if err := PlanTemplate.Execute(&buf, data); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	result := buf.String()
	// Should not have personality/instructions/memory sections.
	if strings.Contains(result, "Personality:") {
		t.Error("should not contain personality section when empty")
	}
	if strings.Contains(result, "Specific Instructions:") {
		t.Error("should not contain instructions section when empty")
	}
	if strings.Contains(result, "Relevant Memory Context:") {
		t.Error("should not contain memory section when empty")
	}
}

func TestReplanTemplateRenders(t *testing.T) {
	data := ReplanData{
		OriginalSteps: []ReplanStepData{
			{Type: "tool", Name: "builtin.now", Args: "{}"},
			{Type: "skill", Name: "summarize", Args: ""},
		},
		FailedStepType:  "skill",
		FailedStepName:  "summarize",
		ErrorMessage:    "timeout exceeded",
		Trigger:         "error",
		PreviousResults: map[string]string{"builtin.now": `{"time":"2026-01-01"}`},
		Skills:          "1. summarize - Summarizes text",
		UserRequest:     "Summarize this document",
	}

	var buf bytes.Buffer
	if err := ReplanTemplate.Execute(&buf, data); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	result := buf.String()

	if !strings.Contains(result, "previous execution plan failed") {
		t.Error("missing replan header")
	}
	if !strings.Contains(result, "builtin.now") {
		t.Error("missing original step")
	}
	if !strings.Contains(result, "summarize") {
		t.Error("missing failed step")
	}
	if !strings.Contains(result, "timeout exceeded") {
		t.Error("missing error message")
	}
	if !strings.Contains(result, "REMAINING work") {
		t.Error("missing IMPORTANT instruction")
	}
	if !strings.Contains(result, `<user_request>`) {
		t.Error("missing user_request tags")
	}
}

func TestReplanStepNumbering(t *testing.T) {
	data := ReplanData{
		OriginalSteps: []ReplanStepData{
			{Type: "tool", Name: "step1"},
			{Type: "tool", Name: "step2"},
			{Type: "tool", Name: "step3"},
		},
		Skills:      "1. test",
		UserRequest: "test",
	}

	var buf bytes.Buffer
	if err := ReplanTemplate.Execute(&buf, data); err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	result := buf.String()
	// Verify 1-based numbering.
	if !strings.Contains(result, "1. [tool] step1") {
		t.Error("missing step 1")
	}
	if !strings.Contains(result, "2. [tool] step2") {
		t.Error("missing step 2")
	}
	if !strings.Contains(result, "3. [tool] step3") {
		t.Error("missing step 3")
	}
}
