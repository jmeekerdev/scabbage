package logic

import (
	"testing"

	"scabbage/internal/world"
)

// delayEast waits n quarter seconds, n ticks of a ticked grid.
func delayEast(n int) world.Block { return world.Delay + world.Block(12*(n-1)+2) }

const relayEast = world.Relay + 2

func transmitter(channel int) world.Block { return world.Transmitter + world.Block(2*channel) }
func receiver(channel int) world.Block    { return world.Receiver + world.Block(2*channel) }

// ticks runs n ticks of w.
func (w *ticked) ticks(t *testing.T, n int) {
	t.Helper()
	for range n {
		w.tick(t)
	}
}

func TestADelayPassesPowerAfterItsSetting(t *testing.T) {
	w := newTicked(line(on, delayEast(3), lamp))
	w.update(t, Pos{0, 0, 0})
	w.ticks(t, 2)
	if w.grid[Pos{2, 0, 0}] != lamp {
		t.Fatalf("lit early: %v", w.grid)
	}
	w.tick(t)
	if w.grid[Pos{1, 0, 0}] != delayEast(3)+1 || w.grid[Pos{2, 0, 0}] != lit {
		t.Fatalf("after its delay: %v", w.grid)
	}
}

func TestADelayStretchesAShortPulseToItsSetting(t *testing.T) {
	w := newTicked(line(on, delayEast(4), lamp))
	w.update(t, Pos{0, 0, 0})
	w.tick(t)
	w.grid[Pos{0, 0, 0}] = world.Switch
	w.update(t, Pos{0, 0, 0})
	w.ticks(t, 3)
	if w.grid[Pos{2, 0, 0}] != lit {
		t.Fatalf("the pulse was lost: %v", w.grid)
	}
	w.ticks(t, 3)
	if w.grid[Pos{2, 0, 0}] != lit {
		t.Fatalf("the pulse was cut short: %v", w.grid)
	}
	w.tick(t)
	if w.grid[Pos{2, 0, 0}] != lamp || len(w.due) != 0 {
		t.Fatalf("still lit: %v, due %v", w.grid, w.due)
	}
}

func TestTappingADelayCyclesItsSettingFacingTheSameWay(t *testing.T) {
	g := line(delayEast(world.Delays))
	if _, used, err := Use(g, Pos{0, 0, 0}); err != nil || !used || g[Pos{0, 0, 0}] != delayEast(1) {
		t.Fatalf("used=%v err=%v: %v", used, err, g)
	}
	if _, _, err := Use(g, Pos{0, 0, 0}); err != nil || g[Pos{0, 0, 0}] != delayEast(2) {
		t.Fatalf("err=%v: %v", err, g)
	}
}

func TestARelayPassesPowerOnlyWhileASideIsOn(t *testing.T) {
	g := line(on, relayEast, lamp)
	g[Pos{1, 0, 1}] = world.Switch
	w := newTicked(g)
	w.update(t, Pos{0, 0, 0})
	w.tick(t)
	if g[Pos{2, 0, 0}] != lamp {
		t.Fatalf("passed power with its side off: %v", g)
	}
	g[Pos{1, 0, 1}] = on
	w.update(t, Pos{1, 0, 1})
	w.tick(t)
	if g[Pos{1, 0, 0}] != relayEast+1 || g[Pos{2, 0, 0}] != lit {
		t.Fatalf("with its side on: %v", g)
	}
	g[Pos{0, 0, 0}] = world.Switch
	w.update(t, Pos{0, 0, 0})
	w.tick(t)
	if g[Pos{2, 0, 0}] != lamp {
		t.Fatalf("with its back off: %v", g)
	}
}

func TestATransmitterPowersReceiversOnItsChannel(t *testing.T) {
	g := line(on, transmitter(3))
	g[Pos{10, 0, 0}], g[Pos{11, 0, 0}] = receiver(3), lamp
	g[Pos{10, 0, 5}], g[Pos{11, 0, 5}] = receiver(4), lamp
	w := newTicked(g)
	w.update(t, Pos{0, 0, 0})
	if g[Pos{1, 0, 0}] != transmitter(3)+1 || g[Pos{10, 0, 0}] != receiver(3) {
		t.Fatalf("in the update: %v", g)
	}
	w.tick(t)
	if g[Pos{10, 0, 0}] != receiver(3)+1 || g[Pos{11, 0, 0}] != lit || g[Pos{11, 0, 5}] != lamp {
		t.Fatalf("after a tick: %v", g)
	}
	g[Pos{0, 0, 0}] = world.Switch
	w.update(t, Pos{0, 0, 0})
	w.tick(t)
	if g[Pos{10, 0, 0}] != receiver(3) || g[Pos{11, 0, 0}] != lamp {
		t.Fatalf("after switching off: %v", g)
	}
}

func TestAnyTransmitterOnAChannelKeepsItOn(t *testing.T) {
	g := line(on, transmitter(0), world.Air, transmitter(0))
	g[Pos{10, 0, 0}] = receiver(0)
	w := newTicked(g)
	w.update(t, Pos{0, 0, 0})
	w.tick(t)
	if g[Pos{10, 0, 0}] != receiver(0)+1 {
		t.Fatalf("one of two transmitters on: %v", g)
	}
}

func TestTappingWirelessBlocksCyclesTheirChannel(t *testing.T) {
	g := line(on, transmitter(world.Channels-1), world.Air, receiver(world.Channels-2))
	w := newTicked(g)
	w.update(t, Pos{0, 0, 0})
	if _, _, err := Use(w, Pos{1, 0, 0}); err != nil || g[Pos{1, 0, 0}] != transmitter(0)+1 {
		t.Fatalf("err=%v: %v", err, g)
	}
	for range 2 {
		if _, _, err := Use(w, Pos{3, 0, 0}); err != nil {
			t.Fatal(err)
		}
	}
	if g[Pos{3, 0, 0}] != receiver(0) {
		t.Fatalf("receiver tapped twice from 14: %v", g)
	}
	w.tick(t)
	if g[Pos{3, 0, 0}] != receiver(0)+1 {
		t.Fatalf("receiver on a lit channel: %v", g)
	}
}
