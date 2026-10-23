package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"scabbage/internal/rate"
)

const (
	perSecond    = 5 // calls each token may make, after a burst
	burst        = 10
	maxBodyBytes = 1 << 20 // a full place call is about 200 KB
)

// Options configure a Server; the zero value uses time.Now and slog.Default.
type Options struct {
	Now    func() time.Time
	Logger *slog.Logger
}

// Server serves the tools over MCP and plain JSON, for whoever holds a token.
type Server struct {
	tokens *Tokens
	caller Caller
	now    func() time.Time
	log    *slog.Logger

	mu      sync.Mutex
	buckets map[string]*rate.Bucket // by player and label
}

// NewServer serves caller's tools to token holders.
func NewServer(tokens *Tokens, caller Caller, opts Options) *Server {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	return &Server{tokens: tokens, caller: caller, now: opts.Now, log: opts.Logger, buckets: map[string]*rate.Bucket{}}
}

// httpError is a refused request: its status and what to tell the agent.
type httpError struct {
	status int
	msg    string
}

func (e *httpError) Error() string { return e.msg }

// authorize finds the token's owner, checking they may use agents and aren't calling too fast.
func (s *Server) authorize(r *http.Request) (Owner, error) {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok {
		return Owner{}, &httpError{http.StatusUnauthorized, "send Authorization: Bearer <token>; get one in game with /agent link"}
	}
	o, err := s.tokens.Owner(strings.TrimSpace(token))
	if errors.Is(err, ErrBadToken) {
		return Owner{}, &httpError{http.StatusUnauthorized, err.Error() + "; get a new one in game with /agent link"}
	}
	if err != nil {
		return Owner{}, err
	}
	if !o.MayUseAgents() {
		return Owner{}, &httpError{http.StatusForbidden, "agents are off for " + o.Name + "; an admin can turn them on with /agent allow " + o.Name}
	}
	if !s.take(o) {
		return Owner{}, &httpError{http.StatusTooManyRequests, fmt.Sprintf("slow down: at most %d calls a second", perSecond)}
	}
	return o, nil
}

func (s *Server) take(o Owner) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := fmt.Sprint(o.PlayerID, "/", o.Label)
	b := s.buckets[key]
	if b == nil {
		b = rate.New(s.now(), perSecond, burst)
		s.buckets[key] = b
	}
	return b.Take(s.now())
}

// call runs a known tool, making sure its arguments are a JSON object.
func (s *Server) call(ctx context.Context, o Owner, tool string, args json.RawMessage) (any, error) {
	args = bytes.TrimSpace(args)
	if len(args) == 0 || bytes.Equal(args, []byte("null")) {
		args = json.RawMessage("{}")
	}
	if args[0] != '{' || !json.Valid(args) {
		return nil, Refusal("arguments must be a JSON object")
	}
	out, err := s.caller.Call(ctx, o, tool, args)
	var r Refusal
	if err == nil || errors.As(err, &r) {
		s.log.Info("agent call", "name", o.Name, "label", o.Label, "tool", tool, "refused", string(r))
	}
	return out, err
}

// refusal says why a tool failed, logging and hiding internal errors from the agent.
func (s *Server) refusal(tool string, err error) (string, bool) {
	var r Refusal
	if errors.As(err, &r) {
		return string(r), true
	}
	if errors.Is(err, context.Canceled) {
		return "the request was canceled", false // the agent hung up first
	}
	s.log.Error("agent tool failed", "tool", tool, "err", err)
	return "something went wrong in the game server", false
}

func readBody(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	if err != nil {
		return nil, &httpError{http.StatusRequestEntityTooLarge, "request too large"}
	}
	return b, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError sends a refused request's status, or 500 for anything unexpected.
func (s *Server) writeError(w http.ResponseWriter, err error) {
	var he *httpError
	if !errors.As(err, &he) {
		s.log.Error("agent request failed", "err", err)
		he = &httpError{http.StatusInternalServerError, "something went wrong in the game server"}
	}
	if he.status == http.StatusUnauthorized {
		w.Header().Set("WWW-Authenticate", `Bearer realm="scabbage"`)
	}
	writeJSON(w, he.status, map[string]string{"error": he.msg})
}
