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
	admin = ts.joinAdmin(t)
	kid = ts.requestJoin(t, secret(2), "Kid")
	list := recv(t, admin).GetPendingRequests()
	send(t, admin, decide(list.Requests[0].Id, true))
	recv(t, kid)   // welcome
	recv(t, admin) // empty pending list
	waitPlayers(t, ts.hub, 2)
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
	_ = ts.joinAdmin(t).Close(websocket.StatusNormalClosure, "")
	waitPlayers(t, ts.hub, 0)

	c := dial(t, ts.url)
	msg := hello(secret(1), nil)
	msg.GetHello().Console = true
	send(t, c, msg)
	recv(t, c) // welcome
	recv(t, c) // pending list
	send(t, c, editBlock(0, 0, 0, world.Air))
	expectDisconnect(t, c, pb.DisconnectCode_DISCONNECT_CODE_PROTOCOL)
}
