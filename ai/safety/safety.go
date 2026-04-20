// Package safety provides content safety filtering for OpenBotStack.
//
// Implements regex-based detection of harmful content, PII masking,
// and secret/credential leak prevention. ML-based toxicity detection
// is reserved for the runtime layer.
package safety

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// SafetyCategory represents the type of safety concern.
type SafetyCategory string

const (
	CategoryHarmful SafetyCategory = "harmful"
	CategoryPII     SafetyCategory = "pii"
	CategoryToxic   SafetyCategory = "toxic"
	CategorySecret  SafetyCategory = "secret"
	CategoryCustom  SafetyCategory = "custom"
)

// SafetyVerdict is the result of a safety check.
type SafetyVerdict struct {
	Safe       bool
	Category   SafetyCategory
	Confidence float64
	Reason     string
	MaskedText string
}

// SafetyPolicy configures what categories to check.
type SafetyPolicy struct {
	TenantID          string
	EnabledCategories []SafetyCategory
	BlockThreshold    float64
	MaskPII           bool
}

// Filter checks content for safety concerns.
type Filter struct {
	policy SafetyPolicy

	// PII patterns
	emailRe *regexp.Regexp
	phoneRe *regexp.Regexp
	ssnRe   *regexp.Regexp

	// Secret patterns
	apiKeyRe    *regexp.Regexp
	passwordRe  *regexp.Regexp

	// Harmful keywords (simplified rule-based approach)
	harmfulWords []string
}

// NewFilter creates a safety filter with the given policy.
func NewFilter(policy SafetyPolicy) *Filter {
	f := &Filter{
		policy: policy,
		emailRe:    regexp.MustCompile(`(?i)\b[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}\b`),
		phoneRe:    regexp.MustCompile(`\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`),
		ssnRe:      regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
		apiKeyRe:   regexp.MustCompile(`(?i)(api[_\-]?key|sk\-|obs_)[a-zA-Z0-9\-_]{8,}`),
		passwordRe: regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[:=]\s*\S+`),
		harmfulWords: []string{
			"bomb", "kill", "attack", "weapon", "explosive",
			"hack", "exploit", "malware", "terrorist",
		},
	}
	return f
}

// hasCategory checks if a category is enabled in the policy.
func (f *Filter) hasCategory(cat SafetyCategory) bool {
	if len(f.policy.EnabledCategories) == 0 {
		return false
	}
	for _, c := range f.policy.EnabledCategories {
		if c == cat {
			return true
		}
	}
	return false
}

// exceedsThreshold returns true if confidence meets or exceeds BlockThreshold.
// If BlockThreshold is not set (0), defaults to always block (threshold 0).
func (f *Filter) exceedsThreshold(confidence float64) bool {
	threshold := f.policy.BlockThreshold
	if threshold <= 0 {
		return true
	}
	return confidence >= threshold
}

// CheckInput evaluates user input for safety concerns.
func (f *Filter) CheckInput(ctx context.Context, text string) (*SafetyVerdict, error) {
	if ctx == nil {
		return nil, fmt.Errorf("safety: context is required")
	}
	if text == "" {
		return &SafetyVerdict{Safe: true}, nil
	}

	// Check harmful content
	if f.hasCategory(CategoryHarmful) {
		lower := strings.ToLower(text)
		for _, word := range f.harmfulWords {
			if strings.Contains(lower, word) {
				confidence := 0.8
				if !f.exceedsThreshold(confidence) {
					continue
				}
				return &SafetyVerdict{
					Safe:       false,
					Category:   CategoryHarmful,
					Confidence: confidence,
					Reason:     fmt.Sprintf("potentially harmful content detected: %s", word),
				}, nil
			}
		}
	}

	return &SafetyVerdict{Safe: true}, nil
}

// CheckOutput evaluates AI output for safety concerns.
func (f *Filter) CheckOutput(ctx context.Context, text string) (*SafetyVerdict, error) {
	if ctx == nil {
		return nil, fmt.Errorf("safety: context is required")
	}
	if text == "" {
		return &SafetyVerdict{Safe: true}, nil
	}

	// Check for secrets/leaked credentials
	if f.hasCategory(CategorySecret) {
		if f.apiKeyRe.MatchString(text) {
			confidence := 0.9
			if f.exceedsThreshold(confidence) {
				return &SafetyVerdict{
					Safe:       false,
					Category:   CategorySecret,
					Confidence: confidence,
					Reason:     "potential API key or secret detected",
				}, nil
			}
		}
		if f.passwordRe.MatchString(text) {
			confidence := 0.85
			if f.exceedsThreshold(confidence) {
				return &SafetyVerdict{
					Safe:       false,
					Category:   CategorySecret,
					Confidence: confidence,
					Reason:     "potential password detected",
				}, nil
			}
		}
	}

	return &SafetyVerdict{Safe: true}, nil
}

// MaskPII detects and masks personally identifiable information.
func (f *Filter) MaskPII(ctx context.Context, text string) (string, []SafetyVerdict, error) {
	if ctx == nil {
		return "", nil, fmt.Errorf("safety: context is required")
	}
	if text == "" {
		return "", nil, nil
	}
	if !f.policy.MaskPII || !f.hasCategory(CategoryPII) {
		return text, nil, nil
	}

	var verdicts []SafetyVerdict
	masked := text

	// Mask emails (must have @ and domain)
	masked = f.emailRe.ReplaceAllStringFunc(masked, func(match string) string {
		// Only mask complete emails (has @ and domain)
		if strings.Contains(match, "@") && strings.Count(match, ".") >= 1 {
			parts := strings.SplitN(match, "@", 2)
			if len(parts[0]) > 0 && len(parts[1]) > 0 && strings.Contains(parts[1], ".") {
				verdicts = append(verdicts, SafetyVerdict{
					Category:   CategoryPII,
					Confidence: 0.95,
					Reason:     "email address detected",
					MaskedText: "[EMAIL]",
				})
				return "[EMAIL]"
			}
		}
		return match
	})

	// Mask phone numbers
	masked = f.phoneRe.ReplaceAllStringFunc(masked, func(match string) string {
		verdicts = append(verdicts, SafetyVerdict{
			Category:   CategoryPII,
			Confidence: 0.9,
			Reason:     "phone number detected",
			MaskedText: "[PHONE]",
		})
		return "[PHONE]"
	})

	// Mask SSN
	masked = f.ssnRe.ReplaceAllStringFunc(masked, func(match string) string {
		verdicts = append(verdicts, SafetyVerdict{
			Category:   CategoryPII,
			Confidence: 0.95,
			Reason:     "SSN detected",
			MaskedText: "[SSN]",
		})
		return "[SSN]"
	})

	return masked, verdicts, nil
}
