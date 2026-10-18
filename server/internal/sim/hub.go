// Package sim runs the game simulation loop, which only ticks while players are connected.
package sim

import (
	"sync"
	"time"
)

// Ticker abstracts time.Ticker so tests can drive ticks manually.
type Ticker interface {
	C() <-chan time.Time
	Stop()
}

type realTicker struct{ t *time.Ticker }

func (r realTicker) C() <-chan time.Time { return r.t.C }
func (r realTicker) Stop()               { r.t.Stop() }

// NewRealTicker wraps time.NewTicker.
func NewRealTicker(d time.Duration) Ticker { return realTicker{time.NewTicker(d)} }

// Config configures a Hub.
type Config struct {
	TickRate  int                        // ticks per second
	NewTicker func(time.Duration) Ticker // defaults to NewRealTicker
	OnTick    func(tick uint64)          // called on the loop goroutine; must not call Join/Leave
}

// Hub tracks connected players and runs the tick loop only while at least one is present.
type Hub struct {
	cfg     Config
	mu      sync.Mutex
	players int
	stop    chan struct{}
	done    chan struct{}
}

// NewHub returns an idle hub.
func NewHub(cfg Config) *Hub {
	if cfg.NewTicker == nil {
		cfg.NewTicker = NewRealTicker
	}
	if cfg.OnTick == nil {
		cfg.OnTick = func(uint64) {}
	}
	return &Hub{cfg: cfg}
}

// TickRate returns the configured ticks per second.
func (h *Hub) TickRate() int { return h.cfg.TickRate }

// Join registers a player, starting the loop if the hub was idle.
func (h *Hub) Join() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.players++
	if h.players == 1 {
		h.startLocked()
	}
}

// Leave unregisters a player, stopping the loop when the hub becomes empty.
func (h *Hub) Leave() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.players == 0 {
		return
	}
	h.players--
	if h.players == 0 {
		h.stopLocked()
	}
}

// Running reports whether the tick loop is active.
func (h *Hub) Running() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.stop != nil
}

// Players returns the number of connected players.
func (h *Hub) Players() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.players
}

// Close stops the loop regardless of player count.
func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.stopLocked()
}

func (h *Hub) startLocked() {
	h.stop = make(chan struct{})
	h.done = make(chan struct{})
	ticker := h.cfg.NewTicker(time.Second / time.Duration(h.cfg.TickRate))
	go h.loop(ticker, h.stop, h.done)
}

func (h *Hub) stopLocked() {
	if h.stop == nil {
		return
	}
	close(h.stop)
	<-h.done
	h.stop, h.done = nil, nil
}

func (h *Hub) loop(ticker Ticker, stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)
	defer ticker.Stop()
	var tick uint64
	for {
		select {
		case <-stop:
			return
		case <-ticker.C():
			tick++
			h.cfg.OnTick(tick)
		}
	}
}
