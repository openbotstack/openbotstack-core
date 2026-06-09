package prompts

import (
	_ "embed"
	"text/template"
)

//go:embed plan_prompt.md
var planPromptRaw string

//go:embed replan_prompt.md
var replanPromptRaw string

// PlanTemplate is the parsed plan prompt template.
var PlanTemplate *template.Template

// ReplanTemplate is the parsed replan prompt template.
var ReplanTemplate *template.Template

func init() {
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}

	PlanTemplate = template.Must(template.New("plan").Funcs(funcMap).Parse(planPromptRaw))
	ReplanTemplate = template.Must(template.New("replan").Funcs(funcMap).Parse(replanPromptRaw))
}

// PlanData is the template data for plan prompts.
type PlanData struct {
	Personality   string
	Instructions  string
	MemoryContext []string
	Skills        string
	UserRequest   string
}

// ReplanStepData represents a step in the original plan for replan prompts.
type ReplanStepData struct {
	Type string
	Name string
	Args string
}

// ReplanData is the template data for replan prompts.
type ReplanData struct {
	OriginalSteps   []ReplanStepData
	FailedStepType  string
	FailedStepName  string
	ErrorMessage    string
	Trigger         string
	PreviousResults map[string]string
	Personality     string
	Instructions    string
	MemoryContext   []string
	Skills          string
	UserRequest     string
}
