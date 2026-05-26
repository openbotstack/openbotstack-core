// Package planner provides the execution planning subsystem.
//
// This is the canonical planner package, supporting multi-step execution plans
// with validation, bounded limits (max steps, tool calls, timeout), persona
// injection, and memory context.
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
// Alias to skills.SkillDescriptor — the canonical definition lives in the
// control/skills package to avoid duplication across planner and agent packages.
type SkillDescriptor = skills.SkillDescriptor

// ProgressFn is the callback signature for planner progress events.
type ProgressFn func(eventType, content string)

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
	if len(pCtx.Skills) == 0 && len(pCtx.Capabilities) == 0 {
		return nil, ErrNoSkillsAvailable
	}

	prompt := p.buildPrompt(pCtx)

	mReq := skills.GenerateRequest{
		Messages: []skills.Message{
			{Role: "system", Content: pCtx.Soul.SystemPrompt},
			{Role: "user", Content: prompt},
		},
		MaxTokens: 8192,
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

	var responseContent string

	// Try streaming first for progress feedback during planning.
	if sp, ok := provider.(providers.StreamingModelProvider); ok && pCtx.ProgressFn != nil {
		pCtx.ProgressFn("planning_generating", "")
		ch, err := sp.GenerateStream(planCtx, mReq)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrPlanningFailed, err)
		}
		var buf strings.Builder
		for chunk := range ch {
			if chunk.Error != nil {
				return nil, fmt.Errorf("%w: %v", ErrPlanningFailed, chunk.Error)
			}
			if chunk.Content != "" {
				buf.WriteString(chunk.Content)
				// Forward each token as a planning_token event so any SSE client
				// receives real-time feedback during the planning phase.
				pCtx.ProgressFn("planning_token", chunk.Content)
			}
		}
		responseContent = buf.String()
		if pCtx.ProgressFn != nil {
			pCtx.ProgressFn("planning_complete", "")
		}
	} else {
		response, err := provider.Generate(planCtx, mReq)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrPlanningFailed, err)
		}
		responseContent = response.Content
	}

	plan, err := p.parseResponse(responseContent)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to parse LLM response: %v", ErrPlanningFailed, err)
	}

	if plan.AssistantID == "" {
		plan.AssistantID = pCtx.AssistantID
	}

	// Empty plan is the cooperative stop signal — skip validation and return as-is.
	// The inner loop checks len(plan.Steps) == 0 to decide plannerStopped.
	if len(plan.Steps) == 0 {
		return plan, nil
	}

	if err := p.validator.Validate(plan); err != nil {
		return nil, fmt.Errorf("%w: validation failed: %v", ErrPlanningFailed, err)
	}

	return plan, nil
}

// buildPrompt constructs the LLM prompt for skill selection.
func (p *LLMPlanner) buildPrompt(pCtx *PlannerContext) string {
	var sb strings.Builder

	sb.WriteString("You are an execution planner. Create a deterministic execution plan to handle the user's request.\n/no_think\n")
	
	if pCtx.Soul.Personality != "" {
		fmt.Fprintf(&sb, "\nPersonality: %s\n", pCtx.Soul.Personality)
	}
	
	if pCtx.Soul.Instructions != "" {
		fmt.Fprintf(&sb, "\nSpecific Instructions:\n%s\n", pCtx.Soul.Instructions)
	}

	if len(pCtx.MemoryContext) > 0 {
		sb.WriteString("\nRelevant Memory Context:\n")
		for _, mem := range pCtx.MemoryContext {
			fmt.Fprintf(&sb, "- %s\n", string(mem.Content))
		}
	}

	sb.WriteString("\nAvailable skills/tools:\n")
	var specs []ToolSpec
	if len(pCtx.Capabilities) > 0 {
		for _, cap := range pCtx.Capabilities {
			specs = append(specs, CapabilityToToolSpec(cap))
		}
	} else {
		for _, skill := range pCtx.Skills {
			specs = append(specs, SchemaToToolSpec(skill))
		}
	}
	sb.WriteString(FormatToolSpecs(specs))

	// Structural boundary: wrap user input in XML tags to prevent
	// prompt injection. Escape XML special characters within user content.
	userInput := pCtx.UserRequest
	userInput = strings.ReplaceAll(userInput, "&", "&amp;")
	userInput = strings.ReplaceAll(userInput, "<", "&lt;")
	userInput = strings.ReplaceAll(userInput, ">", "&gt;")
	sb.WriteString("\n<user_request>\n")
	sb.WriteString(userInput)
	sb.WriteString("\n</user_request>\n\n")

	sb.WriteString(`Respond with a JSON object containing the execution plan. Do not include any other text or reasoning.
	Format:
	{
	  "assistant_id": "...",
	  "steps": [
	    {
	      "type": "tool",
	      "name": "mcp.server.tool_name",
	      "arguments": {"param": "value"}
	    },
	    {
	      "type": "skill",
	      "name": "skill_name",
	      "arguments": {"param": "{{mcp.server.tool_name.result}}"}
	    }
	  ]
	}

	IMPORTANT rules:
	- Use "type": "tool" for tools (IDs starting with "mcp."). Use "type": "skill" for skills.
	- When the user mentions a patient or medical data, ALWAYS first call the relevant mcp.* tools to fetch real data, then pass results to a skill.
	- NEVER skip tool calls and go directly to a skill if relevant mcp.* tools exist for the required data.
		- ALWAYS end the plan with a skill step when tool calls are present — skills format data for the user. Never end a plan with only tool calls.
	- Chain tool calls before skills: fetch data with tools, then pass results to skills via {{step_name}}.
	- Reference outputs from earlier steps using {{step_name}} in argument values.
	- If a step returns a JSON object, use dot notation: {{step_name.field}}.
	- Example plan for patient handover:
	  {"type":"tool","name":"mcp.his.query_patient","arguments":{"patient_id":"P001"}}
	  {"type":"tool","name":"mcp.vitals.get_vitals","arguments":{"patient_id":"P001"}}
	  {"type":"skill","name":"sbar-handover","arguments":{"patient_data":"{{mcp.his.query_patient}}","vitals":"{{mcp.vitals.get_vitals}}"}}
	- Example plan for first day note:
		  {"type":"tool","name":"mcp.his.get_patient_demographics","arguments":{"patient_id":"P001"}}
		  {"type":"tool","name":"mcp.his.get_diagnosis","arguments":{"patient_id":"P001"}}
		  {"type":"tool","name":"mcp.lis.get_lab_results","arguments":{"patient_id":"P001"}}
		  {"type":"tool","name":"mcp.vitals.get_vitals","arguments":{"patient_id":"P001"}}
		  {"type":"skill","name":"medical.first-day-note","arguments":{"patient_data":"{{mcp.his.get_patient_demographics}}","diagnosis":"{{mcp.his.get_diagnosis}}","lab_results":"{{mcp.lis.get_lab_results}}","vitals":"{{mcp.vitals.get_vitals}}"}}
		- Always generate at least one step. If no tools are relevant, generate a single skill step.

	/no_think`)

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
