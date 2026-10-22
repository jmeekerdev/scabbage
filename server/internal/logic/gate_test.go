package logic

import (
	"testing"

	"scabbage/internal/world"
)

const (
	andGate = world.Gate + 12*iota + 2 // each mode facing east
	orGate
	xorGate
	nandGate
	norGate
	notGate
)

// Gates below sit at the origin facing east, with a lamp in front.
var (
	gateAt   = Pos{0, 0, 0}
	gateBack = Pos{-1, 0, 0}
	north    = Pos{0, 0, -1}
	south    = Pos{0, 0, 1}
	inFront  = Pos{1, 0, 0}
)

// gateLit sets up the gate with switches at the given inputs, runs a tick and reports the lamp.
func gateLit(t *testing.T, gate world.Block, inputs map[Pos]world.Block) bool {
	t.Helper()
	g := grid{gateAt: gate, inFront: lamp}
	for p, b := range inputs {
		g[p] = b
	}
	w := newTicked(g)
	w.update(t, gateAt)
	w.tick(t)
	return g[inFront] == lit
}

func TestGatesJudgeTheInputsJoinedToThem(t *testing.T) {
	off := world.Switch
	cases := []struct {
		name   string
		gate   world.Block
		inputs map[Pos]world.Block
		want   bool
	}{
		{"and, both on", andGate, map[Pos]world.Block{north: on, south: on}, true},
		{"and, one on", andGate, map[Pos]world.Block{north: on, south: off}, false},
		{"and, its only input on", andGate, map[Pos]world.Block{gateBack: on}, true},
		{"and, nothing joined", andGate, nil, false},
		{"or, one on", orGate, map[Pos]world.Block{north: off, south: on}, true},
		{"or, none on", orGate, map[Pos]world.Block{north: off, south: off}, false},
		{"xor, one on", xorGate, map[Pos]world.Block{north: on, south: off}, true},
		{"xor, both on", xorGate, map[Pos]world.Block{north: on, south: on}, false},
		{"nand, both on", nandGate, map[Pos]world.Block{north: on, south: on}, false},
		{"nand, one on", nandGate, map[Pos]world.Block{north: on, south: off}, true},
		{"nor, none on", norGate, map[Pos]world.Block{north: off, gateBack: off}, true},
		{"nor, one on", norGate, map[Pos]world.Block{north: off, gateBack: on}, false},
		{"not, back off", notGate, map[Pos]world.Block{gateBack: off}, true},
		{"not, back on", notGate, map[Pos]world.Block{gateBack: on}, false},
		{"not ignores its sides", notGate, map[Pos]world.Block{north: on}, true},
	}
	for _, c := range cases {
		if got := gateLit(t, c.gate, c.inputs); got != c.want {
			t.Errorf("%s: lit %v, want %v", c.name, got, c.want)
		}
	}
}

func TestAGateTakesNoInputFromItsFront(t *testing.T) {
	g := grid{gateAt: andGate, inFront: wire, Pos{2, 0, 0}: on, south: on}
	w := newTicked(g)
	w.update(t, gateAt)
	w.tick(t)
	if g[gateAt] != andGate+1 {
		t.Fatalf("the powered wire in front counted as an input: %v", g)
	}
}

func TestAGateReadsWireBesideIt(t *testing.T) {
	g := grid{gateAt: andGate, inFront: lamp, Pos{0, 0, 2}: on, south: wire, north: wire, Pos{0, 0, -2}: on}
	w := newTicked(g)
	w.update(t, Pos{0, 0, 2})
	w.update(t, Pos{0, 0, -2})
	w.tick(t)
	if g[inFront] != lit {
		t.Fatalf("grid %v", g)
	}
}

func TestTappingAGateCyclesItsModeAndKeepsItsFacing(t *testing.T) {
	g := grid{gateAt: notGate + 1, inFront: lamp}
	w := newTicked(g)
	if _, used, err := Use(w, gateAt); err != nil || !used {
		t.Fatal(used, err)
	}
	if _, due := w.due[gateAt]; g[gateAt] != andGate+1 || !due {
		t.Fatalf("after tapping not: %v, due %v", g[gateAt], w.due)
	}
	w.tick(t)
	if g[gateAt] != andGate {
		t.Fatalf("an and gate with nothing joined stays on: %v", g[gateAt])
	}
	if _, _, err := Use(w, gateAt); err != nil || g[gateAt] != orGate {
		t.Fatalf("after tapping and: %v, %v", g[gateAt], err)
	}
}

func TestANotGateLoopedIntoItselfTicksAsAClock(t *testing.T) {
	// The wire curls from the front round the south side into the back, which a not gate alone reads.
	g := grid{gateAt: notGate}
	for _, p := range []Pos{inFront, {1, 0, 1}, south, {-1, 0, 1}, gateBack} {
		g[p] = wire
	}
	w := newTicked(g)
	w.update(t, gateAt)
	var seen []bool
	for range 6 {
		w.tick(t)
		seen = append(seen, isOn(g[gateAt]))
	}
	for i := 1; i < len(seen); i++ {
		if seen[i] == seen[i-1] {
			t.Fatalf("not flipping each tick: %v", seen)
		}
	}
}
