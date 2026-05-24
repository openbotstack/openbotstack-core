package mcp

import (
	"encoding/json"
	"testing"
)

// --- JSON-RPC types ---

func TestJSONRPCRequest_Marshal(t *testing.T) {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      float64(1),
		Method:  "tools/list",
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	expected := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
	if string(data) != expected {
		t.Errorf("expected %s, got %s", expected, string(data))
	}
}

func TestJSONRPCRequest_WithParams(t *testing.T) {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      "abc",
		Method:  "tools/call",
		Params:  map[string]any{"name": "search", "arguments": map[string]any{"q": "test"}},
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded["method"] != "tools/call" {
		t.Errorf("expected tools/call, got %v", decoded["method"])
	}
}

func TestJSONRPCResponse_Success(t *testing.T) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      float64(1),
		Result:  map[string]any{"tools": []any{}},
	}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if _, hasError := decoded["error"]; hasError {
		t.Error("success response should not have error field")
	}
}

func TestJSONRPCResponse_Error(t *testing.T) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      float64(1),
		Error: &JSONRPCError{
			Code:    -32600,
			Message: "Invalid Request",
		},
	}
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if _, hasResult := decoded["result"]; hasResult {
		t.Error("error response should not have result field")
	}
}

func TestJSONRPCError_StandardCodes(t *testing.T) {
	if ErrCodeParseError != -32700 {
		t.Errorf("ParseError code should be -32700, got %d", ErrCodeParseError)
	}
	if ErrCodeInvalidRequest != -32600 {
		t.Errorf("InvalidRequest code should be -32600, got %d", ErrCodeInvalidRequest)
	}
	if ErrCodeMethodNotFound != -32601 {
		t.Errorf("MethodNotFound code should be -32601, got %d", ErrCodeMethodNotFound)
	}
	if ErrCodeInvalidParams != -32602 {
		t.Errorf("InvalidParams code should be -32602, got %d", ErrCodeInvalidParams)
	}
	if ErrCodeInternalError != -32603 {
		t.Errorf("InternalError code should be -32603, got %d", ErrCodeInternalError)
	}
}

// --- MCP Tool types ---

func TestTool_Marshal(t *testing.T) {
	tool := Tool{
		Name:        "search",
		Description: "Search the web",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{"query": map[string]any{"type": "string"}},
			"required":   []string{"query"},
		},
	}
	data, err := json.Marshal(tool)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded["name"] != "search" {
		t.Errorf("expected name=search, got %v", decoded["name"])
	}
}

func TestToolListResult_Marshal(t *testing.T) {
	result := ToolListResult{
		Tools: []Tool{
			{Name: "search", Description: "Search"},
			{Name: "calc", Description: "Calculate"},
		},
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	tools := decoded["tools"].([]any)
	if len(tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(tools))
	}
}

func TestToolCallParams_Parse(t *testing.T) {
	raw := `{"name":"search","arguments":{"query":"golang"}}`
	var params ToolCallParams
	if err := json.Unmarshal([]byte(raw), &params); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if params.Name != "search" {
		t.Errorf("expected name=search, got %q", params.Name)
	}
	if params.Arguments["query"] != "golang" {
		t.Errorf("expected query=golang, got %v", params.Arguments["query"])
	}
}

func TestToolCallResult_TextContent(t *testing.T) {
	result := ToolCallResult{
		Content: []ContentItem{
			{Type: "text", Text: "Found 3 results"},
		},
		IsError: false,
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	expected := `{"content":[{"type":"text","text":"Found 3 results"}]}`
	if string(data) != expected {
		t.Errorf("expected %s, got %s", expected, string(data))
	}
}

func TestToolCallResult_ErrorContent(t *testing.T) {
	result := ToolCallResult{
		Content: []ContentItem{
			{Type: "text", Text: "Tool execution failed: timeout"},
		},
		IsError: true,
	}
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded["isError"] != true {
		t.Error("expected isError=true")
	}
}

// --- Notification types ---

func TestNotification_ToolsListChanged(t *testing.T) {
	notif := JSONRPCNotification{
		JSONRPC: "2.0",
		Method:  "notifications/tools/list_changed",
	}
	data, err := json.Marshal(notif)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	expected := `{"jsonrpc":"2.0","method":"notifications/tools/list_changed"}`
	if string(data) != expected {
		t.Errorf("expected %s, got %s", expected, string(data))
	}
}
func TestToolCallParams_NilArguments(t *testing.T) {
	raw := `{"name":"noop"}`
	var params ToolCallParams
	if err := json.Unmarshal([]byte(raw), &params); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if params.Name != "noop" {
		t.Errorf("expected name=noop, got %q", params.Name)
	}
	if params.Arguments != nil {
		t.Error("expected nil arguments")
	}
}

func TestToolCallParams_InvalidArguments(t *testing.T) {
	raw := `{"name":"test","arguments":"not_an_object"}`
	var params ToolCallParams
	if err := json.Unmarshal([]byte(raw), &params); err == nil {
		t.Error("non-object arguments should fail unmarshal")
	}
}

func TestContentItem_Annotations(t *testing.T) {
	cb := ContentItem{
		Type: "text",
		Text: "result",
		Annotations: &Annotation{
			Audience: []string{"user"},
			Priority: 0.5,
		},
	}
	data, err := json.Marshal(cb)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	annotations, ok := decoded["annotations"].(map[string]any)
	if !ok {
		t.Fatal("expected annotations object")
	}
	if annotations["priority"] != 0.5 {
		t.Errorf("expected priority 0.5, got %v", annotations["priority"])
	}
}

// --- MCP Client types ---

func TestServerConfig_Fields(t *testing.T) {
	cfg := ServerConfig{
		ID:        "test-server",
		Name:      "Test Server",
		Transport: "stdio",
		Command:   "node",
		Args:      []string{"server.js"},
		Enabled:   true,
	}
	if cfg.ID != "test-server" {
		t.Errorf("ID = %q", cfg.ID)
	}
	if cfg.Transport != "stdio" {
		t.Errorf("Transport = %q", cfg.Transport)
	}
}

func TestClientTool_Fields(t *testing.T) {
	tool := ClientTool{
		Name:        "search",
		Description: "Search for items",
	}
	if tool.Name != "search" {
		t.Errorf("Name = %q", tool.Name)
	}
}

func TestCallToolResult_IsError(t *testing.T) {
	result := &CallToolResult{
		Content: []ContentBlock{{Type: "text", Text: "error: not found"}},
		IsError: true,
	}
	if !result.IsError {
		t.Error("expected IsError = true")
	}
	if len(result.Content) != 1 {
		t.Errorf("Content len = %d, want 1", len(result.Content))
	}
}

func TestServerStatus_Fields(t *testing.T) {
	status := ServerStatus{
		ID:        "srv1",
		Name:      "My Server",
		Transport: "sse",
		Status:    "connected",
		ToolCount: 5,
	}
	if status.ID != "srv1" {
		t.Errorf("ID = %q", status.ID)
	}
	if status.Status != "connected" {
		t.Errorf("Status = %q", status.Status)
	}
	if status.ToolCount != 5 {
		t.Errorf("ToolCount = %d, want 5", status.ToolCount)
	}
}

func TestContentBlock_Fields(t *testing.T) {
	cb := ContentBlock{
		Type: "text",
		Text: "hello world",
	}
	if cb.Type != "text" {
		t.Errorf("Type = %q", cb.Type)
	}
	if cb.Text != "hello world" {
		t.Errorf("Text = %q", cb.Text)
	}
}

func TestServerConfig_AllFields(t *testing.T) {
	cfg := ServerConfig{
		ID:        "sse-server",
		Name:      "SSE Server",
		Transport: "sse",
		URL:       "http://localhost:3000/mcp",
		Env:       map[string]string{"API_KEY": "secret"},
		Enabled:   true,
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded["transport"] != "sse" {
		t.Errorf("expected transport=sse, got %v", decoded["transport"])
	}
	if decoded["url"] != "http://localhost:3000/mcp" {
		t.Errorf("expected url, got %v", decoded["url"])
	}
}

// --- ServerAuth tests ---

func TestServerAuth_HTTPHeaders_Bearer(t *testing.T) {
	auth := &ServerAuth{Type: "bearer", Token: "mytoken"}
	headers := auth.HTTPHeaders()
	if headers["Authorization"] != "Bearer mytoken" {
		t.Errorf("expected Bearer mytoken, got %q", headers["Authorization"])
	}
}

func TestServerAuth_HTTPHeaders_APIKey(t *testing.T) {
	auth := &ServerAuth{Type: "api_key", Token: "key123"}
	headers := auth.HTTPHeaders()
	if headers["X-API-Key"] != "key123" {
		t.Errorf("expected X-API-Key=key123, got %v", headers)
	}
}

func TestServerAuth_HTTPHeaders_APIKey_CustomHeader(t *testing.T) {
	auth := &ServerAuth{Type: "api_key", Token: "key123", Header: "X-Custom-Auth"}
	headers := auth.HTTPHeaders()
	if headers["X-Custom-Auth"] != "key123" {
		t.Errorf("expected X-Custom-Auth=key123, got %v", headers)
	}
}

func TestServerAuth_HTTPHeaders_Custom(t *testing.T) {
	auth := &ServerAuth{
		Type:    "custom",
		Headers: map[string]string{"X-Auth": "val1", "X-Extra": "val2"},
	}
	headers := auth.HTTPHeaders()
	if headers["X-Auth"] != "val1" || headers["X-Extra"] != "val2" {
		t.Errorf("expected custom headers, got %v", headers)
	}
}

func TestServerAuth_HTTPHeaders_None(t *testing.T) {
	auth := &ServerAuth{Type: "none"}
	if headers := auth.HTTPHeaders(); headers != nil {
		t.Errorf("expected nil headers for none auth, got %v", headers)
	}
}

func TestServerAuth_HTTPHeaders_Nil(t *testing.T) {
	var auth *ServerAuth
	if headers := auth.HTTPHeaders(); headers != nil {
		t.Errorf("expected nil headers for nil auth, got %v", headers)
	}
}

func TestServerAuth_EnvVars(t *testing.T) {
	auth := &ServerAuth{
		Type:    "bearer",
		EnvAuth: map[string]string{"MCP_TOKEN": "secret", "MCP_USER": "admin"},
	}
	env := auth.EnvVars()
	if env["MCP_TOKEN"] != "secret" {
		t.Errorf("expected MCP_TOKEN=secret, got %v", env)
	}
}

func TestServerAuth_EnvVars_Nil(t *testing.T) {
	var auth *ServerAuth
	if env := auth.EnvVars(); env != nil {
		t.Errorf("expected nil for nil auth, got %v", env)
	}
}

func TestServerConfig_WithAuth(t *testing.T) {
	cfg := ServerConfig{
		ID:        "auth-server",
		Name:      "Auth Server",
		Transport: "sse",
		URL:       "http://localhost:3000/mcp",
		Auth:      &ServerAuth{Type: "bearer", Token: "tok123"},
		Enabled:   true,
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var decoded ServerConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.Auth == nil {
		t.Fatal("expected auth to be non-nil")
	}
	if decoded.Auth.Type != "bearer" {
		t.Errorf("expected auth type=bearer, got %q", decoded.Auth.Type)
	}
	if decoded.Auth.Token != "tok123" {
		t.Errorf("expected token=tok123, got %q", decoded.Auth.Token)
	}
}

func TestServerConfig_NoAuth(t *testing.T) {
	cfg := ServerConfig{
		ID:        "noauth",
		Transport: "stdio",
		Command:   "node",
		Enabled:   true,
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var decoded ServerConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.Auth != nil {
		t.Error("expected nil auth for config without auth")
	}
}
