package logic

import (
	"maps"
	"slices"
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

// fence is where the grid's protected land starts; pistons can't push past it.
const fence = 50

func (g grid) CanPush(_, at Pos) bool { return at.X < fence }

func (g grid) Occupied(Pos) bool { return false }

// crowded is a grid with people standing in it.
type crowded struct {
	grid
	people map[Pos]bool
}

func (c crowded) Occupied(p Pos) bool { return c.people[p] }

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

func use(t *testing.T, g grid, at Pos) bool {
	t.Helper()
	_, ok, err := Use(g, at)
	if err != nil {
		t.Fatal(err)
	}
	return ok
}

func TestUse(t *testing.T) {
	cases := map[world.Block]world.Block{
		world.Switch: on, on: world.Switch, world.Button: world.Button + 1, world.Button + 1: world.Button + 1,
		world.NoteBlock + 1: world.NoteBlock + 3, world.NoteBlock + 14: world.NoteBlock,
		world.Counter + 7: world.Counter + 1,
	}
	for b, want := range cases {
		g := line(b)
		if !use(t, g, Pos{}) || g[Pos{}] != want {
			t.Errorf("using %v made %v", b, g[Pos{}])
		}
	}
	if use(t, line(world.Stone), Pos{}) || use(t, line(lamp), Pos{}) {
		t.Error("stone and lamps are not usable")
	}
}

func TestInvolves(t *testing.T) {
	if !Involves(world.Plate+1) || !Involves(lamp) || !Involves(world.Piston+5) || Involves(world.Lamp) || Involves(world.Stone) {
		t.Fatal("only circuit blocks are involved")
	}
}

const (
	door      = world.Door
	open      = world.Door + 1
	eastward  = world.Piston + 2
	pushing   = world.Piston + 3
	counter   = world.Counter
	sounding  = world.NoteBlock + 1
	timerOn   = world.Timer + 1
	teleport  = world.Teleporter
	somewhere = world.Stone
)

func TestPowerOpensAWholeDoor(t *testing.T) {
	g := line(on, wire, door)
	g[Pos{2, 1, 0}] = door
	update(t, g, Pos{})
	if g[Pos{2, 0, 0}] != open || g[Pos{2, 1, 0}] != open {
		t.Fatalf("grid %v", g)
	}
	g[Pos{}] = world.Switch
	update(t, g, Pos{})
	if g[Pos{2, 0, 0}] != door || g[Pos{2, 1, 0}] != door {
		t.Fatalf("grid %v", g)
	}
}

func TestDoorsOpenByHand(t *testing.T) {
	g := grid{{0, 0, 0}: door, {0, 1, 0}: door}
	if !use(t, g, Pos{0, 1, 0}) || g[Pos{}] != open || g[Pos{0, 1, 0}] != open {
		t.Fatalf("grid %v", g)
	}
	use(t, g, Pos{})
	if g[Pos{}] != door {
		t.Fatalf("grid %v", g)
	}
}

func TestDoorsWaitForPeopleToStepOut(t *testing.T) {
	g := line(world.Switch, wire+1, open)
	g[Pos{2, 1, 0}] = open
	w := crowded{g, map[Pos]bool{{2, 1, 0}: true}}
	if _, err := Update(w, Pos{}); err != nil || g[Pos{2, 0, 0}] != open {
		t.Fatalf("power closed a door on someone: %v %v", g, err)
	}
	if _, ok, err := Use(w, Pos{2, 0, 0}); err != nil || !ok || g[Pos{2, 1, 0}] != open {
		t.Fatalf("a hand closed a door on someone: %v %v", g, err)
	}
	if _, ok, _ := Use(crowded{line(door), w.people}, Pos{}); !ok {
		t.Fatal("doors still open with people beside them")
	}
}

func TestPistonsPushARowOnce(t *testing.T) {
	g := line(on, eastward, somewhere, world.Dirt, world.Air, world.Sand)
	update(t, g, Pos{})
	want := line(on, pushing, world.Air, somewhere, world.Dirt, world.Sand)
	for p, b := range want {
		if g[p] != b {
			t.Fatalf("grid %v, want %v", g, want)
		}
	}
	update(t, g, Pos{})
	if g[Pos{3, 0, 0}] != somewhere {
		t.Fatal("a piston that stays powered pushes only once")
	}
}

func TestPistonsPushIntoWater(t *testing.T) {
	g := line(on, eastward, somewhere, world.Water)
	update(t, g, Pos{})
	if g[Pos{3, 0, 0}] != somewhere {
		t.Fatalf("grid %v", g)
	}
}

func TestPistonsCannotPushEverything(t *testing.T) {
	cases := map[string]grid{
		"bedrock":   line(on, eastward, somewhere, world.Bedrock),
		"logic":     line(on, eastward, wire),
		"open land": line(on, eastward),
		"too many":  line(append([]world.Block{on, eastward}, slices.Repeat([]world.Block{somewhere}, MaxPush+1)...)...),
	}
	fenced := grid{{fence - 2, 0, 0}: on, {fence - 1, 0, 0}: eastward, {fence, 0, 0}: somewhere}
	cases["protected"] = fenced
	for name, g := range cases {
		before := maps.Clone(g)
		update(t, g, Pos{})
		update(t, g, Pos{fence - 2, 0, 0})
		for p, b := range before {
			if p.X > 1 && p.X != fence-1 && g[p] != b {
				t.Errorf("%s: pushed %v", name, g)
			}
		}
	}
}

func TestCountersPowerAfterNinePulses(t *testing.T) {
	g := line(world.Switch, wire, counter, wire, lamp)
	for range 9 {
		if g[Pos{4, 0, 0}] != lamp {
			t.Fatalf("lit early: %v", g)
		}
		set(t, g, Pos{}, on)
		set(t, g, Pos{}, world.Switch)
	}
	if g[Pos{4, 0, 0}] != lit || g[Pos{2, 0, 0}].Item() != counter {
		t.Fatalf("grid %v", g)
	}
	use(t, g, Pos{2, 0, 0})
	if g[Pos{4, 0, 0}] != lamp || g[Pos{2, 0, 0}] != counter {
		t.Fatalf("resetting should darken it: %v", g)
	}
}

func TestCountersCountBackDownToZero(t *testing.T) {
	g := line(world.Switch, wire, counter, wire, lamp)
	pulse := func() {
		set(t, g, Pos{}, on)
		set(t, g, Pos{}, world.Switch)
	}
	for range 9 {
		pulse()
	}
	for i := range 8 {
		pulse()
		if g[Pos{4, 0, 0}] != lit || g[Pos{2, 0, 0}].Item() != counter {
			t.Fatalf("dark after %d pulses down: %v", i+1, g)
		}
	}
	pulse()
	if g[Pos{4, 0, 0}] != lamp || g[Pos{2, 0, 0}] != counter {
		t.Fatalf("back at zero should darken it: %v", g)
	}
	for range 10 {
		pulse()
	}
	use(t, g, Pos{2, 0, 0})
	if g[Pos{4, 0, 0}] != lamp || g[Pos{2, 0, 0}] != counter {
		t.Fatalf("resetting while counting down should darken it: %v", g)
	}
}

func TestNoteBlocksSoundWhilePowered(t *testing.T) {
	g := line(on, world.NoteBlock+4)
	update(t, g, Pos{})
	if g[Pos{1, 0, 0}] != world.NoteBlock+5 {
		t.Fatalf("grid %v", g)
	}
}

func TestTimersBeat(t *testing.T) {
	g := line(world.Timer, wire, lamp)
	beat(t, g, true)
	if g[Pos{}] != timerOn || g[Pos{2, 0, 0}] != lit {
		t.Fatalf("grid %v", g)
	}
	beat(t, g, false)
	if g[Pos{2, 0, 0}] != lamp {
		t.Fatalf("grid %v", g)
	}
}

func TestTeleportersTakeTurnsAlongTheirWire(t *testing.T) {
	g := line(teleport, wire, wire, wire, teleport)
	g[Pos{2, 0, 1}] = teleport
	g[Pos{9, 0, 9}] = teleport
	for from, to := range map[Pos]Pos{{0, 0, 0}: {2, 0, 1}, {2, 0, 1}: {4, 0, 0}, {4, 0, 0}: {0, 0, 0}} {
		if got, ok, err := Partner(g, from); err != nil || !ok || got != to {
			t.Errorf("%v goes to %v, want %v", from, got, to)
		}
	}
	if _, ok, _ := Partner(g, Pos{9, 0, 9}); ok {
		t.Error("a teleporter without wire goes nowhere")
	}
}

func set(t *testing.T, g grid, at Pos, b world.Block) {
	t.Helper()
	if _, err := Set(g, at, b); err != nil {
		t.Fatal(err)
	}
}

func beat(t *testing.T, g grid, on bool) {
	t.Helper()
	if _, err := Beat(g, []Pos{{}}, on); err != nil {
		t.Fatal(err)
	}
}
