package gateway

import (
	"testing"

	"github.com/coder/websocket"

	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/world"
)

func editBlock(x, y, z int32, b world.Block) *pb.ClientMessage {
	return &pb.ClientMessage{Body: &pb.ClientMessage_EditBlock{EditBlock: &pb.EditBlock{
		Pos: &pb.BlockPos{X: x, Y: y, Z: z}, Block: uint32(b),
	}}}
}

func expectBlockChanged(t *testing.T, c *websocket.Conn, x, y, z int32, want world.Block) {
	t.Helper()
	ch := recv(t, c).GetBlockChanged()
	if ch == nil || ch.Pos.X != x || ch.Pos.Y != y || ch.Pos.Z != z || ch.Block != uint32(want) {
		t.Fatalf("block changed = %v, want %v at %d,%d,%d", ch, want, x, y, z)
	}
}

// joinKid admits a second player and returns both connections with nothing left to read.
func (ts *testServer) joinKid(t *testing.T) (admin, kid *websocket.Conn) {
	t.Helper()
	admin, kid = ts.admitKid(t)
	adminSeesJoin(t, admin)
	recvJoined(t, kid)
	return admin, kid
}

func TestBreakingABlockIsSentToEveryone(t *testing.T) {
	ts := startServer(t)
	admin, kid := ts.joinKid(t)
	x, y, z := ts.spawn()
	bx, by, bz := int32(x)+1, int32(y), int32(z)
	if err := ts.world.SetBlock(bx, by, bz, world.Stone); err != nil {
		t.Fatal(err)
	}

	send(t, admin, editBlock(bx, by, bz, world.Air))
	expectBlockChanged(t, admin, bx, by, bz, world.Air)
	expectBlockChanged(t, kid, bx, by, bz, world.Air)
	if b, _ := ts.world.Block(bx, by, bz); b != world.Air {
		t.Fatalf("block = %v, want air", b)
	}
}

func TestRejectedPlacementCorrectsOnlyTheSender(t *testing.T) {
	ts := startServer(t)
	admin, kid := ts.joinKid(t)
	x, y, z := ts.spawn()
	bx, by, bz := int32(x), int32(y)+1, int32(z) // inside the admin's body

	send(t, admin, editBlock(bx, by, bz, world.Planks))
	expectBlockChanged(t, admin, bx, by, bz, world.Air)
	expectPong(t, kid)
	if b, _ := ts.world.Block(bx, by, bz); b != world.Air {
		t.Fatalf("block = %v, want air", b)
	}
}

func TestConsoleCannotEditBlocks(t *testing.T) {
	ts := startServer(t)
	c := ts.joinConsole(t)
	send(t, c, editBlock(0, 0, 0, world.Air))
	expectDisconnect(t, c, pb.DisconnectCode_DISCONNECT_CODE_PROTOCOL)
}

func TestPlantsGoWithTheGroundUnderThem(t *testing.T) {
	ts := startServer(t)
	admin := ts.joinAdmin(t)
	x, y, z := ts.spawn()
	bx, by, bz := int32(x)+2, int32(y), int32(z)
	for dy, b := range []world.Block{world.Grass, world.RedFlower} {
		if err := ts.world.SetBlock(bx, by+int32(dy), bz, b); err != nil {
			t.Fatal(err)
		}
	}

	send(t, admin, editBlock(bx, by, bz, world.Air))
	batch := recv(t, admin).GetBlockBatch()
	if len(batch.GetBlocks()) != 2 || batch.Blocks[1].Pos.Y != by+1 || batch.Blocks[1].Block != uint32(world.Air) {
		t.Fatalf("batch = %v", batch)
	}
	if b, _ := ts.world.Block(bx, by+1, bz); b != world.Air {
		t.Fatalf("flower left floating: %v", b)
	}
}
