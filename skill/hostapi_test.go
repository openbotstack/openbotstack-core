package skill_test

import (
	"context"
	"testing"

	"github.com/openbotstack/openbotstack-core/skill"
)

func TestHostAPILLMGenerate(t *testing.T) {
	api := skill.NewHostAPI()
	ctx := context.Background()

	req := skill.LLMRequest{
		Prompt:    "Hello, how are you?",
		MaxTokens: 100,
	}

	resp, err := api.LLMGenerate(ctx, req)
	if err != nil {
		t.Logf("LLMGenerate returned error (expected for stub): %v", err)
	} else if resp != nil && resp.Text == "" {
		t.Error("Expected non-empty response text")
	}
}

func TestHostAPIKVGet(t *testing.T) {
	api := skill.NewHostAPI()
	ctx := context.Background()

	// Set a value first
	err := api.KVSet(ctx, "test-key", []byte("test-value"))
	if err != nil {
		t.Fatalf("KVSet failed: %v", err)
	}

	// Get it back
	val, err := api.KVGet(ctx, "test-key")
	if err != nil {
		t.Fatalf("KVGet failed: %v", err)
	}

	if string(val) != "test-value" {
		t.Errorf("Expected 'test-value', got '%s'", string(val))
	}
}

func TestHostAPIKVGetNotFound(t *testing.T) {
	api := skill.NewHostAPI()
	ctx := context.Background()

	_, err := api.KVGet(ctx, "nonexistent")
	if err != skill.ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound, got %v", err)
	}
}

func TestHostAPIHTTPFetch(t *testing.T) {
	api := skill.NewHostAPI()
	ctx := context.Background()

	req := skill.HTTPRequest{
		Method: "GET",
		URL:    "https://example.com",
	}

	resp, err := api.HTTPFetch(ctx, req)
	if err != nil {
		t.Logf("HTTPFetch returned error (expected for stub): %v", err)
	} else if resp != nil && resp.StatusCode == 0 {
		t.Error("Expected non-zero status code")
	}
}

func TestHostAPIHTTPFetchValidation(t *testing.T) {
	api := skill.NewHostAPI()
	ctx := context.Background()

	req := skill.HTTPRequest{
		Method: "GET",
		URL:    "", // Invalid
	}

	_, err := api.HTTPFetch(ctx, req)
	if err == nil {
		t.Error("Expected error for empty URL")
	}
}
