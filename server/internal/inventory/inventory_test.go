package inventory

import (
	"testing"

	"scabbage/internal/world"
)

func TestAddFillsMatchingStacksThenEmptySlots(t *testing.T) {
	var inv Inventory
	inv[3] = Stack{world.Dirt, StackMax - 2}
	if left := inv.Add(world.Dirt, 5); left != 0 {
		t.Fatalf("left %d", left)
	}
	if inv[3].Count != StackMax || inv[0] != (Stack{world.Dirt, 3}) {
		t.Fatalf("got %+v and %+v", inv[3], inv[0])
	}
}

func TestAddReturnsWhatDoesNotFit(t *testing.T) {
	var inv Inventory
	for i := range inv {
		inv[i] = Stack{world.Stone, StackMax}
	}
	inv[7].Count = StackMax - 1
	if left := inv.Add(world.Stone, 3); left != 2 {
		t.Fatalf("left %d, want 2", left)
	}
	if inv.Add(world.Dirt, 1) != 1 {
		t.Fatal("a full inventory took dirt")
	}
}

func TestTakeEmptiesTheSlot(t *testing.T) {
	var inv Inventory
	inv[2] = Stack{world.Glass, 1}
	if inv.Take(2, world.Dirt) || inv.Take(9, world.Glass) || inv.Take(Slots, world.Glass) {
		t.Fatal("took the wrong block or from the wrong slot")
	}
	if !inv.Take(2, world.Glass) || inv[2] != (Stack{}) || inv.Take(2, world.Glass) {
		t.Fatalf("slot after taking its only block: %+v", inv[2])
	}
}

func TestMoveMergesOrSwaps(t *testing.T) {
	var inv Inventory
	inv[0] = Stack{world.Dirt, 60}
	inv[1] = Stack{world.Dirt, 10}
	inv[2] = Stack{world.Log, 1}
	inv.Move(1, 0)
	if inv[0].Count != StackMax || inv[1] != (Stack{world.Dirt, 6}) {
		t.Fatalf("merge gave %+v, %+v", inv[0], inv[1])
	}
	inv.Move(2, 0)
	if inv[0] != (Stack{world.Log, 1}) || inv[2] != (Stack{world.Dirt, StackMax}) {
		t.Fatalf("swap gave %+v, %+v", inv[0], inv[2])
	}
	if inv.Move(0, Slots) || inv.Move(0, 0) {
		t.Fatal("moved out of range or onto itself")
	}
}

func TestEncodeRoundTrips(t *testing.T) {
	var inv Inventory
	inv[0] = Stack{world.Dirt, 3}
	inv[Slots-1] = Stack{world.Torch, 64}
	got, err := Decode(inv.Encode())
	if err != nil || got != inv {
		t.Fatalf("got %v, %v", got, err)
	}
	if empty, err := Decode(nil); err != nil || empty != (Inventory{}) {
		t.Fatal("nothing saved should be an empty inventory")
	}
}

func TestDecodeDropsUnknownBlocks(t *testing.T) {
	var inv Inventory
	inv[4] = Stack{world.FirstCustom + 999, 2}
	got, err := Decode(inv.Encode())
	if err != nil || got[4] != (Stack{}) {
		t.Fatalf("got %+v, %v", got[4], err)
	}
}
