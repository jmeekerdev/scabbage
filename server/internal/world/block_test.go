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
