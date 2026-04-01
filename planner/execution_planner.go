package planner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openbotstack/openbotstack-core/ai/providers"
	"github.com/openbotstack/openbotstack-core/control/skills"
	"github.com/openbotstack/openbotstack-core/execution"
)

var (
	// ErrPlanningFailed is returned when the planner fails to produce a plan.
	ErrPlanningFailed = fmt.Errorf("planner: planning failed")

	// ErrNoSkillsAvailable is returned when no skills are registered.
	ErrNoSkillsAvailable = fmt.Errorf("planner: no skills available")
)

// ExecutionPlanner uses an LLM to generate bounded execution plans.
type ExecutionPlanner interface {
	// Plan analyzes user intent and produces a validated execution plan.
	Plan(ctx context.Context, pCtx *PlannerContext) (*execution.ExecutionPlan, error)
}

// SkillDescriptor describes a skill for LLM context building.
type SkillDescriptor struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	InputSchema *skills.JSONSchema `json:"input_schema,omitempty"`
}

// LLMPlanner implements ExecutionPlanner using an LLM provider to generate JSON plans.
type LLMPlanner struct {
	router    providers.ModelRouter
	validator *Validator
}

// NewLLMPlanner creates a new planner with the given model router and optional limits.
func NewLLMPlanner(router providers.ModelRouter, limits *ExecutionLimits) *LLMPlanner {
	return &LLMPlanner{
		router:    router,
		validator: NewValidator(limits),
	}
}

// Plan uses the assembled context to generate a validated execution plan.
func (p *LLMPlanner) Plan(ctx context.Context, pCtx *PlannerContext) (*execution.ExecutionPlan, error) {
	if len(pCtx.Skills) == 0 {
		return nil, ErrNoSkillsAvailable
	}

	prompt := p.buildPrompt(pCtx)

	mReq := skills.GenerateRequest{
		Messages: []skills.Message{
			{Role: "system", Content: pCtx.Soul.SystemPrompt},
			{Role: "user", Content: prompt},
		},
	}

	provider, err := p.router.Route(
		[]skills.CapabilityType{skills.CapTextGeneration},
		skills.ModelConstraints{},
	)
	if err != nil {
		return nil, fmt.Errorf("%w: routing failed: %v", ErrPlanningFailed, err)
	}

	planCtx, cancel := context.WithTimeout(ctx, p.validator.limits.MaxExecutionTime)
	defer cancel()

	response, err := provider.Generate(planCtx, mReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPlanningFailed, err)
	}

	plan, err := p.parseResponse(response.Content)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to parse LLM response: %v", ErrPlanningFailed, err)
	}

	if plan.AssistantID == "" {
		plan.AssistantID = pCtx.AssistantID
	}

	if err := p.validator.Validate(plan); err != nil {
		return nil, fmt.Errorf("%w: validation failed: %v", ErrPlanningFailed, err)
	}

	return plan, nil
}

// buildPrompt constructs the LLM prompt for skill selection.
func (p *LLMPlanner) buildPrompt(pCtx *PlannerContext) string {
	var sb strings.Builder

	sb.WriteString("You are an execution planner. Create a deterministic execution plan to handle the user's request.\n")
	
	if pCtx.Soul.Personality != "" {
		sb.WriteString(fmt.Sprintf("\nPersonality: %s\n", pCtx.Soul.Personality))
	}
	
	if pCtx.Soul.Instructions != "" {
		sb.WriteString(fmt.Sprintf("\nSpecific Instructions:\n%s\n", pCtx.Soul.Instructions))
	}

	if len(pCtx.MemoryContext) > 0 {
		sb.WriteString("\nRelevant Memory Context:\n")
		for _, mem := range pCtx.MemoryContext {
			sb.WriteString(fmt.Sprintf("- %s\n", string(mem.Content)))
		}
	}

	sb.WriteString("\nAvailable skills/tools:\n")
	for _, skill := range pCtx.Skills {
		schemaJSON := "{}"
		if skill.InputSchema != nil {
			bytes, _ := json.Marshal(skill.InputSchema)
			schemaJSON = string(bytes)
		}
		sb.WriteString(fmt.Sprintf("- %s (%s): %s\n  Input schema: %s\n", skill.ID, skill.Name, skill.Description, schemaJSON))
	}

	sb.WriteString("\nUser request: ")
	sb.WriteString(pCtx.UserRequest)
	sb.WriteString("\n\n")

	sb.WriteString(`Respond with a JSON object containing the execution plan. Do not include any other text or reasoning.
Format:
{
  "assistant_id": "...",
  "steps": [
    {
      "type": "skill", // or "tool"
      "name": "namespace/skill_name",
      "arguments": {"arg": "value"}
    }
  ]
}`)

	return sb.String()
}

// parseResponse extracts an ExecutionPlan from the LLM response.
func (p *LLMPlanner) parseResponse(response string) (*execution.ExecutionPlan, error) {
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var plan execution.ExecutionPlan
	if err := json.Unmarshal([]byte(response), &plan); err != nil {
		return nil, fmt.Errorf("invalid json: %w (response: %s)", err, response)
	}

	return &plan, nil
}
