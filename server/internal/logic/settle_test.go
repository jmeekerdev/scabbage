package logic

import (
	"maps"
	"math/rand/v2"
	"slices"
	"testing"
	"time"

	"scabbage/internal/world"
)

// palette is every kind of circuit block, each with all its states, plus things to push.
var palette = func() [][]world.Block {
	var out [][]world.Block
	for _, b := range []world.Block{world.Air, world.Air, world.Air, world.Stone, world.Sand, wire, wire, wire} {
		out = append(out, []world.Block{b})
	}
	for b := world.Wire; b < world.NoteBlock+16; b++ {
		out = append(out, []world.Block{b})
	}
	return append(out, laterKinds()...)
}()

// laterKinds groups the built-ins from the second range by item, so many-state devices aren't
// picked more often than the rest.
func laterKinds() [][]world.Block {
	var out [][]world.Block
	for b := world.Diode; b < world.Tripwire+2; b++ {
		if b.Item() == b {
			out = append(out, nil)
		}
		out[len(out)-1] = append(out[len(out)-1], b)
	}
	return out
}

// box is the size of the random circuits.
var box = Pos{6, 5, 6}

func randomGrid(seed uint64) grid {
	r := rand.New(rand.NewPCG(seed, 1))
	g := grid{}
	for _, p := range everywhere() {
		kind := palette[r.IntN(len(palette))]
		g[p] = kind[r.IntN(len(kind))]
	}
	return g
}

// everywhere is every position in the box, in a fixed order.
func everywhere() []Pos {
	var out []Pos
	for x := range box.X {
		for y := range box.Y {
			for z := range box.Z {
				out = append(out, Pos{x, y, z})
			}
		}
	}
	return out
}

// settle updates every position once, returning all the changes in order.
func settle(t *testing.T, g grid) []Change {
	t.Helper()
	var out []Change
	for _, p := range everywhere() {
		out = append(out, update(t, g, p)...)
	}
	return out
}

func TestSettledCircuitsStaySettled(t *testing.T) {
	for seed := range uint64(300) {
		g := randomGrid(seed)
		settle(t, g)
		if changes := settle(t, g); len(changes) > 0 {
			t.Fatalf("seed %d: still changing %v", seed, changes)
		}
	}
}

func TestTheSameCircuitAlwaysChangesTheSameWay(t *testing.T) {
	for seed := range uint64(50) {
		if a, b := settle(t, randomGrid(seed)), settle(t, randomGrid(seed)); !slices.Equal(a, b) {
			t.Fatalf("seed %d: %v then %v", seed, a, b)
		}
	}
}

func TestTallDoorsMoveAsOne(t *testing.T) {
	g := grid{{-1, 0, 0}: on}
	for y := range int32(6) {
		g[Pos{0, y, 0}] = door
	}
	settle(t, g)
	for y := range int32(6) {
		if g[Pos{0, y, 0}] != open {
			t.Fatalf("door %d is %v", y, g[Pos{0, y, 0}])
		}
	}
}

// senses is what every sensor in the random circuits notices: a player in the box, rain, noon.
var senses = Senses{
	Players: []Spot{{X: 2, Y: 1, Z: 2}},
	Now:     time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC),
	Wet:     true,
}

// run settles a ticking circuit then runs it for `ticks` quarter seconds, returning the
// changes of each tick: its timers beat (to the quarter `clock` gives), its sensors look and its
// due switches are made.
func run(t *testing.T, g grid, ticks int, clock func(tick int) int64) [][]Change {
	t.Helper()
	w := newTicked(g)
	for _, p := range slices.SortedFunc(maps.Keys(g), comparePos) {
		w.update(t, p)
	}
	out := make([][]Change, ticks)
	for i := range out {
		beat, err := Beat(w, everywhere(), clock(i))
		if err != nil {
			t.Fatal(err)
		}
		sensed, err := Sense(w, everywhere(), senses)
		if err != nil {
			t.Fatal(err)
		}
		out[i] = slices.Concat(beat, sensed, w.tick(t))
	}
	return out
}

func TestTickingCircuitsAlwaysChangeTheSameWay(t *testing.T) {
	for seed := range uint64(50) {
		a, b := run(t, randomGrid(seed), 40, quarters), run(t, randomGrid(seed), 40, quarters)
		for i := range a {
			if !slices.Equal(a[i], b[i]) {
				t.Fatalf("seed %d tick %d: %v then %v", seed, i, a[i], b[i])
			}
		}
	}
}

func quarters(tick int) int64 { return int64(tick) }

// clockAt is a not gate looped into itself, planted above the random circuits out of their reach.
var clockAt = Pos{1, box.Y + 3, 1}

func plantClock(g grid) {
	g[clockAt] = notGate
	for _, d := range []Pos{{1, 0, 0}, {1, 0, 1}, {0, 0, 1}, {-1, 0, 1}, {-1, 0, 0}} {
		g[add(clockAt, d)] = wire
	}
}

// TestLoopsTickWithoutStalling runs a clock beside random circuits, which may loop in any way,
// checking it flips every tick and that no tick grows past a bound.
func TestLoopsTickWithoutStalling(t *testing.T) {
	limit := 2 * len(everywhere())
	for seed := range uint64(200) {
		g := randomGrid(seed)
		plantClock(g)
		for i, changes := range run(t, g, 80, quarters) {
			if len(changes) > limit {
				t.Fatalf("seed %d tick %d: %d changes", seed, i, len(changes))
			}
			if !slices.ContainsFunc(changes, func(c Change) bool { return c.Pos == clockAt }) {
				t.Fatalf("seed %d tick %d: the clock stopped", seed, i)
			}
		}
	}
}
