package world

import "testing"

func TestRegisterAddsCustomBlock(t *testing.T) {
	id := FirstCustom + 7
	if id.Valid() {
		t.Fatal("unregistered custom block is valid")
	}
	if err := Register(BlockInfo{ID: id, Name: "flower tile", Solid: true}); err != nil {
		t.Fatal(err)
	}
	if !id.Valid() || Info(id).Name != "flower tile" || !Info(id).Solid {
		t.Fatalf("Info = %+v", Info(id))
	}
	last := Blocks()[len(Blocks())-1]
	if last.ID < id {
		t.Fatalf("Blocks() is missing the custom block, ends with %+v", last)
	}
}

func TestRegisterRefusesBuiltinRange(t *testing.T) {
	if err := Register(BlockInfo{ID: Stone, Name: "fake"}); err == nil {
		t.Fatal("overwrote a built-in block")
	}
	if Info(Stone).Name != "stone" {
		t.Fatal("stone changed")
	}
}

func TestBlocksListsCustomInIDOrder(t *testing.T) {
	for _, id := range []Block{FirstCustom + 21, FirstCustom + 20} {
		if err := Register(BlockInfo{ID: id, Name: "b", Solid: true}); err != nil {
			t.Fatal(err)
		}
	}
	all := Blocks()
	for i := 1; i < len(all); i++ {
		if all[i].ID <= all[i-1].ID {
			t.Fatalf("out of order at %d: %v then %v", i, all[i-1].ID, all[i].ID)
		}
	}
}

func TestPaletteKeepsItsIDs(t *testing.T) {
	for id, b := range blocks {
		if b.ID != Block(id) {
			t.Fatalf("block %d has ID %d", id, b.ID)
		}
	}
	pinned := map[Block]string{
		WhiteWool: "white_wool", WhiteWool + 15: "black_wool", Lamp: "lamp", Torch: "torch",
		StoneSlab: "stone_slab", StoneStairs: "stone_stairs", StoneStairs + 15: "brick_stairs_west",
	}
	for id, name := range pinned {
		if Info(id).Name != name {
			t.Errorf("block %d = %q, want %q", id, Info(id).Name, name)
		}
	}
	if len(blocks) != int(StoneStairs)+16 {
		t.Fatalf("%d built-in blocks", len(blocks))
	}
}

func TestStairsFacingsAreConsecutive(t *testing.T) {
	for id := StoneStairs; id < StoneStairs+16; id += 4 {
		for f := range Block(4) {
			if got, want := Info(id+f).Shape, StairsNorth+Shape(f); got != want {
				t.Fatalf("block %v has shape %d, want %d", id+f, got, want)
			}
		}
	}
}

func TestTorchesGlowAndLetPlayersThrough(t *testing.T) {
	torch, lamp := Info(Torch), Info(Lamp)
	if torch.Light == 0 || lamp.Light != MaxLight || torch.Solid || !torch.Transparent {
		t.Fatalf("torch = %+v, lamp = %+v", torch, lamp)
	}
	if !torch.Targetable() || Info(Water).Targetable() || Info(Air).Targetable() {
		t.Fatal("only solid blocks and torches can be aimed at")
	}
	if torch.Boxes() != nil {
		t.Fatal("torches don't collide")
	}
}

func TestShapesCollideWithTheirBoxes(t *testing.T) {
	slab := Info(StoneSlab).Boxes()
	if len(slab) != 1 || slab[0].Max[1] != 0.5 || !Info(StoneSlab).Transparent {
		t.Fatalf("slab = %+v", slab)
	}
	north := Info(StoneStairs).Boxes()
	if len(north) != 2 || north[1] != (Box{Min: [3]float64{0, 0.5, 0}, Max: [3]float64{1, 1, 0.5}}) {
		t.Fatalf("north stairs = %+v", north)
	}
	if Info(Stone).Boxes()[0] != FullBox {
		t.Fatal("stone is a full cube")
	}
}
