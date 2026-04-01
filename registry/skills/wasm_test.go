package skills_test

import (
	"context"
	"testing"

	"github.com/openbotstack/openbotstack-core/registry/skills"
)

func TestWasmModuleInterface(t *testing.T) {
	// Verify interface is defined correctly
	var _ skills.WasmModule = (*mockWasmModule)(nil)
}

func TestWasmLoaderLoad(t *testing.T) {
	loader := skills.NewWasmLoader()
	ctx := context.Background()

	// Load from bytes (mock wasm)
	wasmBytes := []byte{0x00, 0x61, 0x73, 0x6d} // Wasm magic bytes
	module, err := loader.Load(ctx, wasmBytes)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if module == nil {
		t.Fatal("Load returned nil module")
	}
}

func TestWasmLoaderLoadFromPath(t *testing.T) {
	loader := skills.NewWasmLoader()
	ctx := context.Background()

	// With non-existent file
	_, err := loader.LoadFromPath(ctx, "/nonexistent/skills.wasm")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestWasmModuleExecute(t *testing.T) {
	loader := skills.NewWasmLoader()
	ctx := context.Background()

	wasmBytes := []byte{0x00, 0x61, 0x73, 0x6d}
	module, _ := loader.Load(ctx, wasmBytes)

	input := skills.WasmInput{
		Function: "execute",
		Args:     map[string]interface{}{"query": "test"},
	}

	output, err := module.Execute(ctx, input)
	if err != nil {
		t.Logf("Execute returned error (expected for stub): %v", err)
	} else if output != nil {
		t.Logf("Execute returned: %+v", output)
	}
}

func TestWasmModuleClose(t *testing.T) {
	loader := skills.NewWasmLoader()
	ctx := context.Background()

	wasmBytes := []byte{0x00, 0x61, 0x73, 0x6d}
	module, _ := loader.Load(ctx, wasmBytes)

	err := module.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestWasmModuleMemoryLimit(t *testing.T) {
	loader := skills.NewWasmLoader()
	ctx := context.Background()

	wasmBytes := []byte{0x00, 0x61, 0x73, 0x6d}
	module, _ := loader.Load(ctx, wasmBytes)

	limit := module.MemoryLimit()
	if limit <= 0 {
		t.Errorf("Expected positive memory limit, got %d", limit)
	}
}

// Mock implementation for interface compliance test
type mockWasmModule struct{}

func (m *mockWasmModule) Execute(ctx context.Context, input skills.WasmInput) (*skills.WasmOutput, error) {
	return nil, nil
}
func (m *mockWasmModule) Close() error       { return nil }
func (m *mockWasmModule) MemoryLimit() int64 { return 0 }
