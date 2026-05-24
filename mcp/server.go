package mcp

import "encoding/json"

// ToolProvider supplies tools and executes tool calls.
// Implemented by runtime to bridge with the skill registry.
type ToolProvider interface {
	// ListTools returns all available tools.
	ListTools() []Tool

	// CallTool executes a tool by name with the given arguments.
	CallTool(name string, arguments map[string]any) (*ToolCallResult, error)
}

// Server implements the MCP JSON-RPC method dispatcher.
type Server struct {
	provider ToolProvider
}

// NewServer creates a new MCP server with the given tool provider.
func NewServer(provider ToolProvider) *Server {
	return &Server{provider: provider}
}

// HandleRequest dispatches a JSON-RPC request to the appropriate handler.
func (s *Server) HandleRequest(req JSONRPCRequest) JSONRPCResponse {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(req)
	default:
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &JSONRPCError{Code: ErrCodeMethodNotFound, Message: "method not found"},
		}
	}
}

func (s *Server) handleInitialize(req JSONRPCRequest) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    "openbotstack",
				"version": "1.0.0",
			},
		},
	}
}

func (s *Server) handleToolsList(req JSONRPCRequest) JSONRPCResponse {
	tools := s.provider.ListTools()
	if tools == nil {
		tools = []Tool{}
	}
	// Ensure all tools have non-nil InputSchema per MCP spec
	for i := range tools {
		if tools[i].InputSchema == nil {
			tools[i].InputSchema = map[string]any{"type": "object"}
		}
	}
	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  ToolListResult{Tools: tools},
	}
}

func (s *Server) handleToolsCall(req JSONRPCRequest) JSONRPCResponse {
	// Extract params
	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &JSONRPCError{Code: ErrCodeInvalidParams, Message: "invalid params"},
		}
	}

	var params ToolCallParams
	if err := json.Unmarshal(paramsBytes, &params); err != nil {
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &JSONRPCError{Code: ErrCodeInvalidParams, Message: "invalid params format"},
		}
	}

	if params.Name == "" {
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &JSONRPCError{Code: ErrCodeInvalidParams, Message: "tool name is required"},
		}
	}

	args := params.Arguments
	if args == nil {
		args = map[string]any{}
	}

	result, err := s.provider.CallTool(params.Name, args)
	if err != nil {
		return JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: ToolCallResult{
				Content: []ContentItem{{Type: "text", Text: err.Error()}},
				IsError: true,
			},
		}
	}

	return JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

// HandleNotification handles JSON-RPC notifications.
// Per JSON-RPC spec, notifications produce no response.
func (s *Server) HandleNotification(notif JSONRPCNotification) {
	// No side effects needed for current MCP methods.
	// Reserved for future notifications/cancelled etc.
}
