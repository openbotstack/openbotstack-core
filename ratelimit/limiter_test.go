package ratelimit_test

import (
	"testing"
	"time"

	"github.com/openbotstack/openbotstack-core/ratelimit"
)

func TestRateLimitKey(t *testing.T) {
	key := ratelimit.RateLimitKey{
		TenantID: "tenant-1",
		UserID:   "user-1",
	}

	if key.TenantID != "tenant-1" {
		t.Errorf("Expected TenantID 'tenant-1', got '%s'", key.TenantID)
	}
	if key.UserID != "user-1" {
		t.Errorf("Expected UserID 'user-1', got '%s'", key.UserID)
	}
}

func TestQuotaConfig(t *testing.T) {
	config := ratelimit.QuotaConfig{
		TenantTokensPerMinute:   5_000_000,
		TenantRequestsPerMinute: 1000,
		UserRequestsPerMinute:   60,
		UserTokensPerMinute:     100_000,
	}

	if config.TenantTokensPerMinute != 5_000_000 {
		t.Errorf("Expected 5M tokens, got %d", config.TenantTokensPerMinute)
	}
}

func TestRateLimitResult(t *testing.T) {
	result := ratelimit.RateLimitResult{
		Allowed:    true,
		Remaining:  100,
		ResetAt:    time.Now().Add(time.Minute),
		RetryAfter: 0,
	}

	if !result.Allowed {
		t.Error("Expected Allowed to be true")
	}
}
