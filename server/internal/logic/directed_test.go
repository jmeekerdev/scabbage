package logic

import (
	"maps"
	"slices"
	"testing"
	"time"

	"scabbage/internal/world"
)

const (
	diodeEast   = world.Diode + 2
	diodeEastOn = world.Diode + 3
	diodeWest   = world.Diode + 6
)

// tickRate is how many times a second a ticked grid ticks, so a quarter second is one tick.
const tickRate = 4

// ticked is a grid that keeps the switches it is asked to make on coming ticks, and like the
// world, rechecks every receiver when a transmitter changes.
type ticked struct {
	grid
	due map[Pos]waiting
}

type waiting struct {
	ticks int
	flip  bool
}

func newTicked(g grid) *ticked { return &ticked{grid: g, due: map[Pos]waiting{}} }

func (w *ticked) Schedule(p Pos, after time.Duration, flip bool) {
	at := waiting{max(1, int(after*tickRate/time.Second)), flip}
	if was, ok := w.due[p]; !ok || (!was.flip && !flip && at.ticks > was.ticks) {
		w.due[p] = at
	}
}

func (w *ticked) SetBlock(x, y, z int32, b world.Block) error {
	was := w.grid[Pos{x, y, z}]
	w.grid[Pos{x, y, z}] = b
	if was != b && (was.Item() == world.Transmitter || b.Item() == world.Transmitter) {
		for p, r := range w.grid {
			if r.Item() == world.Receiver {
				w.Schedule(p, 0, false)
			}
		}
	}
	return nil
}

// tick counts down one tick and makes the switches whose wait is over.
func (w *ticked) tick(t *testing.T) []Change {
	t.Helper()
	var due []Due
	for _, p := range slices.SortedFunc(maps.Keys(w.due), comparePos) {
		if at := w.due[p]; at.ticks > 1 {
			w.due[p] = waiting{at.ticks - 1, at.flip}
		} else {
			delete(w.due, p)
			due = append(due, Due{p, at.flip})
		}
	}
	changes, err := Step(w, due)
	if err != nil {
		t.Fatal(err)
	}
	return changes
}

func (w *ticked) update(t *testing.T, at Pos) {
	t.Helper()
	if _, err := Update(w, at); err != nil {
		t.Fatal(err)
	}
}

func TestADiodePassesPowerForwardOnTheNextTick(t *testing.T) {
	w := newTicked(line(on, wire, diodeEast, wire, lamp))
	w.update(t, Pos{0, 0, 0})
	if w.grid[Pos{2, 0, 0}] != diodeEast || w.grid[Pos{4, 0, 0}] != lamp {
		t.Fatalf("switched inside the update: %v", w.grid)
	}
	w.tick(t)
	if w.grid[Pos{2, 0, 0}] != diodeEastOn || w.grid[Pos{4, 0, 0}] != lit {
		t.Fatalf("after a tick: %v", w.grid)
	}
	w.grid[Pos{0, 0, 0}] = world.Switch
	w.update(t, Pos{0, 0, 0})
	w.tick(t)
	if w.grid[Pos{2, 0, 0}] != diodeEast || w.grid[Pos{4, 0, 0}] != lamp {
		t.Fatalf("after switching off: %v", w.grid)
	}
}

func TestADiodeWontPassPowerBackwards(t *testing.T) {
	w := newTicked(line(on, wire, diodeWest, wire, lamp))
	w.update(t, Pos{0, 0, 0})
	w.tick(t)
	if w.grid[Pos{2, 0, 0}] != diodeWest || w.grid[Pos{4, 0, 0}] != lamp {
		t.Fatalf("power went backwards: %v", w.grid)
	}
}

func TestADiodePowersOnlyWhatIsInFront(t *testing.T) {
	g := line(on, diodeEast, lamp)
	g[Pos{1, 1, 0}] = lamp
	g[Pos{1, 0, 1}] = wire
	w := newTicked(g)
	w.update(t, Pos{0, 0, 0})
	w.tick(t)
	if g[Pos{2, 0, 0}] != lit || g[Pos{1, 1, 0}] != lamp || g[Pos{1, 0, 1}] != wire {
		t.Fatalf("grid %v", g)
	}
}

func TestEachDiodeInARowTakesATick(t *testing.T) {
	w := newTicked(line(on, diodeEast, diodeEast, lamp))
	w.update(t, Pos{0, 0, 0})
	w.tick(t)
	if w.grid[Pos{1, 0, 0}] != diodeEastOn || w.grid[Pos{2, 0, 0}] != diodeEast {
		t.Fatalf("after one tick: %v", w.grid)
	}
	w.tick(t)
	if w.grid[Pos{2, 0, 0}] != diodeEastOn || w.grid[Pos{3, 0, 0}] != lit {
		t.Fatalf("after two ticks: %v", w.grid)
	}
}

func TestATickSwitchesFromHowThingsWereBeforeIt(t *testing.T) {
	w := newTicked(line(on, diodeEast, diodeEast, lamp))
	w.Schedule(Pos{1, 0, 0}, 0, false)
	w.Schedule(Pos{2, 0, 0}, 0, false)
	w.tick(t)
	if _, due := w.due[Pos{2, 0, 0}]; w.grid[Pos{2, 0, 0}] != diodeEast || !due {
		t.Fatalf("the second diode saw this tick's power: %v, due %v", w.grid, w.due)
	}
}

func TestADiodeLoopLatchesWithoutTickingForever(t *testing.T) {
	// The diode's front wire curls round to its back, so once on it keeps itself on.
	g := line(on, wire, diodeEast, wire)
	for _, p := range []Pos{{1, 0, 1}, {2, 0, 1}, {3, 0, 1}} {
		g[p] = wire
	}
	w := newTicked(g)
	w.update(t, Pos{0, 0, 0})
	w.tick(t)
	g[Pos{0, 0, 0}] = world.Switch
	w.update(t, Pos{0, 0, 0})
	for range 3 {
		w.tick(t)
	}
	if g[Pos{2, 0, 0}] != diodeEastOn || len(w.due) != 0 {
		t.Fatalf("grid %v, due %v", g, w.due)
	}
}

func TestStepIgnoresBlocksThatAreNoLongerDirected(t *testing.T) {
	w := newTicked(line(on, world.Stone))
	if changes := w.tick(t); len(changes) != 0 {
		t.Fatal(changes)
	}
	w.Schedule(Pos{1, 0, 0}, 0, false)
	if changes := w.tick(t); len(changes) != 0 {
		t.Fatal(changes)
	}
}
