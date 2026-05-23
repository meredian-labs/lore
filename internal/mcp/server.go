// Package mcp implements a Model Context Protocol server over stdio.
// Protocol: JSON-RPC 2.0, newline-delimited, as specified at https://modelcontextprotocol.io
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Server is a minimal MCP server that handles one session over stdio.
type Server struct {
	name    string
	version string
	tools   []toolDef
}

type toolDef struct {
	Name        string
	Description string
	InputSchema map[string]interface{}
	Handler     func(ctx context.Context, args map[string]interface{}) (string, error)
}

// New returns a new Server.
func New(name, version string) *Server {
	return &Server{name: name, version: version}
}

// Register adds a tool to the server.
func (s *Server) Register(
	name, description string,
	schema map[string]interface{},
	handler func(ctx context.Context, args map[string]interface{}) (string, error),
) {
	s.tools = append(s.tools, toolDef{
		Name:        name,
		Description: description,
		InputSchema: schema,
		Handler:     handler,
	})
}

// Serve runs the JSON-RPC message loop until r is closed.
func (s *Server) Serve(ctx context.Context, r io.Reader, w io.Writer) error {
	enc := json.NewEncoder(w)
	scanner := bufio.NewScanner(r)
	// Allow lines up to 4MB (agent recaps can be large).
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req rpcMessage
		if err := json.Unmarshal(line, &req); err != nil {
			continue // ignore malformed messages
		}

		resp := s.dispatch(ctx, &req)
		if resp == nil {
			continue // notification — no response
		}
		if err := enc.Encode(resp); err != nil {
			fmt.Fprintf(os.Stderr, "lore mcp: encode: %v\n", err)
		}
	}
	return scanner.Err()
}

// dispatch routes a single JSON-RPC message and returns the response, or nil
// for notifications (messages without an id).
func (s *Server) dispatch(ctx context.Context, req *rpcMessage) *rpcMessage {
	// Notifications have no id — no response expected.
	if req.ID == nil && req.Method != "ping" {
		return nil
	}

	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "ping":
		return &rpcMessage{JSONRPC: "2.0", ID: req.ID, Result: map[string]interface{}{}}
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(ctx, req)
	default:
		return errorResponse(req.ID, -32601, "method not found: "+req.Method)
	}
}

func (s *Server) handleInitialize(req *rpcMessage) *rpcMessage {
	return &rpcMessage{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    s.name,
				"version": s.version,
			},
		},
	}
}

func (s *Server) handleToolsList(req *rpcMessage) *rpcMessage {
	tools := make([]interface{}, len(s.tools))
	for i, t := range s.tools {
		schema := t.InputSchema
		if schema == nil {
			schema = map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
		}
		tools[i] = map[string]interface{}{
			"name":        t.Name,
			"description": t.Description,
			"inputSchema": schema,
		}
	}
	return &rpcMessage{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  map[string]interface{}{"tools": tools},
	}
}

func (s *Server) handleToolsCall(ctx context.Context, req *rpcMessage) *rpcMessage {
	var params struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}
	if req.Params != nil {
		json.Unmarshal(req.Params, &params)
	}
	if params.Arguments == nil {
		params.Arguments = map[string]interface{}{}
	}

	for _, t := range s.tools {
		if t.Name != params.Name {
			continue
		}
		text, err := t.Handler(ctx, params.Arguments)
		if err != nil {
			return &rpcMessage{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: map[string]interface{}{
					"content": []interface{}{map[string]interface{}{"type": "text", "text": "error: " + err.Error()}},
					"isError": true,
				},
			}
		}
		return &rpcMessage{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"content": []interface{}{map[string]interface{}{"type": "text", "text": text}},
			},
		}
	}

	return errorResponse(req.ID, -32602, "unknown tool: "+params.Name)
}

// rpcMessage is a combined JSON-RPC 2.0 request/response/notification.
type rpcMessage struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Method  string           `json:"method,omitempty"`
	Params  json.RawMessage  `json:"params,omitempty"`
	Result  interface{}      `json:"result,omitempty"`
	Error   *rpcError        `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func errorResponse(id *json.RawMessage, code int, msg string) *rpcMessage {
	return &rpcMessage{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &rpcError{Code: code, Message: msg},
	}
}
