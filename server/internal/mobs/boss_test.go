package mobs

import (
	"testing"
	"time"

	"scabbage/internal/world"
)

func TestSummonWakesOneBossPerPlace(t *testing.T) {
	h := New(1, false)
	m, ok := h.Summon(Golem, at(0.5, 10, 0.5), field{}, noon)
	if !ok || m.Kind != Golem || m.Health() != m.MaxHealth() || !m.Boss() {
		t.Fatalf("summon = %+v, %v", m, ok)
	}
	if _, ok := h.Summon(KingSlime, at(10, 10, 10), field{}, noon); ok {
		t.Fatal("a second boss woke beside the first")
	}
	if _, ok := h.Summon(KingSlime, at(100, 10, 100), field{}, noon); !ok {
		t.Fatal("a boss far away should wake")
	}
	if _, ok := h.Summon(Slime, at(-200, 10, -200), field{}, noon); ok {
		t.Fatal("slimes aren't bosses")
	}
}

func TestBossesBiteHardAndStayByDay(t *testing.T) {
	h := New(1, false)
	h.Summon(Golem, at(0.5, 10, 0.5), field{}, noon)
	players := []Player{{Pos: at(4.5, 10, 0.5), Prey: true}}
	_, c := steps(h, noon, 100, players, field{})
	if len(c.Bitten) == 0 || c.Bitten[0].Damage != kinds[Golem].damage {
		t.Fatalf("bites %+v", c.Bitten)
	}
	if len(c.Gone) > 0 || h.Len() != 1 {
		t.Fatal("the golem melted in daylight")
	}
}

func TestGolemsStandTheirGroundWhenHit(t *testing.T) {
	h := New(1, false)
	m, _ := h.Summon(Golem, at(0.5, 10, 0.5), field{}, noon)
	hit, ok := h.Hit(m.ID, at(2.5, 11.5, 0.5), noon)
	if !ok || hit.Mob.Health() != m.MaxHealth()-1 {
		t.Fatalf("hit = %+v, %v", hit, ok)
	}
	if h.mobs[0].fleeing {
		t.Error("the golem ran from a hit")
	}
}

func TestKingSlimesShedSlimesAndPopWithLoot(t *testing.T) {
	h := New(1, false)
	m, _ := h.Summon(KingSlime, at(0.5, 10, 0.5), field{}, noon)
	now, shed := noon, 0
	for range m.MaxHealth() - 1 {
		now = now.Add(time.Second)
		hit, _ := h.Hit(m.ID, at(2, 11, 0.5), now)
		shed += len(hit.Spawned)
		h.mobs[0].Pos = at(0.5, 10, 0.5) // keep it in reach
	}
	if shed != (m.MaxHealth()-1)/shedEvery {
		t.Errorf("shed %d slimes", shed)
	}
	last, _ := h.Hit(m.ID, at(2, 11, 0.5), now.Add(time.Second))
	if !last.Gone || !last.Mob.Boss() {
		t.Fatalf("last hit = %+v", last)
	}
}

func TestBossesAreHitFromFurtherAway(t *testing.T) {
	h := New(1, false)
	m, _ := h.Summon(KingSlime, at(0.5, 10, 0.5), field{}, noon)
	if _, ok := h.Hit(m.ID, at(6.3, 11.6, 0.5), noon); !ok {
		t.Fatal("a king slime's side should be in reach")
	}
}

func TestBossesWakeClearOfTheWallBehindTheirAltar(t *testing.T) {
	g := field{}
	for y := int32(10); y <= 30; y++ {
		for z := int32(-10); z <= 10; z++ {
			g[[3]int32{2, y, z}] = world.Cobblestone // the wall just east of the altar
		}
	}
	for _, k := range []Kind{KingSlime, Golem} {
		m, ok := New(1, false).Summon(k, at(1.5, 10, 0.5), g, noon)
		if !ok || m.Pos.X+kinds[k].halfWidth > 2 || m.Pos.Y != 10 {
			t.Fatalf("%v woke at %+v, inside the wall", k, m.Pos)
		}
	}
}
