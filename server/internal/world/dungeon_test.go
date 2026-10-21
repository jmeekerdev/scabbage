package world

import "testing"

// firstDungeon finds a dungeon near the middle of seed 42's world.
func firstDungeon(t *testing.T, g *GenV3) dungeon {
	t.Helper()
	for rx := int32(-4); rx < 4; rx++ {
		for rz := int32(-4); rz < 4; rz++ {
			if d, ok := g.dungeonIn(rx, rz); ok {
				return d
			}
		}
	}
	t.Fatal("no dungeon in 64 regions")
	return dungeon{}
}

func TestDungeonsStayClearOfSpawn(t *testing.T) {
	g := NewGenV3(42)
	for _, r := range [][2]int32{{-1, -1}, {-1, 0}, {0, -1}, {0, 0}} {
		if _, ok := g.dungeonIn(r[0], r[1]); ok {
			t.Errorf("dungeon in region %v, by spawn", r)
		}
	}
}

func TestDungeonRoomHoldsChestsAndAnAltar(t *testing.T) {
	g := NewGenV3(42)
	d := firstDungeon(t, g)
	at := cachedBlocks(g)
	chests, altars := 0, 0
	for x := d.x0; x < d.x0+roomOuter; x++ {
		for z := d.z0; z < d.z0+roomOuter; z++ {
			for y := d.floor - wall; y <= d.top(); y++ {
				switch at(x, y, z) {
				case Chest:
					chests++
				case BossAltar:
					altars++
				}
			}
		}
	}
	if chests != 2 || altars != 1 {
		t.Fatalf("room has %d chests and %d altars", chests, altars)
	}
	if b := at(d.x0+roomOuter/2, d.floor+roomHeight/2, d.z0+roomOuter/2); b != Air {
		t.Errorf("room middle = %v", b)
	}
	if b := at(d.x0, d.top(), d.z0); b != Cobblestone {
		t.Errorf("room corner = %v", b)
	}
}

// Walking up the stairs from the door must reach open sky with headroom all the way.
func TestDungeonStairsReachTheSurface(t *testing.T) {
	g := NewGenV3(42)
	d := firstDungeon(t, g)
	at := cachedBlocks(g)
	z := d.door()
	for y := d.floor; y < d.floor+headroom; y++ {
		if b := at(d.x0, y, z); b != Air {
			t.Fatalf("doorway blocked at %d by %v", y, b)
		}
	}
	for i := int32(0); ; i++ {
		x, y := d.x0-1-i, d.floor+i
		if at(x, y, z) != StoneStairs+3 {
			t.Fatalf("step %d at %d,%d,%d = %v", i, x, y, z, at(x, y, z))
		}
		for h := int32(1); h <= headroom; h++ {
			if b := at(x, y+h, z); b != Air {
				t.Fatalf("step %d blocked at height %d by %v", i, h, b)
			}
		}
		if y >= g.SurfaceY(x, z) {
			if at(x, y+headroom+1, z) != Cobblestone {
				t.Error("no arch over the way in")
			}
			return
		}
	}
}
