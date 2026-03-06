package skill

import (
	"context"
	"errors"
	"os"
)

var (
	// ErrWasmLoadFailed is returned when Wasm loading fails.
	ErrWasmLoadFailed = errors.New("skill: wasm load failed")

	// ErrWasmExecuteFailed is returned when Wasm execution fails.
	ErrWasmExecuteFailed = errors.New("skill: wasm execute failed")
)

// WasmInput represents input for Wasm execution.
type WasmInput struct {
	Function string
	Args     map[string]interface{}
}

// WasmOutput represents output from Wasm execution.
type WasmOutput struct {
	Result interface{}
	Logs   []string
}

// WasmModule represents a loaded Wasm skill module.
type WasmModule interface {
	// Execute runs a function in the Wasm module.
	Execute(ctx context.Context, input WasmInput) (*WasmOutput, error)

	// Close releases resources.
	Close() error

	// MemoryLimit returns the memory limit in bytes.
	MemoryLimit() int64
}

// WasmLoader loads Wasm modules from bytes or files.
type WasmLoader struct {
	defaultMemoryLimit int64
}

// NewWasmLoader creates a new Wasm loader.
func NewWasmLoader() *WasmLoader {
	return &WasmLoader{
		defaultMemoryLimit: 128 * 1024 * 1024, // 128MB
	}
}

// Load loads a Wasm module from bytes.
func (l *WasmLoader) Load(ctx context.Context, wasmBytes []byte) (WasmModule, error) {
	// Validate Wasm magic bytes
	if len(wasmBytes) < 4 {
		return nil, ErrWasmLoadFailed
	}
	if wasmBytes[0] != 0x00 || wasmBytes[1] != 0x61 ||
		wasmBytes[2] != 0x73 || wasmBytes[3] != 0x6d {
		return nil, ErrWasmLoadFailed
	}

	// TODO: Use wasmtime/wasmer to actually load
	return &stubWasmModule{
		memoryLimit: l.defaultMemoryLimit,
	}, nil
}

// LoadFromPath loads a Wasm module from a file path.
func (l *WasmLoader) LoadFromPath(ctx context.Context, path string) (WasmModule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return l.Load(ctx, data)
}

// stubWasmModule is a stub implementation for testing.
type stubWasmModule struct {
	memoryLimit int64
}

func (m *stubWasmModule) Execute(ctx context.Context, input WasmInput) (*WasmOutput, error) {
	// Stub - would actually call Wasm runtime
	return &WasmOutput{
		Result: "stub execution",
		Logs:   []string{},
	}, nil
}

func (m *stubWasmModule) Close() error {
	return nil
}

func (m *stubWasmModule) MemoryLimit() int64 {
	return m.memoryLimit
}
