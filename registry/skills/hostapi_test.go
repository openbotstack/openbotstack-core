package skills_test

import (
	"context"
	"testing"

	"github.com/openbotstack/openbotstack-core/registry/skills"
)

func TestHostAPIKVGet(t *testing.T) {
	api := skills.NewHostAPI()
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
	api := skills.NewHostAPI()
	ctx := context.Background()

	_, err := api.KVGet(ctx, "nonexistent")
	if err != skills.ErrKeyNotFound {
		t.Errorf("Expected ErrKeyNotFound, got %v", err)
	}
}

func TestHostAPIHTTPFetch(t *testing.T) {
	api := skills.NewHostAPI()
	ctx := context.Background()

	req := skills.HTTPRequest{
		Method: "GET",
		URL:    "https://example.com",
	}

	resp, err := api.HTTPFetch(ctx, req)
	if err != nil {
		t.Fatalf("HTTPFetch returned unexpected error: %v", err)
	}

	// Core's HostAPI is an intentional stub (no network calls per AI_CONTRACT.md).
	// The real HTTP execution lives in runtime/sandbox/wasm/hostapi_http.go and is
	// wired via toolrunner/tool_invocation.WireHTTPFetch.
	if resp.StatusCode != 200 {
		t.Errorf("Expected stub status 200, got %d", resp.StatusCode)
	}
	if string(resp.Body) != "stub response" {
		t.Errorf("Expected stub body 'stub response', got %q", string(resp.Body))
	}
}

func TestHostAPIHTTPFetchValidation(t *testing.T) {
	api := skills.NewHostAPI()
	ctx := context.Background()

	req := skills.HTTPRequest{
		Method: "GET",
		URL:    "", // Invalid
	}

	_, err := api.HTTPFetch(ctx, req)
	if err == nil {
		t.Error("Expected error for empty URL")
	}
}
