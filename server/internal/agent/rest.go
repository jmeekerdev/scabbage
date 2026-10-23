package agent

import (
	"net/http"
	"strings"
)

// RESTPrefix is where the plain JSON tools live: POST RESTPrefix + tool name.
const RESTPrefix = "/api/agent/v1/"

// REST serves each tool as POST /api/agent/v1/{tool}, and the spec at openapi.json.
func (s *Server) REST() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, RESTPrefix)
		switch {
		case r.Method == http.MethodGet && name == "":
			writeJSON(w, http.StatusOK, guide(baseURL(r)))
			return
		case r.Method == http.MethodGet && name == "openapi.json":
			writeJSON(w, http.StatusOK, openAPI())
			return
		}
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "use POST"})
			return
		}
		s.serveTool(w, r, name)
	})
}

func (s *Server) serveTool(w http.ResponseWriter, r *http.Request, name string) {
	o, err := s.authorize(r)
	if err != nil {
		s.writeError(w, err)
		return
	}
	if _, ok := ToolNamed(name); !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no tool called " + name + "; GET " + RESTPrefix + " lists them"})
		return
	}
	body, err := readBody(w, r)
	if err != nil {
		s.writeError(w, err)
		return
	}
	out, err := s.call(r.Context(), o, name, body)
	if err != nil {
		msg, refused := s.refusal(name, err)
		status := http.StatusBadRequest
		if !refused {
			status = http.StatusInternalServerError
		}
		writeJSON(w, status, map[string]string{"error": msg})
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// guide explains the API to an agent that found it with a plain GET.
func guide(base string) map[string]any {
	tools := make([]map[string]any, 0, len(Tools))
	for _, t := range Tools {
		tools = append(tools, map[string]any{"name": t.Name, "description": t.Description, "arguments": t.Input})
	}
	return map[string]any{
		"about": InstructionsFor(base),
		"call":  "POST " + RESTPrefix + "{tool} with a JSON object of arguments and the header Authorization: Bearer <token>",
		"token": "the player types /agent link in game to get one",
		"spec":  RESTPrefix + "openapi.json",
		"mcp":   "POST /mcp speaks MCP (Streamable HTTP) with the same tools",
		"tools": tools,
	}
}

// openAPI describes the plain JSON tools, made from the same table MCP lists.
func openAPI() map[string]any {
	errorBody := schema{"type": "object", "properties": schema{"error": schema{"type": "string"}}}
	paths := schema{}
	for _, t := range Tools {
		paths[RESTPrefix+t.Name] = schema{"post": schema{
			"operationId": t.Name,
			"summary":     t.Description,
			"requestBody": schema{"content": schema{"application/json": schema{"schema": t.Input}}},
			"responses": schema{
				"200":     schema{"description": "the result", "content": schema{"application/json": schema{"schema": schema{"type": "object"}}}},
				"default": schema{"description": "why it was refused", "content": schema{"application/json": schema{"schema": errorBody}}},
			},
		}}
	}
	return schema{
		"openapi":    "3.1.0",
		"info":       schema{"title": "Scabbage agent API", "version": "1", "description": Instructions},
		"paths":      paths,
		"security":   []schema{{"token": []string{}}},
		"components": schema{"securitySchemes": schema{"token": schema{"type": "http", "scheme": "bearer"}}},
	}
}
