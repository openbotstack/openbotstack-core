package audit

import "testing"

// mockMapper is a minimal implementation used only to verify the interface contract.
type mockMapper struct{}

func (mockMapper) Format() string                              { return "test_format" }
func (mockMapper) Map(AuditEnvelope) (any, error)              { return nil, nil }
func (mockMapper) MapBatch([]AuditEnvelope) (any, error)       { return nil, nil }

func TestAuditEventMapper_Interface(t *testing.T) {
	// Verify mockMapper satisfies AuditEventMapper at compile time.
	var _ AuditEventMapper = mockMapper{}

	var m AuditEventMapper = mockMapper{}
	if f := m.Format(); f != "test_format" {
		t.Errorf("Format() = %q, want %q", f, "test_format")
	}
}
