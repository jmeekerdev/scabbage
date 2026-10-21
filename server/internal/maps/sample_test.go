package maps

import (
	"math"
	"testing"

	"scabbage/internal/world"
)

func TestSampleIsAPlayableMap(t *testing.T) {
	const ground = 40
	m := Sample(ground)
	a := m.Area(0, 0)
	if len(m.Blocks) != a.Volume() || m.Start == nil {
		t.Fatalf("%d blocks for %d, start %v", len(m.Blocks), a.Volume(), m.Start)
	}
	at := func(x, y, z int32) world.Block { return m.Blocks[a.Index(x, y, z)] }
	sx, sy, sz := int32(math.Floor(m.Start.X)), int32(m.Start.Y), int32(math.Floor(m.Start.Z))
	if !world.Info(at(sx, sy-1, sz)).Solid || at(sx, sy, sz) != world.Air || at(sx, sy+1, sz) != world.Air {
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
