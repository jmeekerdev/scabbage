package mobs

import (
	"math"
	"testing"
	"time"

	"scabbage/internal/motion"
	"scabbage/internal/world"
)

const minute = int64(time.Minute / time.Millisecond)

var (
	noon     = time.UnixMilli(5 * minute)
	midnight = time.UnixMilli(15 * minute)
)

// field is grass at y=9 over stone, plus any blocks set on it.
type field map[[3]int32]world.Block

func (f field) Block(x, y, z int32) (world.Block, error) {
	if b, ok := f[[3]int32{x, y, z}]; ok {
		return b, nil
	}
	switch {
	case y == 9:
		return world.Grass, nil
	case y < 9:
		return world.Stone, nil
	}
	return world.Air, nil
}

func (f field) Boxes(x, y, z int32) []world.Box {
	b, _ := f.Block(x, y, z)
	return world.Info(b).Boxes()
}

func steps(h *Herd, from time.Time, n int, players []Player, g Ground) (time.Time, Changes) {
	var all Changes
	now := from
	for range n {
		now = now.Add(Every)
		c := h.Step(now, players, g)
		all.Moved = append(all.Moved, c.Moved...)
		all.Gone = append(all.Gone, c.Gone...)
		all.Bitten = append(all.Bitten, c.Bitten...)
	}
	return now, all
}

func at(x, y, z float64) motion.Vec { return motion.Vec{X: x, Y: y, Z: z} }

func TestNight(t *testing.T) {
	if Night(noon) || !Night(midnight) {
		t.Fatalf("Night(noon)=%v Night(midnight)=%v", Night(noon), Night(midnight))
	}
}

func TestSheepWanderAndStayOnTheGround(t *testing.T) {
	h := New(1, true)
	h.Add(Sheep, at(0.5, 10, 0.5), noon)
	players := []Player{{Pos: at(3, 10, 3)}}
	_, c := steps(h, noon, 200, players, field{})
	if len(c.Moved) == 0 {
		t.Fatal("the sheep never moved")
	}
	for _, m := range h.All() {
		if m.Pos.Y != 10 {
			t.Fatalf("sheep left the ground: %+v", m.Pos)
		}
	}
}

func TestMobsStandOnTheHighestGroundUnderTheirBody(t *testing.T) {
	g := field{}
	for x := int32(2); x <= 30; x++ {
		for z := int32(-30); z <= 30; z++ {
			g[[3]int32{x, 10, z}] = world.Stone // a ledge one block high
		}
	}
	h := New(1, true)
	h.Add(Sheep, at(1.5, 10, 0.5), noon) // straddling the ledge's edge
	_, c := steps(h, noon, 400, []Player{{Pos: at(2, 11, 3)}}, g)
	w := kinds[Sheep].halfWidth
	for _, m := range c.Moved {
		onLedge := cell(m.Pos.X+w) >= 2 && m.Pos.X-w < 30 && m.Pos.Z-w > -30 && m.Pos.Z+w < 30
		if onLedge && m.Pos.Y < 11 {
			t.Fatalf("sheep at %+v has legs inside the ledge", m.Pos)
		}
	}
}

func TestMobsClimbOneMeterButNotTwo(t *testing.T) {
	g := field{}
	for x := int32(-8); x <= 8; x++ {
		for z := int32(4); z <= 12; z++ {
			top := int32(9 + meter) // a one-meter step...
			if z == 12 {
				top += 2 * meter // ...then a wall two meters above it
			}
			for y := int32(10); y <= top; y++ {
				g[[3]int32{x, y, z}] = world.Stone
			}
		}
	}
	h := New(1, true)
	h.Add(Slime, at(0.5, 10, 0.5), midnight)
	prey := []Player{{Pos: at(0.5, 10+3*meter, 20.5), Prey: true}}
	_, c := steps(h, midnight, 100, prey, g)
	climbed := false
	for _, m := range c.Moved {
		climbed = climbed || m.Pos.Y == 10+meter
	}
	m := h.All()[0]
	if !climbed || m.Pos.Y != 10+meter || m.Pos.Z < 4 || m.Pos.Z > 12 {
		t.Fatalf("climbed=%v, slime at %+v, want over the step and stopped by the wall", climbed, m.Pos)
	}
}

func TestMobsKeepOutOfWater(t *testing.T) {
	g := field{}
	for x := int32(-5); x <= 5; x++ {
		g[[3]int32{x, 9, 2}] = world.Water
	}
	h := New(1, true)
	h.Add(Slime, at(0.5, 10, 0.5), midnight)
	steps(h, midnight, 100, []Player{{Pos: at(0.5, 10, 5.5), Prey: true}}, g)
	if z := h.All()[0].Pos.Z; z >= 2 {
		t.Fatalf("slime walked into the water at z=%v", z)
	}
}

func TestSlimesChaseAndBiteOnlyPrey(t *testing.T) {
	h := New(1, true)
	h.Add(Slime, at(0.5, 10, 0.5), midnight)
	players := []Player{{Pos: at(-6, 10, 0.5)}, {Pos: at(6.5, 10, 0.5), Prey: true}}
	_, c := steps(h, midnight, 60, players, field{})
	if len(c.Bitten) == 0 {
		t.Fatal("the slime never bit")
	}
	for _, b := range c.Bitten {
		if b.Player != 1 || b.Damage != 2 {
			t.Fatalf("bite %+v: player 1 is the prey and slimes bite for 2", b)
		}
	}
	if len(c.Bitten) > 4 {
		t.Fatalf("bit %d times in 6 s", len(c.Bitten))
	}
}

func TestShearingDropsWoolOnceAndItRegrows(t *testing.T) {
	h := New(1, true)
	m := h.Add(Sheep, at(0.5, 10, 0.5), noon)
	eye := at(2, 11.6, 0.5)
	got, ok := h.Hit(m.ID, eye, noon)
	if !ok || got.Drop != world.WhiteWool || !got.Mob.Sheared || !got.Mob.Hurt || got.Gone {
		t.Fatalf("first hit = %+v, %v", got, ok)
	}
	if again, _ := h.Hit(m.ID, eye, noon); again.Drop != world.Air {
		t.Fatalf("sheared sheep dropped %v", again.Drop)
	}
	players := []Player{{Pos: at(2, 10, 0.5)}}
	_, c := steps(h, noon, int(Regrow/Every)+1, players, field{})
	if h.All()[0].Sheared || len(c.Moved) == 0 {
		t.Fatal("wool never grew back")
	}
}

func TestSheepRunFromWhoeverHitThem(t *testing.T) {
	h := New(1, true)
	m := h.Add(Sheep, at(0.5, 10, 0.5), noon)
	h.Hit(m.ID, at(-1, 11.6, 0.5), noon)
	steps(h, noon, 10, []Player{{Pos: at(-1, 10, 0.5)}}, field{})
	if x := h.All()[0].Pos.X; x < 1.5 {
		t.Fatalf("sheep only got to x=%v", x)
	}
}

func TestSlimesPopAfterThreeHits(t *testing.T) {
	h := New(1, true)
	m := h.Add(Slime, at(0.5, 10, 0.5), midnight)
	eye := at(2, 11.6, 0.5)
	for i := range 3 {
		got, ok := h.Hit(m.ID, eye, midnight)
		if !ok || got.Gone != (i == 2) {
			t.Fatalf("hit %d = %+v, %v", i, got, ok)
		}
	}
	if h.Len() != 0 {
		t.Fatal("slime still there")
	}
}

func TestHitsNeedReach(t *testing.T) {
	h := New(1, true)
	m := h.Add(Sheep, at(0.5, 10, 0.5), noon)
	if _, ok := h.Hit(m.ID, at(10*meter, 11.6, 0.5), noon); ok {
		t.Fatal("hit from 10 m away")
	}
	if _, ok := h.Hit(m.ID+1, at(1, 11.6, 0.5), noon); ok {
		t.Fatal("hit a mob that isn't there")
	}
}

func TestSheepSpawnOnGrassByDayAndSlimesAtNight(t *testing.T) {
	players := []Player{{Pos: at(0.5, 10, 0.5), Prey: true}}
	for _, tc := range []struct {
		name string
		now  time.Time
		want Kind
	}{{"day", noon, Sheep}, {"night", midnight, Slime}} {
		h := New(1, true)
		_, c := steps(h, tc.now, 100, players, field{})
		if h.Len() == 0 {
			t.Fatalf("%s: nothing spawned", tc.name)
		}
		seen := map[uint32]bool{}
		for _, m := range c.Moved {
			if seen[m.ID] {
				continue
			}
			seen[m.ID] = true
			d := math.Hypot(m.Pos.X-0.5, m.Pos.Z-0.5)
			if m.Kind != tc.want || m.Pos.Y != 10 || d < near {
				t.Fatalf("%s: spawned %+v", tc.name, m)
			}
		}
	}
}

func TestSpawningStopsAtTheCap(t *testing.T) {
	h := New(1, true)
	steps(h, midnight, 2000, []Player{{Pos: at(0.5, 10, 0.5), Prey: true}}, field{})
	if h.Len() > monstersPerPrey {
		t.Fatalf("%d slimes for one player", h.Len())
	}
	h = New(1, true)
	steps(h, midnight, 2000, []Player{{Pos: at(0.5, 10, 0.5)}}, field{})
	if h.Len() != 0 {
		t.Fatalf("%d slimes spawned with no one in survival", h.Len())
	}
}

func TestMobsLeaveWhenFarAndSlimesMeltByDay(t *testing.T) {
	h := New(1, true)
	far := h.Add(Sheep, at(200, 10, 0.5), noon)
	slime := h.Add(Slime, at(3, 10, 0.5), noon)
	_, c := steps(h, noon, 200, []Player{{Pos: at(0.5, 10, 0.5)}}, field{})
	gone := map[uint32]bool{}
	for _, id := range c.Gone {
		gone[id] = true
	}
	if !gone[far.ID] || !gone[slime.ID] {
		t.Fatalf("gone = %v", c.Gone)
	}
}
