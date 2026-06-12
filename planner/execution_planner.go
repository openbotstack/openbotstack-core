// Package planner provides the execution planning subsystem.
//
// This is the canonical planner package, supporting multi-step execution plans
// with validation, bounded limits (max steps, tool calls, timeout), persona
// injection, and memory context.
package planner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/openbotstack/openbotstack-core/ai/providers"
	"github.com/openbotstack/openbotstack-core/ai/types"
	"github.com/openbotstack/openbotstack-core/execution"
	"github.com/openbotstack/openbotstack-core/planner/prompts"
	"github.com/openbotstack/openbotstack-core/planning"
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

// ProgressFn is a type alias for planning.ProgressFn.
// Kept here for backward compatibility.
type ProgressFn = planning.ProgressFn

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
// If the provider supports streaming, it uses streaming to allow progress feedback
// during the LLM planning call. Otherwise falls back to synchronous Generate.
func (p *LLMPlanner) Plan(ctx context.Context, pCtx *PlannerContext) (*execution.ExecutionPlan, error) {
	if len(pCtx.Skills) == 0 {
		return nil, ErrNoSkillsAvailable
	}

	prompt := p.buildPrompt(pCtx)

	plan, err := p.llmPlanRound(ctx, pCtx, prompt, planRoundConfig{
		events: planProgressEvents{
			generating: "planning_generating",
			token:      "planning_token",
			complete:   "planning_complete",
		},
		wrapRoutingErr: func(err error) error {
			return fmt.Errorf("%w: routing failed: %v", ErrPlanningFailed, err)
		},
		wrapLLMErr: func(err error) error {
			return fmt.Errorf("%w: %v", ErrPlanningFailed, err)
		},
		wrapParseErr: func(err error) error {
			return fmt.Errorf("%w: failed to parse LLM response: %v", ErrPlanningFailed, err)
		},
	})
	if err != nil {
		return nil, err
	}

	// Empty plan is the cooperative stop signal — skip validation and return as-is.
	if len(plan.Steps) == 0 {
		return plan, nil
	}

	if err := p.validator.Validate(plan); err != nil {
		return nil, fmt.Errorf("%w: validation failed: %v", ErrPlanningFailed, err)
	}

	return plan, nil
}

// planProgressEvents holds the progress event names emitted during a planning round.
// Plan and Replan use different event names; both share the llmPlanRound orchestration.
type planProgressEvents struct {
	generating string // emitted before streaming starts
	token      string // emitted per streamed token
	complete   string // emitted when streaming finishes
}

// planRoundConfig carries the caller-specific behavior that differs between Plan
// and Replan: the progress event names and the error-wrapping functions.
//
// The error-wrapping closures let each caller preserve its exact original error
// wording (Plan uses ErrPlanningFailed-based wrapping; Replan uses "replan:"-prefixed
// wrapping) while sharing all LLM orchestration logic.
type planRoundConfig struct {
	events         planProgressEvents
	wrapRoutingErr func(error) error // wraps routing failure
	wrapLLMErr     func(error) error // wraps streaming/generate failure
	wrapParseErr   func(error) error // wraps response parse failure
}

// llmPlanRound is the shared LLM orchestration common to Plan and Replan.
// It builds messages from the planner context + prompt, routes to a provider,
// streams (with progress feedback) or falls back to synchronous Generate,
// parses the response into a plan, and backfills AssistantID.
//
// The caller is responsible for prompt construction, lineage (e.g. ParentID),
// and any post-parse validation/freezing that differs between Plan and Replan.
func (p *LLMPlanner) llmPlanRound(ctx context.Context, pCtx *PlannerContext, prompt string, cfg planRoundConfig) (*execution.ExecutionPlan, error) {
	msgs := []types.Message{
		{Role: "system", Contents: []types.ContentBlock{types.NewTextBlock(pCtx.Soul.SystemPrompt)}},
	}
	msgs = append(msgs, filterSystemMessages(pCtx.ConversationHistory)...)
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
		return nil, cfg.wrapRoutingErr(err)
	}

	planCtx, cancel := context.WithTimeout(ctx, p.validator.limits.MaxExecutionTime)
	defer cancel()

	var responseContent string

	// Try streaming first for progress feedback during planning.
	if sp, ok := provider.(providers.StreamingModelProvider); ok && pCtx.ProgressFn != nil {
		pCtx.ProgressFn(cfg.events.generating, "")
		ch, err := sp.GenerateStream(planCtx, mReq)
		if err != nil {
			return nil, cfg.wrapLLMErr(err)
		}
		var buf strings.Builder
		for chunk := range ch {
			if chunk.Error != nil {
				return nil, cfg.wrapLLMErr(chunk.Error)
			}
			if chunk.Content != "" {
				buf.WriteString(chunk.Content)
				pCtx.ProgressFn(cfg.events.token, chunk.Content)
			}
		}
		responseContent = buf.String()
		if pCtx.ProgressFn != nil {
			pCtx.ProgressFn(cfg.events.complete, "")
		}
	} else {
		response, err := provider.Generate(planCtx, mReq)
		if err != nil {
			return nil, cfg.wrapLLMErr(err)
		}
		responseContent = response.Content
	}

	plan, err := p.parseResponse(responseContent)
	if err != nil {
		return nil, cfg.wrapParseErr(err)
	}

	if plan.AssistantID == "" {
		plan.AssistantID = pCtx.AssistantID
	}

	return plan, nil
}

// buildPrompt constructs the LLM prompt for skill selection using a template.
func (p *LLMPlanner) buildPrompt(pCtx *PlannerContext) string {
	var specs []ToolSpec
	for _, skill := range pCtx.Skills {
		specs = append(specs, SchemaToToolSpec(skill))
	}

	var memContext []string
	for _, mem := range pCtx.MemoryContext {
		memContext = append(memContext, string(mem.Content))
	}

	// Format structured turn results for the template.
	turnResults := make([]prompts.TurnResultData, len(pCtx.TurnResults))
	for i, tr := range pCtx.TurnResults {
		turnResults[i] = prompts.TurnResultData{
			StepName: tr.StepName,
			StepType: tr.StepType,
			Success:  tr.Success,
			Summary:  tr.Summary,
			Error:    tr.Error,
		}
	}

	data := prompts.PlanData{
		Personality:   pCtx.Soul.Personality,
		Instructions:  pCtx.Soul.Instructions,
		MemoryContext: memContext,
		TurnResults:   turnResults,
		Skills:        FormatToolSpecs(specs),
		UserRequest:   escapeXML(pCtx.UserRequest),
	}

	var buf bytes.Buffer
	if err := prompts.PlanTemplate.Execute(&buf, data); err != nil {
		return "You are an execution planner. Create a plan for: " + escapeXML(pCtx.UserRequest)
	}
	return buf.String()
}

// escapeXML escapes XML special characters to prevent prompt injection.
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	return strings.ReplaceAll(s, ">", "&gt;")
}

// filterSystemMessages strips system-role messages to prevent prompt collision.
func filterSystemMessages(msgs []types.Message) []types.Message {
	filtered := make([]types.Message, 0, len(msgs))
	for _, m := range msgs {
		if m.Role != "system" {
			filtered = append(filtered, m)
		}
	}
	return filtered
}

// parseResponse extracts an ExecutionPlan from the LLM response.
func (p *LLMPlanner) parseResponse(response string) (*execution.ExecutionPlan, error) {
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var plan execution.ExecutionPlan
	if err := json.Unmarshal([]byte(response), &plan); err == nil {
		return &plan, nil
	}

	// Fallback: extract JSON object from within text (handles thinking models)
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")
	if start >= 0 && end > start {
		extracted := response[start : end+1]
		if err := json.Unmarshal([]byte(extracted), &plan); err == nil {
			return &plan, nil
		}
	}

	return nil, fmt.Errorf("invalid json: could not extract plan from response (length=%d)", len(response))
}
