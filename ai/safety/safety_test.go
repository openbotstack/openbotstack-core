package safety

import (
	"context"
	"strings"
	"testing"
)

// --- Normal Cases (4) ---

func TestCheckInput_SafeText(t *testing.T) {
	f := NewFilter(SafetyPolicy{EnabledCategories: []SafetyCategory{CategoryHarmful, CategoryToxic}})
	v, err := f.CheckInput(context.Background(), "Hello, how are you today?")
	if err != nil {
		t.Fatalf("CheckInput: %v", err)
	}
	if !v.Safe {
		t.Error("clean input should be safe")
	}
}

func TestCheckOutput_SafeText(t *testing.T) {
	f := NewFilter(SafetyPolicy{EnabledCategories: []SafetyCategory{CategoryHarmful, CategoryToxic}})
	v, err := f.CheckOutput(context.Background(), "The weather is nice today.")
	if err != nil {
		t.Fatalf("CheckOutput: %v", err)
	}
	if !v.Safe {
		t.Error("clean output should be safe")
	}
}

func TestMaskPII_DetectsEmail(t *testing.T) {
	f := NewFilter(SafetyPolicy{MaskPII: true, EnabledCategories: []SafetyCategory{CategoryPII}})
	masked, verdicts, err := f.MaskPII(context.Background(), "Contact me at john@example.com please")
	if err != nil {
		t.Fatalf("MaskPII: %v", err)
	}
	if len(verdicts) == 0 {
		t.Fatal("expected PII verdicts for email")
	}
	if strings.Contains(masked, "john@example.com") {
		t.Error("email should be masked in output")
	}
}

func TestMaskPII_DetectsPhone(t *testing.T) {
	f := NewFilter(SafetyPolicy{MaskPII: true, EnabledCategories: []SafetyCategory{CategoryPII}})
	masked, verdicts, err := f.MaskPII(context.Background(), "Call me at 555-123-4567")
	if err != nil {
		t.Fatalf("MaskPII: %v", err)
	}
	if len(verdicts) == 0 {
		t.Fatal("expected PII verdicts for phone number")
	}
	if strings.Contains(masked, "555-123-4567") {
		t.Error("phone should be masked in output")
	}
}

// --- Abnormal / Edge Cases (14) ---

func TestCheckInput_EmptyText(t *testing.T) {
	f := NewFilter(SafetyPolicy{})
	v, err := f.CheckInput(context.Background(), "")
	if err != nil {
		t.Fatalf("CheckInput empty: %v", err)
	}
	if !v.Safe {
		t.Error("empty text should be safe by default")
	}
}

func TestCheckInput_NilContext(t *testing.T) {
	f := NewFilter(SafetyPolicy{})
	_, err := f.CheckInput(nil, "hello")
	if err == nil {
		t.Error("expected error for nil context")
	}
}

func TestCheckOutput_EmptyText(t *testing.T) {
	f := NewFilter(SafetyPolicy{})
	v, err := f.CheckOutput(context.Background(), "")
	if err != nil {
		t.Fatalf("CheckOutput empty: %v", err)
	}
	if !v.Safe {
		t.Error("empty text should be safe")
	}
}

func TestCheckOutput_NilContext(t *testing.T) {
	f := NewFilter(SafetyPolicy{})
	_, err := f.CheckOutput(nil, "hello")
	if err == nil {
		t.Error("expected error for nil context")
	}
}

func TestMaskPII_NoPII(t *testing.T) {
	f := NewFilter(SafetyPolicy{MaskPII: true, EnabledCategories: []SafetyCategory{CategoryPII}})
	masked, verdicts, err := f.MaskPII(context.Background(), "No sensitive data here")
	if err != nil {
		t.Fatalf("MaskPII: %v", err)
	}
	if len(verdicts) != 0 {
		t.Errorf("expected 0 verdicts for clean text, got %d", len(verdicts))
	}
	if masked != "No sensitive data here" {
		t.Errorf("text should be unchanged, got %q", masked)
	}
}

func TestMaskPII_MultiplePII(t *testing.T) {
	f := NewFilter(SafetyPolicy{MaskPII: true, EnabledCategories: []SafetyCategory{CategoryPII}})
	text := "Email john@test.com or call 555-000-1111, SSN 123-45-6789"
	masked, verdicts, err := f.MaskPII(context.Background(), text)
	if err != nil {
		t.Fatalf("MaskPII: %v", err)
	}
	if len(verdicts) < 2 {
		t.Errorf("expected at least 2 PII detections, got %d", len(verdicts))
	}
	if strings.Contains(masked, "john@test.com") || strings.Contains(masked, "555-000-1111") {
		t.Error("PII should be masked")
	}
}

func TestMaskPII_NilContext(t *testing.T) {
	f := NewFilter(SafetyPolicy{})
	_, _, err := f.MaskPII(nil, "text")
	if err == nil {
		t.Error("expected error for nil context")
	}
}

func TestMaskPII_EmptyText(t *testing.T) {
	f := NewFilter(SafetyPolicy{MaskPII: true, EnabledCategories: []SafetyCategory{CategoryPII}})
	masked, verdicts, err := f.MaskPII(context.Background(), "")
	if err != nil {
		t.Fatalf("MaskPII empty: %v", err)
	}
	if masked != "" {
		t.Errorf("empty input should return empty, got %q", masked)
	}
	if len(verdicts) != 0 {
		t.Errorf("expected 0 verdicts for empty text, got %d", len(verdicts))
	}
}

func TestCheckInput_VeryLongText(t *testing.T) {
	f := NewFilter(SafetyPolicy{})
	longText := strings.Repeat("a", 100001)
	v, err := f.CheckInput(context.Background(), longText)
	if err != nil {
		t.Fatalf("CheckInput long text: %v", err)
	}
	if !v.Safe {
		t.Error("long but clean text should be safe")
	}
}

func TestCheckInput_UnicodeText(t *testing.T) {
	f := NewFilter(SafetyPolicy{})
	unicode := "你好世界 こんにちは مرحبا العالم 🌍"
	v, err := f.CheckInput(context.Background(), unicode)
	if err != nil {
		t.Fatalf("CheckInput unicode: %v", err)
	}
	if !v.Safe {
		t.Error("clean unicode text should be safe")
	}
}

func TestCheckInput_HarmfulDetected(t *testing.T) {
	f := NewFilter(SafetyPolicy{EnabledCategories: []SafetyCategory{CategoryHarmful}})
	v, err := f.CheckInput(context.Background(), "how to make a bomb")
	if err != nil {
		t.Fatalf("CheckInput: %v", err)
	}
	if v.Safe {
		t.Error("harmful content should not be safe")
	}
	if v.Category != CategoryHarmful {
		t.Errorf("category = %q, want %q", v.Category, CategoryHarmful)
	}
}

func TestCheckOutput_SecretDetected(t *testing.T) {
	f := NewFilter(SafetyPolicy{EnabledCategories: []SafetyCategory{CategorySecret}})
	v, err := f.CheckOutput(context.Background(), "my API key is sk-abc123def456ghi789")
	if err != nil {
		t.Fatalf("CheckOutput: %v", err)
	}
	if v.Safe {
		t.Error("secret content should not be safe")
	}
}

func TestMaskPII_PartialMatch(t *testing.T) {
	f := NewFilter(SafetyPolicy{MaskPII: true, EnabledCategories: []SafetyCategory{CategoryPII}})
	masked, verdicts, err := f.MaskPII(context.Background(), "my email is john@")
	if err != nil {
		t.Fatalf("MaskPII: %v", err)
	}
	if len(verdicts) != 0 {
		t.Error("incomplete email should not be detected as PII")
	}
	if masked != "my email is john@" {
		t.Errorf("text should be unchanged, got %q", masked)
	}
}

func TestSafetyPolicy_EmptyCategories(t *testing.T) {
	f := NewFilter(SafetyPolicy{EnabledCategories: nil})
	v, err := f.CheckInput(context.Background(), "anything goes")
	if err != nil {
		t.Fatalf("CheckInput: %v", err)
	}
	if !v.Safe {
		t.Error("no categories enabled = always safe")
	}
}

func TestCheckOutput_PasswordDetected(t *testing.T) {
	f := NewFilter(SafetyPolicy{EnabledCategories: []SafetyCategory{CategorySecret}})
	v, err := f.CheckOutput(context.Background(), "password=supersecret123")
	if err != nil {
		t.Fatalf("CheckOutput: %v", err)
	}
	if v.Safe {
		t.Error("password pattern should not be safe")
	}
	if v.Category != CategorySecret {
		t.Errorf("category = %q, want %q", v.Category, CategorySecret)
	}
}

func TestCheckInput_CategoryToxicNoOp(t *testing.T) {
	f := NewFilter(SafetyPolicy{EnabledCategories: []SafetyCategory{CategoryToxic}})
	v, err := f.CheckInput(context.Background(), "some toxic harmful content")
	if err != nil {
		t.Fatalf("CheckInput: %v", err)
	}
	if !v.Safe {
		t.Error("CategoryToxic is reserved and should not trigger detection")
	}
}

func TestBlockThreshold_BlocksHighConfidence(t *testing.T) {
	f := NewFilter(SafetyPolicy{
		EnabledCategories: []SafetyCategory{CategoryHarmful},
		BlockThreshold:    0.9,
	})
	v, err := f.CheckInput(context.Background(), "how to make a bomb")
	if err != nil {
		t.Fatalf("CheckInput: %v", err)
	}
	if !v.Safe {
		t.Error("confidence 0.8 < threshold 0.9 should not block")
	}
}

func TestBlockThreshold_BlocksWhenAboveThreshold(t *testing.T) {
	f := NewFilter(SafetyPolicy{
		EnabledCategories: []SafetyCategory{CategorySecret},
		BlockThreshold:    0.85,
	})
	v, err := f.CheckOutput(context.Background(), "key is sk-abc123def456ghi789")
	if err != nil {
		t.Fatalf("CheckOutput: %v", err)
	}
	if v.Safe {
		t.Error("confidence 0.9 >= threshold 0.85 should block")
	}
}

func TestMaskPII_RespectsPolicyFlag(t *testing.T) {
	f := NewFilter(SafetyPolicy{MaskPII: false})
	masked, verdicts, err := f.MaskPII(context.Background(), "email john@test.com")
	if err != nil {
		t.Fatalf("MaskPII: %v", err)
	}
	if len(verdicts) != 0 {
		t.Error("MaskPII=false should not mask")
	}
	if masked != "email john@test.com" {
		t.Errorf("text should be unchanged when MaskPII=false, got %q", masked)
	}
}

func TestMaskPII_RespectsCategoryEnabled(t *testing.T) {
	f := NewFilter(SafetyPolicy{MaskPII: true, EnabledCategories: []SafetyCategory{CategoryHarmful}})
	masked, verdicts, err := f.MaskPII(context.Background(), "email john@test.com")
	if err != nil {
		t.Fatalf("MaskPII: %v", err)
	}
	if len(verdicts) != 0 {
		t.Error("PII category not enabled should not mask")
	}
	if masked != "email john@test.com" {
		t.Errorf("text should be unchanged when PII category not enabled, got %q", masked)
	}
}
