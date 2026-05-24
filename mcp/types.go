// Package mcp defines types for the Model Context Protocol (MCP).
//
// MCP is a JSON-RPC 2.0 based protocol that allows AI models to discover
// and invoke tools. This package provides the core type definitions used
// by both the server (runtime) and client integrations.
//
// The package contains two sets of types:
//   - Server-side types (JSONRPCRequest, Tool, ToolCallResult, etc.) used by
//     the MCP server implementation to handle incoming JSON-RPC requests.
//   - Client-side types (ServerConfig, ClientTool, Client interface, etc.)
//     used by the MCP client to connect to external MCP servers and discover
//     their tools.
package mcp

import (
	skills "github.com/openbotstack/openbotstack-core/control/skills"
)

// --- JSON-RPC 2.0 types ---

// JSONRPCRequest represents a JSON-RPC 2.0 request.
type JSONRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// JSONRPCNotification represents a JSON-RPC 2.0 notification (no ID).
type JSONRPCNotification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string       `json:"jsonrpc"`
	ID      any          `json:"id"`
	Result  any          `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error.
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// Standard JSON-RPC error codes.
const (
	ErrCodeParseError     = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternalError  = -32603
)

// --- MCP Tool types ---

// Tool describes a tool available through MCP.
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"inputSchema"`
}

// ToolListResult is the result of a tools/list request.
type ToolListResult struct {
	Tools       []Tool `json:"tools"`
	NextCursor  string `json:"nextCursor,omitempty"`
}

// ToolCallParams contains the parameters for a tools/call request.
type ToolCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

// ToolCallResult is the result of a tools/call request.
type ToolCallResult struct {
	Content []ContentItem `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

// ContentItem represents a content item in a tool call result.
type ContentItem struct {
	Type        string      `json:"type"`
	Text        string      `json:"text,omitempty"`
	Data        string      `json:"data,omitempty"`
	MimeType    string      `json:"mimeType,omitempty"`
	URI         string      `json:"uri,omitempty"`
	Annotations *Annotation `json:"annotations,omitempty"`
}

// Annotation provides metadata about a content item.
type Annotation struct {
	Audience []string `json:"audience,omitempty"`
	Priority float64  `json:"priority,omitempty"`
}

// --- MCP Client types ---
// These types are used by the MCP client to connect to external MCP servers.

// ServerAuth describes authentication for an MCP server connection.
type ServerAuth struct {
	Type    string            `json:"type"`               // "bearer" | "api_key" | "custom" | "none"
	Token   string            `json:"token,omitempty"`     // bearer: the token; api_key: the key value
	Header  string            `json:"header,omitempty"`    // api_key: custom header name (default: X-API-Key)
	Headers map[string]string `json:"headers,omitempty"`   // custom: arbitrary headers for HTTP transports
	EnvAuth map[string]string `json:"env_auth,omitempty"` // env vars for stdio transport
}

// ServerConfig describes an MCP server connection.
type ServerConfig struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Transport string            `json:"transport"` // "stdio" | "sse"
	Command   string            `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	URL       string            `json:"url,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	Auth      *ServerAuth       `json:"auth,omitempty"`
	Enabled   bool              `json:"enabled"`
}

// ClientTool describes an MCP tool discovered from a server.
// Uses skills.JSONSchema for InputSchema to integrate with the planner.
type ClientTool struct {
	Name        string             `json:"name"`
	Description string             `json:"description,omitempty"`
	InputSchema *skills.JSONSchema `json:"input_schema,omitempty"`
}

// CallToolResult is the result of calling an MCP tool via the client.
type CallToolResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"is_error,omitempty"`
}

// ContentBlock represents a piece of content in an MCP client response.
type ContentBlock struct {
	Type string `json:"type"`           // "text" | "image" | "resource"
	Text string `json:"text,omitempty"`
}

// ServerStatus describes the current state of an MCP server.
type ServerStatus struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Transport string `json:"transport"`
	Status    string `json:"status"` // "connected" | "disconnected" | "error"
	ToolCount int    `json:"tool_count"`
	Error     string `json:"error,omitempty"`
}

// HTTPHeaders returns the HTTP headers derived from the auth config.
// Returns nil if auth is nil or type is "none".
func (a *ServerAuth) HTTPHeaders() map[string]string {
	if a == nil || a.Type == "none" || a.Type == "" {
		return nil
	}
	headers := make(map[string]string)
	switch a.Type {
	case "bearer":
		if a.Token != "" {
			headers["Authorization"] = "Bearer " + a.Token
		}
	case "api_key":
		h := a.Header
		if h == "" {
			h = "X-API-Key"
		}
		if a.Token != "" {
			headers[h] = a.Token
		}
	case "custom":
		for k, v := range a.Headers {
			headers[k] = v
		}
	}
	return headers
}

// EnvVars returns the environment variables derived from the auth config.
// Returns nil if auth is nil or has no env_auth.
func (a *ServerAuth) EnvVars() map[string]string {
	if a == nil {
		return nil
	}
	return a.EnvAuth
}
