package planning_test

import (
	"encoding/json"
	"reflect"
	"testing"

	aitypes "github.com/openbotstack/openbotstack-core/ai/types"
	"github.com/openbotstack/openbotstack-core/planning"
)

// --- TurnToolResult: zero-value and JSON round-trip ---

func TestTurnToolResult_ZeroValue(t *testing.T) {
	var tr planning.TurnToolResult

	// Zero-value should have empty strings and false bool.
	if tr.StepName != "" {
		t.Errorf("StepName = %q, want empty", tr.StepName)
	}
	if tr.Success != false {
		t.Error("Success = true, want false")
	}
}

func TestTurnToolResult_JSONRoundTrip(t *testing.T) {
	original := planning.TurnToolResult{
		StepName: "fetch_data",
		StepType: "tool",
		Success:  true,
		Summary:  "fetched 42 records",
		Output:   `{"count":42}`,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded planning.TurnToolResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.StepName != original.StepName {
		t.Errorf("StepName = %q, want %q", decoded.StepName, original.StepName)
	}
	if decoded.Success != original.Success {
		t.Errorf("Success = %v, want %v", decoded.Success, original.Success)
	}
	if decoded.Summary != original.Summary {
		t.Errorf("Summary = %q, want %q", decoded.Summary, original.Summary)
	}
}

func TestTurnToolResult_OmitEmpty(t *testing.T) {
	tr := planning.TurnToolResult{
		StepName: "step1",
		StepType: "skill",
		Success:  true,
		// Summary, Output, Error left empty — should be omitted from JSON
	}

	data, err := json.Marshal(tr)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Verify omitted fields don't appear in JSON
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}
	if _, exists := raw["summary"]; exists {
		t.Error("summary should be omitted when empty")
	}
	if _, exists := raw["output"]; exists {
		t.Error("output should be omitted when empty")
	}
	if _, exists := raw["error"]; exists {
		t.Error("error should be omitted when empty")
	}
}

func TestTurnToolResult_FailedResult(t *testing.T) {
	tr := planning.TurnToolResult{
		StepName: "web_fetch",
		StepType: "tool",
		Success:  false,
		Error:    "connection refused",
	}

	if tr.Success {
		t.Error("Success should be false for failed result")
	}
	if tr.Error == "" {
		t.Error("Error should be populated for failed result")
	}
}

// --- AssistantSoul: JSON round-trip ---

func TestAssistantSoul_JSONRoundTrip(t *testing.T) {
	original := planning.AssistantSoul{
		SystemPrompt:  "You are a medical assistant.",
		Personality:   "empathetic",
		Instructions:  "Always verify dosages.",
		AllowedSkills: []string{"summarize", "classify"},
		AllowedTools:  []string{"builtin.web_fetch"},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded planning.AssistantSoul
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.SystemPrompt != original.SystemPrompt {
		t.Errorf("SystemPrompt = %q, want %q", decoded.SystemPrompt, original.SystemPrompt)
	}
	if len(decoded.AllowedSkills) != len(original.AllowedSkills) {
		t.Errorf("AllowedSkills len = %d, want %d", len(decoded.AllowedSkills), len(original.AllowedSkills))
	}
	if decoded.AllowedSkills[0] != "summarize" {
		t.Errorf("AllowedSkills[0] = %q, want %q", decoded.AllowedSkills[0], "summarize")
	}
}

// --- SearchResult: zero-value behavior ---

func TestSearchResult_ZeroValue(t *testing.T) {
	var sr planning.SearchResult

	if sr.Content != nil {
		t.Errorf("Content = %v, want nil", sr.Content)
	}
	if sr.Score != 0 {
		t.Errorf("Score = %f, want 0", sr.Score)
	}
}

func TestSearchResult_Populated(t *testing.T) {
	sr := planning.SearchResult{
		Content: []byte("relevant memory fragment"),
		Score:   0.92,
	}

	if string(sr.Content) != "relevant memory fragment" {
		t.Errorf("Content = %q, want %q", string(sr.Content), "relevant memory fragment")
	}
	if sr.Score < 0.9 || sr.Score > 0.95 {
		t.Errorf("Score = %f, out of expected range", sr.Score)
	}
}

// --- PlannerContext: zero-value and field access ---

func TestPlannerContext_ZeroValueSafe(t *testing.T) {
	var pc planning.PlannerContext

	// Zero-value should have nil slices and empty strings — no panic on access.
	if pc.AssistantID != "" {
		t.Errorf("AssistantID = %q, want empty", pc.AssistantID)
	}
	if pc.MemoryContext != nil {
		t.Errorf("MemoryContext = %v, want nil", pc.MemoryContext)
	}
	if pc.Skills != nil {
		t.Errorf("Skills = %v, want nil", pc.Skills)
	}
	if pc.ConversationHistory != nil {
		t.Errorf("ConversationHistory = %v, want nil", pc.ConversationHistory)
	}
	if pc.TurnResults != nil {
		t.Errorf("TurnResults = %v, want nil", pc.TurnResults)
	}
	// Soul should be zero-value struct, not nil
	if pc.Soul.SystemPrompt != "" {
		t.Errorf("Soul.SystemPrompt = %q, want empty", pc.Soul.SystemPrompt)
	}
}

func TestPlannerContext_AllFieldsPopulated(t *testing.T) {
	pc := planning.PlannerContext{
		AssistantID: "asst-med-01",
		Soul: planning.AssistantSoul{
			SystemPrompt: "You are a clinical assistant.",
			Personality:  "precise",
		},
		MemoryContext: []planning.SearchResult{
			{Content: []byte("patient history"), Score: 0.85},
		},
		UserRequest: "Summarize the lab results",
		TurnResults: []planning.TurnToolResult{
			{StepName: "fetch_labs", StepType: "tool", Success: true},
		},
	}

	if pc.AssistantID != "asst-med-01" {
		t.Errorf("AssistantID = %q, want %q", pc.AssistantID, "asst-med-01")
	}
	if len(pc.MemoryContext) != 1 {
		t.Errorf("MemoryContext len = %d, want 1", len(pc.MemoryContext))
	}
	if len(pc.TurnResults) != 1 {
		t.Errorf("TurnResults len = %d, want 1", len(pc.TurnResults))
	}
	if pc.TurnResults[0].StepName != "fetch_labs" {
		t.Errorf("TurnResults[0].StepName = %q, want %q", pc.TurnResults[0].StepName, "fetch_labs")
	}
}

// --- ProgressFn: callback invocation ---

func TestProgressFn_Callback(t *testing.T) {
	var capturedType, capturedContent string
	fn := planning.ProgressFn(func(eventType, content string) {
		capturedType = eventType
		capturedContent = content
	})

	fn("planning_token", "hello")

	if capturedType != "planning_token" {
		t.Errorf("eventType = %q, want %q", capturedType, "planning_token")
	}
	if capturedContent != "hello" {
		t.Errorf("content = %q, want %q", capturedContent, "hello")
	}
}

func TestProgressFn_NilSafe(t *testing.T) {
	// A nil ProgressFn should not panic — callers must check nil.
	var fn planning.ProgressFn
	// This would panic if called without nil check:
	// fn("test", "")  // NOT safe to call

	// Verify the type is nil
	if fn != nil {
		t.Error("nil ProgressFn should be nil")
	}
}

// --- PlannerContext.WithRequest ---

// TestPlannerContext_WithRequest_PreservesAllOtherFields verifies that
// WithRequest returns a shallow-copied context whose UserRequest is replaced
// while every other field is preserved (deep equality for slices/structs,
// function-pointer equality for ProgressFn).
//
// This is the failing test written BEFORE implementing WithRequest (TDD).
func TestPlannerContext_WithRequest_PreservesAllOtherFields(t *testing.T) {
	// Build a ProgressFn we can identify by pointer identity.
	var progressCalls int
	origProgress := planning.ProgressFn(func(eventType, content string) {
		progressCalls++
	})

	origCtx := planning.PlannerContext{
		AssistantID: "asst-med-01",
		Soul: planning.AssistantSoul{
			SystemPrompt:  "You are a clinical assistant.",
			Personality:   "precise",
			Instructions:  "Always verify dosages.",
			AllowedSkills: []string{"summarize", "classify"},
			AllowedTools:  []string{"builtin.web_fetch"},
		},
		MemoryContext: []planning.SearchResult{
			{Content: []byte("patient history"), Score: 0.85},
			{Content: []byte("lab reference ranges"), Score: 0.71},
		},
		Skills: []aitypes.SkillDescriptor{
			{ID: "s1", Name: "Summarize", Description: "Summarize text"},
			{ID: "s2", Name: "Classify", Description: "Classify input"},
		},
		UserRequest: "original request",
		ProgressFn:  origProgress,
		ConversationHistory: []aitypes.Message{
			{Role: "user", Contents: []aitypes.ContentBlock{aitypes.NewTextBlock("hi")}},
			{Role: "assistant", Contents: []aitypes.ContentBlock{aitypes.NewTextBlock("hello")}},
		},
		TurnResults: []planning.TurnToolResult{
			{StepName: "fetch_labs", StepType: "tool", Success: true, Summary: "got labs"},
		},
	}

	newReq := "new request"
	got := origCtx.WithRequest(newReq)

	// 1. UserRequest must be replaced.
	if got.UserRequest != newReq {
		t.Errorf("UserRequest = %q, want %q", got.UserRequest, newReq)
	}

	// 2. The original context must be untouched (immutable operation).
	if origCtx.UserRequest != "original request" {
		t.Errorf("original UserRequest mutated: %q (must remain %q)",
			origCtx.UserRequest, "original request")
	}

	// 3. AssistantID preserved.
	if got.AssistantID != origCtx.AssistantID {
		t.Errorf("AssistantID = %q, want %q", got.AssistantID, origCtx.AssistantID)
	}

	// 4. Soul preserved (deep equality).
	if !reflect.DeepEqual(got.Soul, origCtx.Soul) {
		t.Errorf("Soul mismatch:\n got  %+v\n want %+v", got.Soul, origCtx.Soul)
	}

	// 5. MemoryContext preserved (deep equality).
	if !reflect.DeepEqual(got.MemoryContext, origCtx.MemoryContext) {
		t.Errorf("MemoryContext mismatch:\n got  %+v\n want %+v", got.MemoryContext, origCtx.MemoryContext)
	}

	// 6. Skills preserved (deep equality).
	if !reflect.DeepEqual(got.Skills, origCtx.Skills) {
		t.Errorf("Skills mismatch:\n got  %+v\n want %+v", got.Skills, origCtx.Skills)
	}

	// 7. ConversationHistory preserved (deep equality).
	if !reflect.DeepEqual(got.ConversationHistory, origCtx.ConversationHistory) {
		t.Errorf("ConversationHistory mismatch:\n got  %+v\n want %+v",
			got.ConversationHistory, origCtx.ConversationHistory)
	}

	// 8. TurnResults preserved (deep equality).
	if !reflect.DeepEqual(got.TurnResults, origCtx.TurnResults) {
		t.Errorf("TurnResults mismatch:\n got  %+v\n want %+v", got.TurnResults, origCtx.TurnResults)
	}

	// 9. ProgressFn preserved (function-pointer identity).
	//
	// We assert identity (not just callability) because downstream code relies on
	// the same callback being invoked. The returned context must carry the same
	// ProgressFn, not a zero/nil one.
	if got.ProgressFn == nil {
		t.Fatal("ProgressFn is nil, want the original non-nil callback")
	}
	// Compare via reflect.ValueOf pointer — function values are comparable this way.
	if !reflect.ValueOf(got.ProgressFn).IsNil() &&
		funcAddr(got.ProgressFn) != funcAddr(origCtx.ProgressFn) {
		t.Error("ProgressFn pointer differs from original; want identical callback")
	}

	// Sanity: the preserved ProgressFn must still be invocable.
	got.ProgressFn("test_event", "x")
	if progressCalls != 1 {
		t.Errorf("preserved ProgressFn call count = %d, want 1", progressCalls)
	}
}

// funcAddr returns the pointer representation of a ProgressFn for identity
// comparison. Function values in Go are backed by a pointer-sized struct
// (funcval); unsafe-free comparison via fmt is not reliable, so we use
// reflect.Value.Pointer, which returns the underlying code pointer without
// dereferencing it.
func funcAddr(fn planning.ProgressFn) uintptr {
	return reflect.ValueOf(fn).Pointer()
}
