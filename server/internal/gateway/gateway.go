// Package gateway accepts WebSocket connections, admits devices through the access flow and routes messages.
package gateway

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"

	"scabbage/internal/access"
	"scabbage/internal/arenas"
	"scabbage/internal/art"
	"scabbage/internal/crafting"
	"scabbage/internal/drops"
	"scabbage/internal/logic"
	"scabbage/internal/mobs"
	"scabbage/internal/rounds"
	"scabbage/internal/scripts"
	"scabbage/internal/sim"
	"scabbage/internal/world"
	"scabbage/internal/zones"
)

// Config configures the gateway handler.
type Config struct {
	Hub            *sim.Hub
	Access         *access.Service
	Art            *art.Service
	World          *world.World
	Recipes        []crafting.Recipe // crafting.Default() when nil
	Mobs           *mobs.Herd        // a new herd when nil; guarded by the gateway's mu
	Zones          *zones.Service
	Arenas         *arenas.Service
	ScriptDB       *sql.DB // where script blocks keep their Lua and characters, and players their journals
	ServerName     string
	Mode           access.Mode // for players without their own; creative when unset
	HelloTimeout   time.Duration
	OriginPatterns []string // extra allowed Origin hosts, e.g. the Vite dev server
	Logger         *slog.Logger
}

// Gateway is an http.Handler for the game WebSocket endpoint.
type Gateway struct {
	cfg         Config
	mu          sync.Mutex
	admins      map[*session]struct{}
	players     map[*session]*avatar     // in-world sessions
	entity      uint32                   // last entity id handed out
	editMu      sync.Mutex               // makes each edit's check, change and broadcast atomic
	circuits    circuits                 // guarded by editMu
	scripts     *scripts.Service         // guarded by editMu
	scriptErrAt map[world.Pos]time.Time  // when each script block may next report an error; guarded by editMu
	art         artCache                 // guarded by mu
	drops       drops.Field              // guarded by mu
	savedAt     time.Time                // when changed inventories were last saved; guarded by mu
	mobsAt      time.Time                // when the herd last stepped; guarded by mu
	arenas      map[string]arenaView     // by lowercased zone name; guarded by mu
	rounds      map[string]*rounds.Round // games being played, by lowercased zone name; guarded by mu
	roundEvents []roundEvent             // starts and ends for scripts to hear on the next tick; guarded by mu
	dialogue    uint32                   // last Dialogue id handed out; guarded by mu
}

// New returns a Gateway with defaults applied, registering the custom blocks already approved.
func New(cfg Config) (*Gateway, error) {
	if cfg.HelloTimeout == 0 {
		cfg.HelloTimeout = 10 * time.Second
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.Mobs == nil {
		cfg.Mobs = mobs.New(uint64(time.Now().UnixNano()), true)
	}
	if cfg.Recipes == nil {
		cfg.Recipes = crafting.Default()
	}
	g := &Gateway{cfg: cfg, admins: map[*session]struct{}{}, players: map[*session]*avatar{}, rounds: map[string]*rounds.Round{},
		circuits: circuits{letGo: map[logic.Pos]time.Time{}, held: map[logic.Pos]bool{}}, scriptErrAt: map[world.Pos]time.Time{}}
	var err error
	if g.scripts, err = scripts.New(cfg.ScriptDB, g.scriptHost, g.npcChanged); err != nil {
		return nil, err
	}
	return g, g.loadArt()
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
