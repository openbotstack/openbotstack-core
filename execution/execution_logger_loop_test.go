package execution

import (
	"encoding/json"
	"testing"
)

func TestExecutionLogRecord_LoopFieldsOmitEmpty(t *testing.T) {
	// When TaskIndex and TurnNumber are nil, they should not appear in JSON.
	record := ExecutionLogRecord{
		RequestID: "req1",
		StepName:  "test",
		StepType:  "tool",
		Status:    "success",
	}

	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	// TaskIndex and TurnNumber should NOT be present when nil
	if _, exists := parsed["task_index"]; exists {
		t.Error("task_index should not be present when nil")
	}
	if _, exists := parsed["turn_number"]; exists {
		t.Error("turn_number should not be present when nil")
	}
	// LoopMode should not be present when empty
	if _, exists := parsed["loop_mode"]; exists {
		t.Error("loop_mode should not be present when empty")
	}
}

func TestExecutionLogRecord_LoopFieldsPresent(t *testing.T) {
	taskIdx := 2
	turnNum := 5

	record := ExecutionLogRecord{
		RequestID:  "req1",
		StepName:   "test",
		StepType:   "skill",
		Status:     "success",
		LoopMode:   "dual_loop",
		TaskIndex:  &taskIdx,
		TurnNumber: &turnNum,
	}

	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if v := parsed["loop_mode"]; v != "dual_loop" {
		t.Errorf("loop_mode = %v, want %q", v, "dual_loop")
	}
	if v := parsed["task_index"]; v != float64(2) {
		t.Errorf("task_index = %v, want 2", v)
	}
	if v := parsed["turn_number"]; v != float64(5) {
		t.Errorf("turn_number = %v, want 5", v)
	}
}

func TestExecutionLogRecord_LoopFieldsRoundTrip(t *testing.T) {
	taskIdx := 10
	turnNum := 3

	original := ExecutionLogRecord{
		RequestID:   "req1",
		AssistantID: "asst1",
		StepName:    "fetch_data",
		StepType:    "tool",
		Status:      "running",
		LoopMode:    "dual_loop",
		TaskIndex:   &taskIdx,
		TurnNumber:  &turnNum,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var decoded ExecutionLogRecord
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if decoded.LoopMode != original.LoopMode {
		t.Errorf("LoopMode = %q, want %q", decoded.LoopMode, original.LoopMode)
	}
	if decoded.TaskIndex == nil || *decoded.TaskIndex != taskIdx {
		t.Errorf("TaskIndex = %v, want %d", decoded.TaskIndex, taskIdx)
	}
	if decoded.TurnNumber == nil || *decoded.TurnNumber != turnNum {
		t.Errorf("TurnNumber = %v, want %d", decoded.TurnNumber, turnNum)
	}
}

func TestExecutionLogRecord_ExistingFieldsStillWork(t *testing.T) {
	record := ExecutionLogRecord{
		RequestID:   "req1",
		AssistantID: "asst1",
		StepName:    "summarize",
		StepType:    "skill",
		Status:      "success",
		Error:       "",
	}

	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if v := parsed["request_id"]; v != "req1" {
		t.Errorf("request_id = %v, want %q", v, "req1")
	}
	if v := parsed["step_name"]; v != "summarize" {
		t.Errorf("step_name = %v, want %q", v, "summarize")
	}
	if v := parsed["step_type"]; v != "skill" {
		t.Errorf("step_type = %v, want %q", v, "skill")
	}
	if v := parsed["status"]; v != "success" {
		t.Errorf("status = %v, want %q", v, "success")
	}
}
