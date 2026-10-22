package logic

import (
	"testing"

	"scabbage/internal/world"
)

const stoneCable = world.Cables

func TestWireClimbsAStep(t *testing.T) {
	g := line(on, wire, world.Stone, world.Air, lamp)
	g[Pos{2, 1, 0}], g[Pos{3, 1, 0}] = wire, wire
	g[Pos{4, 1, 0}] = lamp
	update(t, g, Pos{0, 0, 0})
	if g[Pos{3, 1, 0}] != wire+1 || g[Pos{4, 1, 0}] != lit {
		t.Fatalf("grid %v", g)
	}
}

func TestWireDropsDownAStep(t *testing.T) {
	g := grid{{0, 1, 0}: on, {1, 1, 0}: wire, {1, 0, 0}: world.Stone, {2, 0, 0}: wire, {3, 0, 0}: lamp}
	update(t, g, Pos{0, 1, 0})
	if g[Pos{3, 0, 0}] != lit {
		t.Fatalf("grid %v", g)
	}
}

func TestABlockOverTheLowerWireCutsTheStep(t *testing.T) {
	g := line(on, wire, world.Stone)
	g[Pos{2, 1, 0}], g[Pos{3, 1, 0}] = wire, lamp
	g[Pos{1, 1, 0}] = world.Stone
	update(t, g, Pos{0, 0, 0})
	if g[Pos{2, 1, 0}] != wire || g[Pos{3, 1, 0}] != lamp {
		t.Fatalf("grid %v", g)
	}
}

func TestStepsJoinOnlyWire(t *testing.T) {
	g := line(on, wire, world.Stone)
	g[Pos{2, 1, 0}] = lamp
	update(t, g, Pos{0, 0, 0})
	if g[Pos{2, 1, 0}] != lamp {
		t.Fatalf("a lamp up a step isn't beside the wire: %v", g)
	}
}

func TestCablesCarryPowerThroughWalls(t *testing.T) {
	g := line(on, stoneCable, stoneCable, wire, lamp)
	update(t, g, Pos{0, 0, 0})
	if g[Pos{1, 0, 0}] != stoneCable+1 || g[Pos{2, 0, 0}] != stoneCable+1 || g[Pos{4, 0, 0}] != lit {
		t.Fatalf("grid %v", g)
	}
	g[Pos{0, 0, 0}] = world.Switch
	update(t, g, Pos{0, 0, 0})
	if g[Pos{2, 0, 0}] != stoneCable || g[Pos{4, 0, 0}] != lamp {
		t.Fatalf("grid %v", g)
	}
}

func TestPoweredCablesPowerMachinesAndDiodes(t *testing.T) {
	g := line(on, stoneCable, diodeEast)
	g[Pos{1, 1, 0}] = lamp
	w := newTicked(g)
	w.update(t, Pos{0, 0, 0})
	w.tick(t)
	if g[Pos{1, 1, 0}] != lit || g[Pos{2, 0, 0}] != diodeEastOn {
		t.Fatalf("grid %v", g)
	}
}

func TestPistonsLeaveCablesAlone(t *testing.T) {
	if !Involves(stoneCable) || movable(stoneCable) {
		t.Fatal("cables are part of a circuit")
	}
}
