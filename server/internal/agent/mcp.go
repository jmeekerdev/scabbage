package agent

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
)

// mcpVersions are the MCP protocol versions spoken, newest first.
var mcpVersions = []string{"2025-06-18", "2025-03-26", "2024-11-05"}

// JSON-RPC error codes.
const (
	parseError     = -32700
	invalidRequest = -32600
	unknownMethod  = -32601
	invalidParams  = -32602
	internalError  = -32603
)

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

// MCP serves the tools over MCP's Streamable HTTP transport, answering each POST with JSON.
func (s *Server) MCP() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			w.WriteHeader(http.StatusMethodNotAllowed) // no server-sent event stream
			return
		}
		o, err := s.authorize(r)
		if err != nil {
			s.writeError(w, err)
			return
		}
		body, err := readBody(w, r)
		if err != nil {
			s.writeError(w, err)
			return
		}
		var req rpcRequest
		if err := json.Unmarshal(body, &req); err != nil {
			writeJSON(w, http.StatusOK, failed(nil, parseError, "not a JSON-RPC request"))
			return
		}
		if req.ID == nil { // a notification wants no answer
			w.WriteHeader(http.StatusAccepted)
			return
		}
		writeJSON(w, http.StatusOK, s.answer(r, o, req))
	})
}

func (s *Server) answer(r *http.Request, o Owner, req rpcRequest) rpcResponse {
	if req.JSONRPC != "2.0" {
		return failed(req.ID, invalidRequest, "jsonrpc must be 2.0")
	}
	switch req.Method {
	case "initialize":
		return done(req.ID, initialized(req.Params, baseURL(r)))
	case "ping":
		return done(req.ID, struct{}{})
	case "tools/list":
		return done(req.ID, map[string]any{"tools": toolList()})
	case "tools/call":
		return s.callTool(r, o, req)
	}
	return failed(req.ID, unknownMethod, "no method "+req.Method)
}

func initialized(params json.RawMessage, base string) map[string]any {
	var p struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	_ = json.Unmarshal(params, &p)
	version := mcpVersions[0]
	if slices.Contains(mcpVersions, p.ProtocolVersion) {
		version = p.ProtocolVersion
	}
	return map[string]any{
		"protocolVersion": version,
		"capabilities":    map[string]any{"tools": map[string]any{}},
		"serverInfo":      map[string]string{"name": "scabbage", "version": "1"},
		"instructions":    InstructionsFor(base),
	}
}

func toolList() []map[string]any {
	out := make([]map[string]any, 0, len(Tools))
	for _, t := range Tools {
		out = append(out, map[string]any{"name": t.Name, "description": t.Description, "inputSchema": t.Input})
	}
	return out
}

func (s *Server) callTool(r *http.Request, o Owner, req rpcRequest) rpcResponse {
	var p struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &p); err != nil {
		return failed(req.ID, invalidParams, "params need a tool name and arguments")
	}
	if _, ok := ToolNamed(p.Name); !ok {
		return failed(req.ID, invalidParams, "no tool called "+p.Name)
	}
	out, err := s.call(r.Context(), o, p.Name, p.Arguments)
	if err != nil {
		msg, _ := s.refusal(p.Name, err)
		return done(req.ID, map[string]any{"content": []map[string]string{{"type": "text", "text": msg}}, "isError": true})
	}
	text, err := json.Marshal(out)
	if err != nil {
		return failed(req.ID, internalError, err.Error())
	}
	return done(req.ID, map[string]any{
		"content":           []map[string]string{{"type": "text", "text": strings.TrimSpace(string(text))}},
		"structuredContent": out,
	})
}

func done(id json.RawMessage, result any) rpcResponse {
	return rpcResponse{JSONRPC: "2.0", ID: id, Result: result}
}

func failed(id json.RawMessage, code int, msg string) rpcResponse {
	if id == nil {
		id = json.RawMessage("null")
	}
	return rpcResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{code, msg}}
}
