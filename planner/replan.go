package planner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openbotstack/openbotstack-core/ai/types"
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

	msgs := []types.Message{
		{Role: "system", Contents: []types.ContentBlock{types.NewTextBlock(rCtx.PlannerContext.Soul.SystemPrompt)}},
	}
	msgs = append(msgs, filterSystemMessages(rCtx.PlannerContext.ConversationHistory)...)
	msgs = append(msgs, types.Message{Role: "user", Contents: []types.ContentBlock{types.NewTextBlock(prompt)}})

	mReq := types.GenerateRequest{
		Messages:  msgs,
		MaxTokens: 8192,
	}

	provider, err := p.router.Route(
		[]types.CapabilityType{types.CapTextGeneration},
		types.ModelConstraints{},
	)
	if err != nil {
		return nil, fmt.Errorf("replan: routing failed: %w", err)
	}

	planCtx, cancel := context.WithTimeout(ctx, p.validator.limits.MaxExecutionTime)
	defer cancel()

	var responseContent string

	// Try streaming first for progress feedback.
	type streamProvider interface {
		GenerateStream(context.Context, types.GenerateRequest) (<-chan types.StreamChunk, error)
	}
	if sp, ok := provider.(streamProvider); ok && rCtx.PlannerContext.ProgressFn != nil {
		rCtx.PlannerContext.ProgressFn("replanning_generating", "")
		ch, err := sp.GenerateStream(planCtx, mReq)
		if err != nil {
			return nil, fmt.Errorf("replan: %w", err)
		}
		var buf strings.Builder
		for chunk := range ch {
			if chunk.Error != nil {
				return nil, fmt.Errorf("replan: %w", chunk.Error)
			}
			if chunk.Content != "" {
				buf.WriteString(chunk.Content)
				rCtx.PlannerContext.ProgressFn("replanning_token", chunk.Content)
			}
		}
		responseContent = buf.String()
		rCtx.PlannerContext.ProgressFn("replanning_complete", "")
	} else {
		response, err := provider.Generate(planCtx, mReq)
		if err != nil {
			return nil, fmt.Errorf("replan: %w", err)
		}
		responseContent = response.Content
	}

	plan, err := p.parseResponse(responseContent)
	if err != nil {
		return nil, fmt.Errorf("replan: failed to parse LLM response: %w", err)
	}

	// Set lineage fields before validation.
	plan.ParentID = rCtx.OriginalPlan.ID

	if plan.AssistantID == "" {
		plan.AssistantID = rCtx.PlannerContext.AssistantID
	}

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
		Skills:          FormatToolSpecs(specs),
		UserRequest:     escapeXML(rCtx.PlannerContext.UserRequest),
	}

	var buf bytes.Buffer
	if err := prompts.ReplanTemplate.Execute(&buf, data); err != nil {
		return "Generate a revised plan for: " + escapeXML(rCtx.PlannerContext.UserRequest)
	}
	return buf.String()
}
