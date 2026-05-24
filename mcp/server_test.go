package mcp

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"
)

// mockToolProvider implements ToolProvider for testing.
type mockToolProvider struct {
	tools []Tool
}

func (m *mockToolProvider) ListTools() []Tool {
	return m.tools
}

func (m *mockToolProvider) CallTool(name string, arguments map[string]any) (*ToolCallResult, error) {
	return &ToolCallResult{
		Content: []ContentItem{{Type: "text", Text: "mock result for " + name}},
	}, nil
}

func TestServer_HandleToolsList(t *testing.T) {
	server := NewServer(&mockToolProvider{
		tools: []Tool{
			{Name: "search", Description: "Search", InputSchema: map[string]any{"type": "object"}},
			{Name: "calc", Description: "Calculate", InputSchema: map[string]any{"type": "object"}},
		},
	})

	req := JSONRPCRequest{JSONRPC: "2.0", ID: 1, Method: "tools/list"}
	resp := server.HandleRequest(req)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}

	var result ToolListResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if len(result.Tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(result.Tools))
	}
	if result.Tools[0].Name != "search" {
		t.Errorf("expected first tool 'search', got %q", result.Tools[0].Name)
	}
}

func TestServer_HandleToolsList_Empty(t *testing.T) {
	server := NewServer(&mockToolProvider{tools: nil})
	req := JSONRPCRequest{JSONRPC: "2.0", ID: 1, Method: "tools/list"}
	resp := server.HandleRequest(req)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	resultBytes, _ := json.Marshal(resp.Result)
	var result ToolListResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result.Tools) != 0 {
		t.Errorf("expected 0 tools, got %d", len(result.Tools))
	}
}

func TestServer_HandleToolsCall(t *testing.T) {
	server := NewServer(&mockToolProvider{tools: nil})

	params, _ := json.Marshal(map[string]any{
		"name":      "search",
		"arguments": map[string]any{"query": "golang"},
	})
	var rawParams any
	json.Unmarshal(params, &rawParams)

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/call",
		Params:  rawParams,
	}
	resp := server.HandleRequest(req)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	resultBytes, _ := json.Marshal(resp.Result)
	var result ToolCallResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result.Content) == 0 {
		t.Error("expected content")
	}
	if result.Content[0].Text != "mock result for search" {
		t.Errorf("unexpected result: %q", result.Content[0].Text)
	}
}

func TestServer_HandleToolsCall_MissingName(t *testing.T) {
	server := NewServer(&mockToolProvider{})

	params, _ := json.Marshal(map[string]any{"arguments": map[string]any{}})
	var rawParams any
	json.Unmarshal(params, &rawParams)

	req := JSONRPCRequest{JSONRPC: "2.0", ID: 3, Method: "tools/call", Params: rawParams}
	resp := server.HandleRequest(req)

	if resp.Error == nil {
		t.Fatal("missing name should return error")
	}
	if resp.Error.Code != ErrCodeInvalidParams {
		t.Errorf("expected InvalidParams, got code %d", resp.Error.Code)
	}
}

func TestServer_HandleMethodNotFound(t *testing.T) {
	server := NewServer(&mockToolProvider{})
	req := JSONRPCRequest{JSONRPC: "2.0", ID: 4, Method: "unknown/method"}
	resp := server.HandleRequest(req)

	if resp.Error == nil {
		t.Fatal("unknown method should return error")
	}
	if resp.Error.Code != ErrCodeMethodNotFound {
		t.Errorf("expected MethodNotFound, got code %d", resp.Error.Code)
	}
}

func TestServer_HandleNotification(t *testing.T) {
	// Notifications produce no response per JSON-RPC spec
	server := NewServer(&mockToolProvider{})
	server.HandleNotification(JSONRPCNotification{JSONRPC: "2.0", Method: "initialized"})
	// No panic = pass
}

func TestServer_HandleInitialize(t *testing.T) {
	server := NewServer(&mockToolProvider{})
	req := JSONRPCRequest{JSONRPC: "2.0", ID: 5, Method: "initialize"}
	resp := server.HandleRequest(req)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	resultBytes, _ := json.Marshal(resp.Result)
	var result map[string]any
	json.Unmarshal(resultBytes, &result)

	info, ok := result["serverInfo"].(map[string]any)
	if !ok {
		t.Fatal("expected serverInfo in result")
	}
	if info["name"] != "openbotstack" {
		t.Errorf("expected server name 'openbotstack', got %v", info["name"])
	}
}
func TestServer_HandleToolsList_NilInputSchema(t *testing.T) {
	server := NewServer(&mockToolProvider{
		tools: []Tool{{Name: "noop", Description: "No schema", InputSchema: nil}},
	})
	req := JSONRPCRequest{JSONRPC: "2.0", ID: 1, Method: "tools/list"}
	resp := server.HandleRequest(req)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	resultBytes, _ := json.Marshal(resp.Result)
	var result ToolListResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Tools[0].InputSchema == nil {
		t.Error("nil InputSchema should be filled with default object")
	}
	if result.Tools[0].InputSchema["type"] != "object" {
		t.Errorf("expected default type=object, got %v", result.Tools[0].InputSchema["type"])
	}
}

// --- G8: provider error wrapping ---

type errorToolProvider struct{}

func (e *errorToolProvider) ListTools() []Tool { return nil }

func (e *errorToolProvider) CallTool(name string, arguments map[string]any) (*ToolCallResult, error) {
	return nil, fmt.Errorf("tool execution failed: connection timeout")
}

func TestServer_HandleToolsCall_ProviderError(t *testing.T) {
	server := NewServer(&errorToolProvider{})

	params, _ := json.Marshal(map[string]any{
		"name":      "failing_tool",
		"arguments": map[string]any{},
	})
	var rawParams any
	json.Unmarshal(params, &rawParams)

	req := JSONRPCRequest{JSONRPC: "2.0", ID: 10, Method: "tools/call", Params: rawParams}
	resp := server.HandleRequest(req)

	if resp.Error != nil {
		t.Fatalf("provider error should be wrapped in result, not JSON-RPC error: %v", resp.Error)
	}

	resultBytes, _ := json.Marshal(resp.Result)
	var result ToolCallResult
	if err := json.Unmarshal(resultBytes, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !result.IsError {
		t.Error("IsError should be true when provider returns error")
	}
	if len(result.Content) == 0 {
		t.Fatal("expected error content")
	}
	if result.Content[0].Text == "" {
		t.Error("error message should be in content text")
	}
}

// --- G7: concurrent HandleRequest ---

func TestServer_ConcurrentHandleRequests(t *testing.T) {
	server := NewServer(&mockToolProvider{
		tools: []Tool{{Name: "t1", Description: "Tool 1", InputSchema: map[string]any{"type": "object"}}},
	})

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	errs := make(chan error, goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			if id%2 == 0 {
				req := JSONRPCRequest{JSONRPC: "2.0", ID: id, Method: "tools/list"}
				resp := server.HandleRequest(req)
				if resp.Error != nil {
					errs <- fmt.Errorf("goroutine %d: %v", id, resp.Error)
				}
			} else {
				params, _ := json.Marshal(map[string]any{
					"name":      "t1",
					"arguments": map[string]any{},
				})
				var rawParams any
				json.Unmarshal(params, &rawParams)
				req := JSONRPCRequest{JSONRPC: "2.0", ID: id, Method: "tools/call", Params: rawParams}
				resp := server.HandleRequest(req)
				if resp.Error != nil {
					errs <- fmt.Errorf("goroutine %d: %v", id, resp.Error)
				}
			}
		}(i)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent request failed: %v", err)
	}
}
