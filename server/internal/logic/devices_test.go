package logic

import (
	"maps"
	"testing"

	"scabbage/internal/world"
)

const (
	stickyEast    = world.StickyPiston + 2
	stickyEastOn  = world.StickyPiston + 3
	trapdoor      = world.Trapdoor
	trapdoorOpen  = world.Trapdoor + 1
	fenceGate     = world.FenceGate
	fenceGateOpen = world.FenceGate + 1
	launchPad     = world.LaunchPad
)

func display(n int) world.Block { return world.NumberDisplay + world.Block(2*n) }

func TestAStickyPistonPullsItsBlockBackWhenPowerStops(t *testing.T) {
	g := line(on, stickyEast, somewhere)
	update(t, g, Pos{})
	if g[Pos{1, 0, 0}] != stickyEastOn || g[Pos{2, 0, 0}] != world.Air || g[Pos{3, 0, 0}] != somewhere {
		t.Fatalf("pushed %v", g)
	}
	set(t, g, Pos{}, world.Switch)
	if g[Pos{1, 0, 0}] != stickyEast || g[Pos{2, 0, 0}] != somewhere || g[Pos{3, 0, 0}] != world.Air {
		t.Fatalf("pulled %v", g)
	}
}

func TestAStickyPistonPullsOnlyWhatItCould(t *testing.T) {
	cases := map[string]grid{
		"bedrock":   line(world.Switch, stickyEastOn, world.Air, world.Bedrock),
		"blocked":   line(world.Switch, stickyEastOn, world.Dirt, somewhere),
		"logic":     line(world.Switch, stickyEastOn, world.Air, lamp),
		"protected": {{fence - 3, 0, 0}: world.Switch, {fence - 2, 0, 0}: stickyEastOn, {fence, 0, 0}: somewhere},
	}
	for name, g := range cases {
		before := maps.Clone(g)
		update(t, g, Pos{})
		update(t, g, Pos{fence - 3, 0, 0})
		for p, b := range before {
			if b.Item() != world.StickyPiston && g[p] != b {
				t.Errorf("%s: pulled %v", name, g)
			}
		}
	}
}

func TestTrapdoorsOpenWithTheTrapdoorsBesideThem(t *testing.T) {
	g := grid{{0, 0, 0}: trapdoor, {1, 0, 0}: trapdoor, {1, 0, 1}: trapdoor, {0, 1, 0}: trapdoor}
	if !use(t, g, Pos{}) {
		t.Fatal("trapdoors are usable")
	}
	for _, p := range []Pos{{0, 0, 0}, {1, 0, 0}, {1, 0, 1}} {
		if g[p] != trapdoorOpen {
			t.Fatalf("%v stayed shut: %v", p, g)
		}
	}
	if g[Pos{0, 1, 0}] != trapdoor {
		t.Fatalf("a trapdoor above opened: %v", g)
	}
}

func TestPowerOpensTrapdoorsAndTheyWaitForPeople(t *testing.T) {
	g := line(on, wire, trapdoor, trapdoor)
	update(t, g, Pos{})
	if g[Pos{2, 0, 0}] != trapdoorOpen || g[Pos{3, 0, 0}] != trapdoorOpen {
		t.Fatalf("grid %v", g)
	}
	w := crowded{g, map[Pos]bool{{3, 0, 0}: true}}
	g[Pos{}] = world.Switch
	if _, err := Update(w, Pos{}); err != nil || g[Pos{2, 0, 0}] != trapdoorOpen {
		t.Fatalf("closed on someone: %v %v", g, err)
	}
	update(t, g, Pos{})
	if g[Pos{2, 0, 0}] != trapdoor || g[Pos{3, 0, 0}] != trapdoor {
		t.Fatalf("grid %v", g)
	}
}

func TestFenceGatesOpenWithEveryGateJoinedToThem(t *testing.T) {
	g := grid{{0, 0, 0}: fenceGate, {0, 1, 0}: fenceGate, {1, 1, 0}: fenceGate, {0, 0, 1}: trapdoor}
	use(t, g, Pos{})
	if g[Pos{0, 1, 0}] != fenceGateOpen || g[Pos{1, 1, 0}] != fenceGateOpen || g[Pos{0, 0, 1}] != trapdoor {
		t.Fatalf("grid %v", g)
	}
}

func TestHugeTrapdoorFloorsOpenOnlyNearby(t *testing.T) {
	g := grid{}
	for x := range int32(MaxPanel + 10) {
		g[Pos{x, 0, 0}] = trapdoor
	}
	use(t, g, Pos{})
	if g[Pos{MaxPanel + 5, 0, 0}] != trapdoor {
		t.Fatal("opened past the limit")
	}
}

func TestLaunchPadsFollowPower(t *testing.T) {
	g := line(on, wire, launchPad+8)
	update(t, g, Pos{})
	if g[Pos{2, 0, 0}] != launchPad+9 {
		t.Fatalf("grid %v", g)
	}
	set(t, g, Pos{}, world.Switch)
	if g[Pos{2, 0, 0}] != launchPad+8 {
		t.Fatalf("grid %v", g)
	}
}

func TestANumberDisplayCountsItsPulses(t *testing.T) {
	g := line(world.Switch, wire, display(0), lamp)
	for range 3 {
		set(t, g, Pos{}, on)
		set(t, g, Pos{}, world.Switch)
	}
	if g[Pos{2, 0, 0}] != display(3) || g[Pos{3, 0, 0}] != lamp {
		t.Fatalf("grid %v", g)
	}
	for range 7 {
		set(t, g, Pos{}, on)
		set(t, g, Pos{}, world.Switch)
	}
	if g[Pos{2, 0, 0}] != display(0) {
		t.Fatalf("ten pulses should wrap to 0: %v", g)
	}
}

func TestTappingANumberDisplayCountsUp(t *testing.T) {
	g := line(display(9) + 1)
	if !use(t, g, Pos{}) || g[Pos{}] != display(0)+1 {
		t.Fatalf("grid %v", g)
	}
}

func TestANumberDisplayShowsTheCounterBesideIt(t *testing.T) {
	g := line(world.Switch, wire, counter, display(0))
	pulse := func(n int) {
		for range n {
			set(t, g, Pos{}, on)
			set(t, g, Pos{}, world.Switch)
		}
	}
	pulse(4)
	if g[Pos{3, 0, 0}] != display(4) {
		t.Fatalf("counting up: %v", g)
	}
	pulse(7)
	if g[Pos{3, 0, 0}].Item() != world.NumberDisplay || state(g[Pos{3, 0, 0}]) != 7 {
		t.Fatalf("counting down: %v", g)
	}
	use(t, g, Pos{2, 0, 0})
	if g[Pos{3, 0, 0}] != display(0) {
		t.Fatalf("reset: %v", g)
	}
}

func TestANewNumberDisplayShowsTheCounterAlreadyThere(t *testing.T) {
	g := line(counter+2*5, display(0))
	update(t, g, Pos{1, 0, 0})
	if g[Pos{1, 0, 0}] != display(5) {
		t.Fatalf("grid %v", g)
	}
}

func TestNewDevicesAreInvolved(t *testing.T) {
	for _, b := range []world.Block{stickyEast, trapdoor, fenceGateOpen, launchPad + 3, display(4)} {
		if !Involves(b) {
			t.Errorf("%v is not involved", b)
		}
	}
}
