package maps

import (
	"math"
	"testing"

	"scabbage/internal/motion"
	"scabbage/internal/world"
)

const ground = 40

func sampleAt(m Map) func(x, y, z int32) world.Block {
	a := m.Area(0, 0)
	return func(x, y, z int32) world.Block { return m.Blocks[a.Index(x, y, z)] }
}

func TestSampleIsAPlayableMap(t *testing.T) {
	m := Sample(ground)
	a := m.Area(0, 0)
	if len(m.Blocks) != a.Volume() || m.Start == nil {
		t.Fatalf("%d blocks for %d, start %v", len(m.Blocks), a.Volume(), m.Start)
	}
	at := sampleAt(m)
	sx, sy, sz := int32(math.Floor(m.Start.X)), int32(m.Start.Y), int32(math.Floor(m.Start.Z))
	if !world.Info(at(sx, sy-1, sz)).Solid || !roomToStand(at, sx, sy, sz) {
		t.Fatal("the start isn't standing room on the ground")
	}
	if len(m.Scripts) < 3 {
		t.Fatalf("%d scripts", len(m.Scripts))
	}
	for _, s := range m.Scripts {
		if at(s.X, s.Y, s.Z) != world.ScriptBlock {
			t.Errorf("script at %v has no script block", s.Pos)
		}
	}
	counts := map[world.Block]int{}
	for _, b := range m.Blocks {
		counts[b]++
	}
	for _, want := range []world.Block{world.Checkpoint, world.Chest, world.Switch, world.Wire} {
		if counts[want] == 0 {
			t.Errorf("no %v", want)
		}
	}
	if at(0, 0, 0) != world.Bedrock || at(5, ground, 5) != world.Grass {
		t.Fatal("the ground isn't bedrock under grass")
	}
	data, err := Encode(m)
	if err != nil {
		t.Fatal(err)
	}
	if _, unknown, err := Decode(data); err != nil || unknown != 0 {
		t.Fatalf("decode: %v, %d unknown", err, unknown)
	}
}

func TestSampleIsBuiltInMeters(t *testing.T) {
	m := Sample(ground)
	if m.Width != 24*world.Meter || m.Depth != 16*world.Meter {
		t.Fatalf("%d×%d blocks", m.Width, m.Depth)
	}
	at := sampleAt(m)
	// The house is entered through the gate's doorway, a meter wide and two tall.
	inside := cellCenter(19, 8)
	if !roomToStand(at, inside[0], ground+1, inside[1]) {
		t.Fatal("no room to stand in the house")
	}
	wall := int32(16 * world.Meter)
	for y := int32(ground + 1); y < ground+1+2*world.Meter; y++ {
		if at(wall, y, cellCenter(0, 7)[1]) != world.Cobblestone {
			t.Fatalf("the doorway is open at y %d before the switch is on", y)
		}
	}
	// The yard wall is too high to jump.
	if at(0, ground+2*world.Meter, 0) != world.Brick {
		t.Fatal("the yard wall is lower than two meters")
	}
}

func TestSampleWireJoinsTheSwitchToTheGate(t *testing.T) {
	m := Sample(ground)
	at := sampleAt(m)
	gate := m.Scripts[1].Pos
	seen := map[world.Pos]bool{}
	var walk func(p world.Pos) bool
	walk = func(p world.Pos) bool {
		for _, d := range []world.Pos{{X: 1}, {X: -1}, {Z: 1}, {Z: -1}, {Y: 1}, {Y: -1}} {
			n := world.Pos{X: p.X + d.X, Y: p.Y + d.Y, Z: p.Z + d.Z}
			if n.X < 0 || n.Z < 0 || n.X >= m.Width || n.Z >= m.Depth || seen[n] {
				continue
			}
			seen[n] = true
			switch at(n.X, n.Y, n.Z) {
			case world.Switch:
				return true
			case world.Wire:
				if walk(n) {
					return true
				}
			}
		}
		return false
	}
	if !walk(gate) {
		t.Fatal("no wire runs from the gate script to the switch")
	}
}

// roomToStand reports whether a player fits with their feet in block x, y, z.
func roomToStand(at func(x, y, z int32) world.Block, x, y, z int32) bool {
	for dy := int32(0); dy < int32(math.Ceil(motion.Height)); dy++ {
		if world.Info(at(x, y+dy, z)).Solid {
			return false
		}
	}
	return true
}

func cellCenter(mx, mz int32) [2]int32 {
	return [2]int32{mx*world.Meter + world.Meter/2, mz*world.Meter + world.Meter/2}
}
