package logic

import (
	"testing"

	"scabbage/internal/world"
)

// capacitorEast holds power n seconds, 4n ticks of a ticked grid.
func capacitorEast(n int) world.Block { return world.Capacitor + world.Block(12*(n-1)+2) }

const (
	toggle    = world.Toggle
	toggledOn = world.Toggle + 2
)

func TestACapacitorPowersOnTheNextTickAndHoldsAfterItsInputStops(t *testing.T) {
	w := newTicked(line(on, capacitorEast(1), lamp))
	w.update(t, Pos{0, 0, 0})
	w.tick(t)
	if w.grid[Pos{2, 0, 0}] != lit {
		t.Fatalf("not lit: %v", w.grid)
	}
	w.grid[Pos{0, 0, 0}] = world.Switch
	w.update(t, Pos{0, 0, 0})
	w.ticks(t, 3)
	if w.grid[Pos{2, 0, 0}] != lit {
		t.Fatalf("let go early: %v", w.grid)
	}
	w.tick(t)
	if w.grid[Pos{2, 0, 0}] != lamp || len(w.due) != 0 {
		t.Fatalf("still lit: %v, due %v", w.grid, w.due)
	}
}

func TestACapacitorHoldsFromTheLastTimeItsInputStopped(t *testing.T) {
	w := newTicked(line(on, capacitorEast(1), lamp))
	w.update(t, Pos{0, 0, 0})
	w.tick(t)
	for _, b := range []world.Block{world.Switch, on, world.Switch} {
		w.grid[Pos{0, 0, 0}] = b
		w.update(t, Pos{0, 0, 0})
		w.tick(t)
	}
	w.ticks(t, 2)
	if w.grid[Pos{2, 0, 0}] != lit {
		t.Fatalf("let go early: %v", w.grid)
	}
	w.tick(t)
	if w.grid[Pos{2, 0, 0}] != lamp {
		t.Fatalf("still lit: %v", w.grid)
	}
}

func TestTappingACapacitorCyclesItsSetting(t *testing.T) {
	g := line(capacitorEast(world.Capacitors))
	if !use(t, g, Pos{}) || g[Pos{}] != capacitorEast(1) {
		t.Fatalf("grid %v", g)
	}
}

func TestAToggleFlipsEachTimePowerArrives(t *testing.T) {
	g := line(world.Switch, wire, toggle, wire, lamp)
	for i, want := range []world.Block{lit, lit, lamp, lamp, lit} {
		set(t, g, Pos{}, []world.Block{on, world.Switch}[i%2])
		if g[Pos{4, 0, 0}] != want {
			t.Fatalf("after %d changes: %v", i+1, g)
		}
	}
}

func TestATogglesOwnPowerDoesNotFlipIt(t *testing.T) {
	g := line(world.Switch, toggle, wire, wire)
	g[Pos{3, 0, 1}], g[Pos{2, 0, 1}], g[Pos{1, 0, 1}] = wire, wire, wire // loops back beside it
	set(t, g, Pos{}, on)
	if g[Pos{1, 0, 0}].Item() != toggle || !isSource(g[Pos{1, 0, 0}]) {
		t.Fatalf("grid %v", g)
	}
}

func TestTappingAToggleFlipsIt(t *testing.T) {
	g := line(toggle, wire, lamp)
	use(t, g, Pos{})
	if g[Pos{}] != toggledOn || g[Pos{2, 0, 0}] != lit {
		t.Fatalf("grid %v", g)
	}
	use(t, g, Pos{})
	if g[Pos{}] != toggle || g[Pos{2, 0, 0}] != lamp {
		t.Fatalf("grid %v", g)
	}
}

func TestMemoryBlocksAreInvolved(t *testing.T) {
	if !Involves(toggle) || !Involves(capacitorEast(3)) {
		t.Fatal("memory blocks are circuit blocks")
	}
}
