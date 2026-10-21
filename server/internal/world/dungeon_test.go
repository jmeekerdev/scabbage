package world

import (
	"bytes"
	"testing"
)

// firstDungeon finds a dungeon near the middle of seed 42's world.
func firstDungeon(t *testing.T, g *GenV2) dungeon {
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

func TestGenV2KeepsV1TerrainAwayFromDungeons(t *testing.T) {
	v1, v2 := NewGenV1(42), NewGenV2(42)
	same := 0
	for x := int32(-8); x < 8; x++ {
		for z := int32(-8); z < 8; z++ {
			p := ChunkPos{x, 2, z}
			if _, ok := v2.dungeonIn(regionOf(x*Size), regionOf(z*Size)); ok {
				continue
			}
			if !bytes.Equal(Encode(v1.Generate(p)), Encode(v2.Generate(p))) {
				t.Fatalf("chunk %v changed though its region has no dungeon", p)
			}
			same++
		}
	}
	if same == 0 {
		t.Fatal("every region had a dungeon")
	}
}

func TestDungeonsStayClearOfSpawn(t *testing.T) {
	g := NewGenV2(42)
	for _, r := range [][2]int32{{-1, -1}, {-1, 0}, {0, -1}, {0, 0}} {
		if _, ok := g.dungeonIn(r[0], r[1]); ok {
			t.Errorf("dungeon in region %v, by spawn", r)
		}
	}
}

func TestDungeonRoomHoldsChestsAndAnAltar(t *testing.T) {
	g := NewGenV2(42)
	d := firstDungeon(t, g)
	at := cachedBlocks(g)
	chests, altars := 0, 0
	for x := d.x0; x < d.x0+roomOuter; x++ {
		for z := d.z0; z < d.z0+roomOuter; z++ {
			for y := d.floor; y <= d.floor+roomHeight+1; y++ {
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
	if b := at(d.x0+roomOuter/2, d.floor+2, d.z0+roomOuter/2); b != Air {
		t.Errorf("room middle = %v", b)
	}
	if b := at(d.x0, d.floor+roomHeight+1, d.z0); b != Cobblestone {
		t.Errorf("room corner = %v", b)
	}
}

// Walking up the stairs from the door must reach open sky with headroom all the way.
func TestDungeonStairsReachTheSurface(t *testing.T) {
	g := NewGenV2(42)
	d := firstDungeon(t, g)
	at := cachedBlocks(g)
	z := d.z0 + roomOuter/2
	for i := int32(0); ; i++ {
		x, y := d.x0-1-i, d.floor+1+i
		if at(x, y, z) != StoneStairs+3 {
			t.Fatalf("step %d at %d,%d,%d = %v", i, x, y, z, at(x, y, z))
		}
		for h := int32(1); h <= 3; h++ {
			if b := at(x, y+h, z); b != Air {
				t.Fatalf("step %d blocked at height %d by %v", i, h, b)
			}
		}
		if y >= g.SurfaceY(x, z) {
			if at(x, y+4, z) != Cobblestone {
				t.Error("no arch over the way in")
			}
			return
		}
	}
}
