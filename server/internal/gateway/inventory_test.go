package gateway

import (
	"testing"
	"time"

	"github.com/coder/websocket"

	"scabbage/internal/inventory"
	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/world"
)

func moveItem(from, to uint32) *pb.ClientMessage {
	return &pb.ClientMessage{Body: &pb.ClientMessage_MoveItem{MoveItem: &pb.MoveItem{From: from, To: to}}}
}

func placeFrom(slot uint32, x, y, z int32, b world.Block) *pb.ClientMessage {
	msg := editBlock(x, y, z, b)
	msg.GetEditBlock().Slot = slot
	return msg
}

func isInventory(m *pb.ServerMessage) bool { return m.GetInventory() != nil }

func isItems(m *pb.ServerMessage) bool { return m.GetItemsChanged() != nil }

// besideSpawn puts b dy above the ground next to the spawn point, on stone, and returns where.
func (ts *testServer) besideSpawn(t *testing.T, dy int32, b world.Block) (x, y, z int32) {
	t.Helper()
	sx, sy, sz := ts.spawn()
	x, y, z = int32(sx)+1, int32(sy)+dy, int32(sz)
	for _, set := range []struct {
		y int32
		b world.Block
	}{{int32(sy) - 1, world.Stone}, {y, b}} {
		if err := ts.world.SetBlock(x, set.y, z, set.b); err != nil {
			t.Fatal(err)
		}
	}
	return x, y, z
}

// mineStone breaks stone beside spawn in survival and waits for the cobblestone to be picked up.
func (ts *testServer) mineStone(t *testing.T, c *websocket.Conn) (x, y, z int32) {
	t.Helper()
	return ts.mine(t, c, world.Stone, world.Cobblestone)
}

// mine breaks b beside spawn in survival and waits for what it drops to land in slot 0.
func (ts *testServer) mine(t *testing.T, c *websocket.Conn, b, drop world.Block) (x, y, z int32) {
	t.Helper()
	x, y, z = ts.besideSpawn(t, 0, b)
	send(t, c, editBlock(x, y, z, world.Air))
	inv := recvWhere(t, c, isInventory).GetInventory()
	if s := inv.Slots[0]; s.Block != uint32(drop) || s.Count != 1 {
		t.Fatalf("slot 0 = %v", s)
	}
	return x, y, z
}

func TestWelcomeCarriesTheSavedInventory(t *testing.T) {
	ts := startServer(t)
	_ = ts.joinAdmin(t).Close(websocket.StatusNormalClosure, "")
	var inv inventory.Inventory
	inv[4] = inventory.Stack{Block: world.Log, Count: 9}
	if err := ts.svc.SaveInventory(1, inv.Encode()); err != nil {
		t.Fatal(err)
	}
	c := dial(t, ts.url)
	send(t, c, hello(secret(1), nil))
	w := recv(t, c).GetWelcome()
	if got := w.Inventory.GetSlots(); len(got) != inventory.Slots || got[4].Block != uint32(world.Log) || got[4].Count != 9 {
		t.Fatalf("inventory = %v", w.Inventory)
	}
}

func TestSurvivalBreaksDropWhatTheBlockGives(t *testing.T) {
	ts := startServer(t)
	c := ts.survivalAdmin(t)
	x, y, z := ts.besideSpawn(t, 0, world.Stone)
	send(t, c, editBlock(x, y, z, world.Air))
	expectBlockChanged(t, c, x, y, z, world.Air)
	d := recv(t, c).GetItemsChanged().GetDrops()
	if len(d) != 1 || d[0].Block != uint32(world.Cobblestone) || d[0].Pos.X != float32(x)+0.5 || d[0].Pos.Y != float32(y) {
		t.Fatalf("drops = %v", d)
	}
}

func TestCreativeBreaksDropNothing(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	x, y, z := ts.besideSpawn(t, 0, world.Stone)
	send(t, c, editBlock(x, y, z, world.Air))
	expectBlockChanged(t, c, x, y, z, world.Air)
	expectPong(t, c)
}

func TestNearbyDropsArePickedUpAndRemembered(t *testing.T) {
	ts := startServer(t)
	c := ts.survivalAdmin(t)
	ts.mineStone(t, c)
	_ = c.Close(websocket.StatusNormalClosure, "")
	deadline := time.Now().Add(2 * time.Second)
	for {
		data, _ := ts.svc.Inventory(1)
		if inv, _ := inventory.Decode(data); inv[0].Block == world.Cobblestone {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("the inventory was not saved on leaving")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestPlacingInSurvivalSpendsTheSlot(t *testing.T) {
	ts := startServer(t)
	c := ts.survivalAdmin(t)
	x, y, z := ts.mineStone(t, c)
	send(t, c, placeFrom(0, x, y, z, world.Cobblestone))
	if s := recvWhere(t, c, isInventory).GetInventory().Slots[0]; s.Count != 0 {
		t.Fatalf("slot 0 = %v after placing its only block", s)
	}
	expectBlockChanged(t, c, x, y, z, world.Cobblestone)
	send(t, c, placeFrom(0, x, y+1, z, world.Cobblestone))
	expectBlockChanged(t, c, x, y+1, z, world.Air)
}

func TestSurvivalCannotPlaceWhatItLacks(t *testing.T) {
	ts := startServer(t)
	c := ts.survivalAdmin(t)
	sx, sy, sz := ts.spawn()
	x, y, z := int32(sx)+1, int32(sy), int32(sz)
	send(t, c, placeFrom(0, x, y, z, world.Planks))
	expectBlockChanged(t, c, x, y, z, world.Air)
}

func TestDiggingFasterThanTheBlockAllowsIsRefused(t *testing.T) {
	ts := startServer(t)
	c := ts.survivalAdmin(t)
	ts.besideSpawn(t, 0, world.Stone)
	x, y, z := ts.besideSpawn(t, 1, world.Stone)
	send(t, c, editBlock(x, y-1, z, world.Air))
	send(t, c, editBlock(x, y, z, world.Air))
	recvWhere(t, c, func(m *pb.ServerMessage) bool { return m.GetBlockChanged().GetBlock() == uint32(world.Stone) })
	if b, _ := ts.world.Block(x, y, z); b != world.Stone {
		t.Fatalf("block = %v, want stone", b)
	}
}

func TestMoveItemRearrangesTheInventory(t *testing.T) {
	ts := startServer(t)
	c := ts.survivalAdmin(t)
	ts.mineStone(t, c)
	send(t, c, moveItem(0, 20))
	if s := recvWhere(t, c, isInventory).GetInventory().Slots; s[0].Count != 0 || s[20].Block != uint32(world.Cobblestone) {
		t.Fatalf("slots 0 and 20 = %v, %v", s[0], s[20])
	}
}

func TestJoinersSeeItemsLyingAround(t *testing.T) {
	ts := startServer(t)
	c := ts.survivalAdmin(t)
	ts.besideSpawn(t, 2, world.Stone)
	x, y, z := ts.besideSpawn(t, 3, world.Glass) // rests out of reach of the pickup
	send(t, c, editBlock(x, y, z, world.Air))
	recvWhere(t, c, isItems)
	again := dial(t, ts.url)
	send(t, again, hello(secret(1), nil))
	if d := recvWhere(t, again, isItems).GetItemsChanged().GetDrops(); len(d) != 1 || d[0].Block != uint32(world.Glass) {
		t.Fatalf("drops = %v", d)
	}
}

func TestWelcomeTellsHowLongBlocksTakeToMine(t *testing.T) {
	ts := startServer(t)
	c := dial(t, ts.url)
	code, _, _ := ts.svc.AdminInvite()
	send(t, c, hello(secret(1), &pb.JoinRequest{Code: code, Name: "Mom", Color: 7}))
	stone := recv(t, c).GetWelcome().World.Blocks[world.Stone]
	if stone.MineMs != uint32(world.Info(world.Stone).MineMs) || stone.MineMs == 0 {
		t.Fatalf("stone = %v", stone)
	}
}
