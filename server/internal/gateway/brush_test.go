package gateway

import (
	"testing"

	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/world"
)

func brushEdit(x, y, z, size int32, b world.Block) *pb.ClientMessage {
	msg := editBlock(x, y, z, b)
	msg.GetEditBlock().Size = uint32(size)
	return msg
}

func isBatch(m *pb.ServerMessage) bool { return m.GetBlockBatch() != nil }

// cube fills a cube of size blocks a side with b, from dx east of spawn at foot height.
func (ts *testServer) cube(t *testing.T, dx, size int32, b world.Block) (x, y, z int32) {
	t.Helper()
	sx, sy, sz := ts.spawn()
	x, y, z = int32(sx)+dx, int32(sy), int32(sz)-1
	for i := range size {
		for j := range size {
			for k := range size {
				if err := ts.world.SetBlock(x+i, y+j, z+k, b); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	return x, y, z
}

func TestSurvivalBrushDigsAWholeCubeIntoOnePile(t *testing.T) {
	ts := startServer(t)
	c := ts.survivalAdmin(t)
	x, y, z := ts.cube(t, 2, 3, world.Stone)
	send(t, c, brushEdit(x, y, z, 3, world.Air))
	if got := recvWhere(t, c, isBatch).GetBlockBatch().GetBlocks(); len(got) != 27 {
		t.Fatalf("batch of %d blocks, want 27", len(got))
	}
	if d := recvWhere(t, c, isItems).GetItemsChanged().GetDrops(); len(d) != 1 {
		t.Fatalf("drops = %v, want one pile", d)
	}
	if s := recvWhere(t, c, isInventory).GetInventory().Slots[0]; s.Block != uint32(world.Cobblestone) || s.Count != 27 {
		t.Fatalf("slot 0 = %v, want 27 cobblestone", s)
	}
}

func TestBrushPlacesAroundBodiesAndCorrectsTheRest(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	x, y, z := ts.cube(t, 1, 3, world.Air) // the player stands in its west side
	send(t, c, brushEdit(x, y, z, 3, world.Planks))
	counts := map[uint32]int{}
	for range 2 { // what everyone sees placed, and the sender's corrections, in either order
		for _, b := range recvWhere(t, c, isBatch).GetBlockBatch().GetBlocks() {
			counts[b.Block]++
		}
	}
	placed, refused := counts[uint32(world.Planks)], counts[uint32(world.Air)]
	if placed+refused != 27 || placed == 0 || refused == 0 {
		t.Fatalf("placed %d, refused %d", placed, refused)
	}
}
