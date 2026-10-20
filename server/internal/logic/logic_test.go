package logic

import (
	"testing"

	"scabbage/internal/world"
)

// grid is a tiny in-memory world; everything unset is air.
type grid map[Pos]world.Block

func (g grid) Block(x, y, z int32) (world.Block, error) { return g[Pos{x, y, z}], nil }

func (g grid) SetBlock(x, y, z int32, b world.Block) error {
	g[Pos{x, y, z}] = b
	return nil
}

// line lays blocks along +x from the origin.
func line(blocks ...world.Block) grid {
	g := grid{}
	for x, b := range blocks {
		g[Pos{int32(x), 0, 0}] = b
	}
	return g
}

const (
	on   = world.Switch + 1
	wire = world.Wire
	lamp = world.SignalLamp
	lit  = world.SignalLamp + 1
)

func update(t *testing.T, g grid, at Pos) []Change {
	t.Helper()
	changes, err := Update(g, at)
	if err != nil {
		t.Fatal(err)
	}
	return changes
}

func TestASwitchLightsALampDownTheWire(t *testing.T) {
	g := line(on, wire, wire, wire, lamp)
	changes := update(t, g, Pos{0, 0, 0})
	if g[Pos{4, 0, 0}] != lit || g[Pos{2, 0, 0}] != world.Wire+1 || len(changes) != 4 {
		t.Fatalf("grid %v, changes %v", g, changes)
	}
}

func TestTurningOffDarkensTheCircuit(t *testing.T) {
	g := line(world.Switch, wire+1, wire+1, lit)
	update(t, g, Pos{0, 0, 0})
	if g[Pos{1, 0, 0}] != wire || g[Pos{3, 0, 0}] != lamp {
		t.Fatalf("grid %v", g)
	}
}

func TestAnotherSourceKeepsTheWirePowered(t *testing.T) {
	g := line(world.Switch, wire+1, wire+1, lit, wire+1, world.Plate+1)
	g[Pos{2, 1, 0}] = world.Button + 1
	if changes := update(t, g, Pos{0, 0, 0}); len(changes) != 0 {
		t.Fatalf("the button still powers it: %v", changes)
	}
}

func TestBreakingTheWireCutsThePower(t *testing.T) {
	g := line(on, wire+1, wire+1, wire+1, lit)
	g[Pos{2, 0, 0}] = world.Air
	update(t, g, Pos{2, 0, 0})
	if g[Pos{1, 0, 0}] != wire+1 || g[Pos{3, 0, 0}] != wire || g[Pos{4, 0, 0}] != lamp {
		t.Fatalf("grid %v", g)
	}
}

func TestLampsBesideASourceLight(t *testing.T) {
	g := line(lamp, on, lamp)
	update(t, g, Pos{1, 0, 0})
	if g[Pos{0, 0, 0}] != lit || g[Pos{2, 0, 0}] != lit {
		t.Fatalf("grid %v", g)
	}
}

func TestLampsDoNotCarryPower(t *testing.T) {
	g := line(on, lamp, wire, lamp)
	update(t, g, Pos{0, 0, 0})
	if g[Pos{2, 0, 0}] != wire || g[Pos{3, 0, 0}] != lamp {
		t.Fatalf("grid %v", g)
	}
}

func TestPlacingWireJoinsAPoweredCircuit(t *testing.T) {
	g := line(on, wire+1, wire, wire, lamp)
	update(t, g, Pos{2, 0, 0})
	if g[Pos{4, 0, 0}] != lit {
		t.Fatalf("grid %v", g)
	}
}

func TestHugeCircuitsStopSpreading(t *testing.T) {
	g := grid{{-1, 0, 0}: on}
	for x := range int32(MaxWires + 10) {
		g[Pos{x, 0, 0}] = wire
	}
	update(t, g, Pos{-1, 0, 0})
	if g[Pos{MaxWires + 5, 0, 0}] != wire {
		t.Fatal("power spread past the limit")
	}
}

func TestUse(t *testing.T) {
	cases := map[world.Block]world.Block{
		world.Switch: on, on: world.Switch, world.Button: world.Button + 1, world.Button + 1: world.Button + 1,
	}
	for b, want := range cases {
		if got, ok := Use(b); !ok || got != want {
			t.Errorf("Use(%v) = %v, %v", b, got, ok)
		}
	}
	if _, ok := Use(world.Stone); ok {
		t.Error("stone is not usable")
	}
}

func TestInvolves(t *testing.T) {
	if !Involves(world.Plate+1) || !Involves(lamp) || Involves(world.Lamp) || Involves(world.Stone) {
		t.Fatal("only circuit blocks are involved")
	}
}
