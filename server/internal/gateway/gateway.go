// Package gateway accepts WebSocket connections, admits devices through the access flow and routes messages.
package gateway

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"

	"scabbage/internal/access"
	"scabbage/internal/sim"
	"scabbage/internal/world"
)

// Config configures the gateway handler.
type Config struct {
	Hub            *sim.Hub
	Access         *access.Service
	World          *world.World
	ServerName     string
	HelloTimeout   time.Duration
	OriginPatterns []string // extra allowed Origin hosts, e.g. the Vite dev server
	Logger         *slog.Logger
}

// Gateway is an http.Handler for the game WebSocket endpoint.
type Gateway struct {
	cfg     Config
	mu      sync.Mutex
	admins  map[*session]struct{}
	players map[*session]*avatar // in-world sessions
	entity  uint32               // last entity id handed out
	editMu  sync.Mutex           // makes each edit's check, change and broadcast atomic
}

// New returns a Gateway with defaults applied.
func New(cfg Config) *Gateway {
	if cfg.HelloTimeout == 0 {
		cfg.HelloTimeout = 10 * time.Second
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	return &Gateway{cfg: cfg, admins: map[*session]struct{}{}, players: map[*session]*avatar{}}
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: g.cfg.OriginPatterns})
	if err != nil {
		g.cfg.Logger.Debug("websocket accept failed", "err", err)
		return
	}
	c.SetReadLimit(maxMessageBytes)
	ip := clientIP(r)
	ctx, cancel := context.WithCancel(r.Context())
	sess := newSession(g, c, ip, g.cfg.Logger.With("ip", ip), cancel)
	err = sess.run(ctx)
	sess.close(err)
	cancel()
}

// PendingChanged pushes the current pending-request list to every connected admin.
func (g *Gateway) PendingChanged() {
	msg, err := g.pendingMessage()
	if err != nil {
		g.cfg.Logger.Error("listing pending requests", "err", err)
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	for s := range g.admins {
		s.push(msg)
	}
}

func (g *Gateway) addAdmin(s *session) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.admins[s] = struct{}{}
}

func (g *Gateway) removeAdmin(s *session) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.admins, s)
}
