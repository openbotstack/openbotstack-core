package assistant

import "testing"

func TestAssistantRuntime_Construction(t *testing.T) {
	rt := &AssistantRuntime{
		AssistantID: "test-assistant",
	}
	if rt.AssistantID != "test-assistant" {
		t.Errorf("AssistantID = %q, want %q", rt.AssistantID, "test-assistant")
	}
}
