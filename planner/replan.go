package planner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/openbotstack/openbotstack-core/execution"
	"github.com/openbotstack/openbotstack-core/planner/prompts"
)

// Replan generates a revised execution plan after a step failure.
func (p *LLMPlanner) Replan(ctx context.Context, rCtx *ReplanContext) (*execution.ExecutionPlan, error) {
	if rCtx.OriginalPlan == nil {
		return nil, fmt.Errorf("replan: original plan is nil")
	}
	if rCtx.PlannerContext == nil {
		return nil, fmt.Errorf("replan: planner context is nil")
	}
	if len(rCtx.PlannerContext.Skills) == 0 {
		return nil, ErrNoSkillsAvailable
	}

	prompt := p.buildReplanPrompt(rCtx)

	plan, err := p.llmPlanRound(ctx, rCtx.PlannerContext, prompt, planRoundConfig{
		events: planProgressEvents{
			generating: "replanning_generating",
			token:      "replanning_token",
			complete:   "replanning_complete",
		},
		wrapRoutingErr: func(err error) error {
			return fmt.Errorf("replan: routing failed: %w", err)
		},
		wrapLLMErr: func(err error) error {
			return fmt.Errorf("replan: %w", err)
		},
		wrapParseErr: func(err error) error {
			return fmt.Errorf("replan: failed to parse LLM response: %w", err)
		},
	})
	if err != nil {
		return nil, err
	}

	// Set lineage fields before validation.
	plan.ParentID = rCtx.OriginalPlan.ID

	// Empty plan is the cooperative stop signal.
	if len(plan.Steps) == 0 {
		return plan, nil
	}

	if err := p.validator.Validate(plan); err != nil {
		return nil, fmt.Errorf("replan: validation failed: %w", err)
	}

	// Freeze the plan and auto-generate IDs.
	if err := plan.Validate(); err != nil {
		return nil, fmt.Errorf("replan: plan freeze failed: %w", err)
	}

	return plan, nil
}

// buildReplanPrompt constructs a specialized prompt for replanning using a template.
func (p *LLMPlanner) buildReplanPrompt(rCtx *ReplanContext) string {
	// Build original step descriptions.
	var steps []prompts.ReplanStepData
	for _, step := range rCtx.OriginalPlan.Steps {
		sd := prompts.ReplanStepData{
			Type: string(step.Type),
			Name: step.Name,
		}
		if step.Arguments != nil {
			argsJSON, _ := json.Marshal(step.Arguments)
			sd.Args = string(argsJSON)
		}
		steps = append(steps, sd)
	}

	// Build previous results as string map.
	prevResults := make(map[string]string)
	for name, result := range rCtx.PreviousResults {
		switch v := result.(type) {
		case string:
			prevResults[name] = v
		default:
			b, _ := json.Marshal(v)
			prevResults[name] = string(b)
		}
	}

	var specs []ToolSpec
	for _, skill := range rCtx.PlannerContext.Skills {
		specs = append(specs, SchemaToToolSpec(skill))
	}

	var memContext []string
	for _, mem := range rCtx.PlannerContext.MemoryContext {
		memContext = append(memContext, string(mem.Content))
	}

	turnResults := make([]prompts.TurnResultData, len(rCtx.PlannerContext.TurnResults))
	for i, tr := range rCtx.PlannerContext.TurnResults {
		turnResults[i] = prompts.TurnResultData{
			StepName: tr.StepName,
			StepType: tr.StepType,
			Success:  tr.Success,
			Summary:  tr.Summary,
			Error:    tr.Error,
		}
	}

	data := prompts.ReplanData{
		OriginalSteps:   steps,
		FailedStepType:  string(rCtx.FailedStep.Type),
		FailedStepName:  rCtx.FailedStep.Name,
		ErrorMessage:    rCtx.ErrorMessage,
		Trigger:         string(rCtx.Trigger),
		PreviousResults: prevResults,
		Personality:     rCtx.PlannerContext.Soul.Personality,
		Instructions:    rCtx.PlannerContext.Soul.Instructions,
		MemoryContext:   memContext,
		TurnResults:     turnResults,
		Skills:          FormatToolSpecs(specs),
		UserRequest:     escapeXML(rCtx.PlannerContext.UserRequest),
	}

	var buf bytes.Buffer
	if err := prompts.ReplanTemplate.Execute(&buf, data); err != nil {
		return "Generate a revised plan for: " + escapeXML(rCtx.PlannerContext.UserRequest)
	}
	return buf.String()
}
