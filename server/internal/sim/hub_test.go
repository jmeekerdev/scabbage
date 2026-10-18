package sim

import (
	"sync"
	"testing"
	"time"
)

type fakeTicker struct {
	c       chan time.Time
	mu      sync.Mutex
	stopped bool
}

func (f *fakeTicker) C() <-chan time.Time { return f.c }

func (f *fakeTicker) Stop() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stopped = true
}

func (f *fakeTicker) isStopped() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.stopped
}

type fakeClock struct {
	mu      sync.Mutex
	tickers []*fakeTicker
}

func (fc *fakeClock) NewTicker(time.Duration) Ticker {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	t := &fakeTicker{c: make(chan time.Time)}
	fc.tickers = append(fc.tickers, t)
	return t
}

func (fc *fakeClock) created() []*fakeTicker {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	return append([]*fakeTicker(nil), fc.tickers...)
}

func newTestHub(t *testing.T) (*Hub, *fakeClock, chan uint64) {
	t.Helper()
	clock := &fakeClock{}
	ticked := make(chan uint64, 16)
	h := NewHub(Config{
		TickRate:  20,
		NewTicker: clock.NewTicker,
		OnTick:    func(n uint64) { ticked <- n },
	})
	t.Cleanup(h.Close)
	return h, clock, ticked
}

func waitTick(t *testing.T, ticked <-chan uint64) uint64 {
	t.Helper()
	select {
	case n := <-ticked:
		return n
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for tick")
		return 0
	}
}

func TestHubIdleWithNoPlayers(t *testing.T) {
	h, clock, _ := newTestHub(t)
	if h.Running() {
		t.Fatal("hub should not run with zero players")
	}
	if n := len(clock.created()); n != 0 {
		t.Fatalf("expected no tickers while idle, got %d", n)
	}
}

func TestHubTicksWhilePlayersPresent(t *testing.T) {
	h, clock, ticked := newTestHub(t)
	h.Join()
	if !h.Running() {
		t.Fatal("hub should run once a player joins")
	}
	tk := clock.created()[0]
	tk.c <- time.Now()
	tk.c <- time.Now()
	if got := waitTick(t, ticked); got != 1 {
		t.Fatalf("first tick = %d, want 1", got)
	}
	if got := waitTick(t, ticked); got != 2 {
		t.Fatalf("second tick = %d, want 2", got)
	}
}

func TestHubStopsWhenLastPlayerLeaves(t *testing.T) {
	h, clock, _ := newTestHub(t)
	h.Join()
	h.Join()
	h.Leave()
	if !h.Running() {
		t.Fatal("hub should keep running while a player remains")
	}
	h.Leave()
	if h.Running() {
		t.Fatal("hub should stop when empty")
	}
	if !clock.created()[0].isStopped() {
		t.Fatal("ticker should be stopped when empty")
	}
}

func TestHubRestartsOnRejoin(t *testing.T) {
	h, clock, ticked := newTestHub(t)
	h.Join()
	h.Leave()
	h.Join()
	tickers := clock.created()
	if len(tickers) != 2 {
		t.Fatalf("expected a fresh ticker on rejoin, got %d", len(tickers))
	}
	tickers[1].c <- time.Now()
	if got := waitTick(t, ticked); got != 1 {
		t.Fatalf("tick after rejoin = %d, want 1", got)
	}
}

func TestHubExtraLeaveIsHarmless(t *testing.T) {
	h, _, _ := newTestHub(t)
	h.Leave()
	h.Join()
	if !h.Running() {
		t.Fatal("hub should run after join despite earlier stray leave")
	}
}
