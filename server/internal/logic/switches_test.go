package logic

import (
	"testing"

	"scabbage/internal/world"
)

func timerOf(beats int) world.Block {
	for i, b := range world.TimerBeats {
		if b == beats {
			return world.Timers + world.Block(2*i)
		}
	}
	return world.Timer
}

func TestTimersBeatAtTheirSpeed(t *testing.T) {
	for _, beats := range []int{1, 2, 4, 8, 16} {
		g := line(timerOf(beats), wire, lamp)
		for q := range int64(40) {
			if _, err := Beat(g, []Pos{{}}, q); err != nil {
				t.Fatal(err)
			}
			if want := (q/int64(beats))%2 == 0; isOn(g[Pos{}]) != want || (g[Pos{2, 0, 0}] == lit) != want {
				t.Fatalf("a %d-quarter timer at quarter %d: %v", beats, q, g)
			}
		}
	}
}

func TestTappingATimerChangesItsSpeed(t *testing.T) {
	g := line(timerOf(1) + 1)
	for _, beats := range []int{2, 4, 8, 16, 1} {
		if !use(t, g, Pos{}) || g[Pos{}] != timerOf(beats)+1 {
			t.Fatalf("want a %d-quarter timer still on, got %v", beats, g)
		}
	}
}

func TestTripwiresGivePowerWhileCrossed(t *testing.T) {
	g := line(world.Tripwire, wire, lamp)
	set(t, g, Pos{}, world.Tripwire+1)
	if g[Pos{2, 0, 0}] != lit {
		t.Fatalf("grid %v", g)
	}
	set(t, g, Pos{}, world.Tripwire)
	if g[Pos{2, 0, 0}] != lamp {
		t.Fatalf("grid %v", g)
	}
}

func TestNewSwitchesAreInvolved(t *testing.T) {
	for _, b := range []world.Block{world.Tripwire, world.Timers + 3} {
		if !Involves(b) {
			t.Errorf("%v is not involved", b)
		}
	}
}
