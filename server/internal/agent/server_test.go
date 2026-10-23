package agent

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// fakeCaller answers every tool with its name and arguments, or refuses "goto".
type fakeCaller struct{ calls []string }

func (f *fakeCaller) Call(_ context.Context, o Owner, tool string, args json.RawMessage) (any, error) {
	f.calls = append(f.calls, o.Name+" "+tool+" "+string(args))
	switch tool {
	case "goto":
		return nil, Refusal("open Scabbage in a browser first")
	case "undo":
		return nil, errors.New("database on fire")
	}
	return map[string]any{"tool": tool}, nil
}

type rig struct {
	srv    *Server
	caller *fakeCaller
	admin  string // Ada's token
	kid    string // Kid's token; agents off
	now    time.Time
}

func newRig(t *testing.T) *rig {
	t.Helper()
	tokens, _ := newTokens(t)
	r := &rig{caller: &fakeCaller{}, now: time.Unix(1_700_000_000, 0)}
	r.srv = NewServer(tokens, r.caller, Options{Now: func() time.Time { return r.now }, Logger: slog.New(slog.DiscardHandler)})
	r.admin, _ = tokens.Create(1, "laptop")
	r.kid, _ = tokens.Create(2, "tablet")
	return r
}

func (r *rig) post(h http.Handler, path, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func TestRequestsNeedAWorkingToken(t *testing.T) {
	r := newRig(t)
	cases := []struct {
		token string
		want  int
	}{{"", 401}, {"scab_wrong", 401}, {r.kid, 403}, {r.admin, 200}}
	for _, c := range cases {
		if w := r.post(r.srv.REST(), "/api/agent/v1/whereami", c.token, `{}`); w.Code != c.want {
			t.Errorf("token %q: status %d, want %d (%s)", c.token, w.Code, c.want, w.Body)
		}
	}
	if w := r.post(r.srv.REST(), "/api/agent/v1/whereami", "", `{}`); w.Header().Get("WWW-Authenticate") == "" {
		t.Error("401 without WWW-Authenticate")
	}
}

func TestEachTokenIsRateLimited(t *testing.T) {
	r := newRig(t)
	limited := 0
	for range 20 {
		if r.post(r.srv.REST(), "/api/agent/v1/blocks", r.admin, ``).Code == http.StatusTooManyRequests {
			limited++
		}
	}
	if limited == 0 || limited > 15 {
		t.Fatalf("%d of 20 calls at once were limited", limited)
	}
	r.now = r.now.Add(time.Second)
	if w := r.post(r.srv.REST(), "/api/agent/v1/blocks", r.admin, ``); w.Code != 200 {
		t.Fatalf("status %d after waiting", w.Code)
	}
}

func TestRESTCallsTheTool(t *testing.T) {
	r := newRig(t)
	w := r.post(r.srv.REST(), "/api/agent/v1/fill", r.admin, `{"block":"planks"}`)
	if w.Code != 200 || strings.TrimSpace(w.Body.String()) != `{"tool":"fill"}` {
		t.Fatalf("got %d %s", w.Code, w.Body)
	}
	if r.caller.calls[0] != `Ada fill {"block":"planks"}` {
		t.Fatalf("call = %q", r.caller.calls[0])
	}
	cases := []struct {
		path, body string
		want       int
		says       string
	}{
		{"/api/agent/v1/nope", `{}`, 404, "no tool"},
		{"/api/agent/v1/goto", `{}`, 400, "open Scabbage"},
		{"/api/agent/v1/undo", `{}`, 500, "went wrong"},
		{"/api/agent/v1/fill", `[1]`, 400, "JSON object"},
	}
	for _, c := range cases {
		w := r.post(r.srv.REST(), c.path, r.admin, c.body)
		if w.Code != c.want || !strings.Contains(w.Body.String(), c.says) {
			t.Errorf("%s: got %d %s, want %d saying %q", c.path, w.Code, w.Body, c.want, c.says)
		}
	}
}

func TestAPlainGetExplainsTheAPI(t *testing.T) {
	r := newRig(t)
	w := httptest.NewRecorder()
	r.srv.REST().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/agent/v1/", nil))
	var g struct {
		About string           `json:"about"`
		Tools []map[string]any `json:"tools"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &g); err != nil || !strings.Contains(g.About, "air") ||
		!strings.Contains(g.About, "MCP at http://example.com/mcp") || len(g.Tools) != len(Tools) {
		t.Fatalf("guide: %v %s", err, w.Body)
	}
	if w := r.post(r.srv.REST(), "/api/agent/v1/build", r.admin, `{}`); !strings.Contains(w.Body.String(), "GET /api/agent/v1/") {
		t.Fatalf("404 gives no hint: %s", w.Body)
	}
}

func TestOpenAPIListsEveryTool(t *testing.T) {
	r := newRig(t)
	w := httptest.NewRecorder()
	r.srv.REST().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/agent/v1/openapi.json", nil))
	var spec struct {
		OpenAPI string                    `json:"openapi"`
		Paths   map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &spec); err != nil || spec.OpenAPI == "" {
		t.Fatalf("spec: %v %s", err, w.Body)
	}
	for _, tool := range Tools {
		if spec.Paths["/api/agent/v1/"+tool.Name]["post"] == nil {
			t.Errorf("spec misses %s", tool.Name)
		}
	}
}

func rpc(t *testing.T, r *rig, body string) map[string]any {
	t.Helper()
	w := r.post(r.srv.MCP(), "/mcp", r.admin, body)
	b, _ := io.ReadAll(w.Body)
	if w.Code != 200 {
		t.Fatalf("status %d: %s", w.Code, b)
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("%v: %s", err, b)
	}
	return out
}

func TestMCPInitializeAndListTools(t *testing.T) {
	r := newRig(t)
	init := rpc(t, r, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26"}}`)
	res := init["result"].(map[string]any)
	if init["id"] != 1.0 || res["protocolVersion"] != "2025-03-26" ||
		!strings.Contains(res["instructions"].(string), "http://example.com/api/agent/v1/place in batches") {
		t.Fatalf("initialize = %v", init)
	}
	if res["capabilities"].(map[string]any)["tools"] == nil {
		t.Fatal("no tools capability")
	}
	if w := r.post(r.srv.MCP(), "/mcp", r.admin, `{"jsonrpc":"2.0","method":"notifications/initialized"}`); w.Code != http.StatusAccepted {
		t.Fatalf("notification status %d", w.Code)
	}
	list := rpc(t, r, `{"jsonrpc":"2.0","id":"a","method":"tools/list"}`)
	tools := list["result"].(map[string]any)["tools"].([]any)
	if len(tools) != len(Tools) {
		t.Fatalf("%d tools, want %d", len(tools), len(Tools))
	}
	first := tools[0].(map[string]any)
	if first["name"] != "whereami" || first["inputSchema"].(map[string]any)["type"] != "object" {
		t.Fatalf("first tool = %v", first)
	}
}

func TestMCPCallsTools(t *testing.T) {
	r := newRig(t)
	ok := rpc(t, r, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"look","arguments":{"from":{"x":1}}}}`)
	res := ok["result"].(map[string]any)
	if res["isError"] == true || res["structuredContent"].(map[string]any)["tool"] != "look" {
		t.Fatalf("call = %v", ok)
	}
	if text := res["content"].([]any)[0].(map[string]any)["text"]; text != `{"tool":"look"}` {
		t.Fatalf("text = %v", text)
	}
	refused := rpc(t, r, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"goto","arguments":{}}}`)
	res = refused["result"].(map[string]any)
	if res["isError"] != true || !strings.Contains(res["content"].([]any)[0].(map[string]any)["text"].(string), "browser") {
		t.Fatalf("refusal = %v", refused)
	}
	unknown := rpc(t, r, `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"fly"}}`)
	if unknown["error"].(map[string]any)["code"] != -32602.0 {
		t.Fatalf("unknown tool = %v", unknown)
	}
	method := rpc(t, r, `{"jsonrpc":"2.0","id":5,"method":"resources/list"}`)
	if method["error"].(map[string]any)["code"] != -32601.0 {
		t.Fatalf("unknown method = %v", method)
	}
	if rpc(t, r, `{"jsonrpc":"2.0","id":6,"method":"ping"}`)["result"] == nil {
		t.Fatal("ping has no result")
	}
}

func TestMCPRefusesGetsAndGarbage(t *testing.T) {
	r := newRig(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer "+r.admin)
	r.srv.MCP().ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET status %d", w.Code)
	}
	if bad := rpc(t, r, `{nope`); bad["error"].(map[string]any)["code"] != -32700.0 {
		t.Fatalf("parse error = %v", bad)
	}
}
