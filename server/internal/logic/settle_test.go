package logic

import (
	"math/rand/v2"
	"slices"
	"testing"

	"scabbage/internal/world"
)

// palette is every kind of circuit block, in each of its states, plus things to push.
var palette = func() []world.Block {
	out := []world.Block{world.Air, world.Air, world.Air, world.Stone, world.Sand, wire, wire, wire}
	for b := world.Wire; b < world.NoteBlock+16; b++ {
		out = append(out, b)
	}
	return out
}()

// box is the size of the random circuits.
var box = Pos{6, 5, 6}

func randomGrid(seed uint64) grid {
	r := rand.New(rand.NewPCG(seed, 1))
	g := grid{}
	for _, p := range everywhere() {
		g[p] = palette[r.IntN(len(palette))]
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
