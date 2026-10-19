package gateway

import (
	"testing"

	"github.com/coder/websocket"

	pb "scabbage/internal/pb/scabbage/v1"
)

func move(x, y, z float32, teleport uint32) *pb.ClientMessage {
	return &pb.ClientMessage{Body: &pb.ClientMessage_Move{Move: &pb.Move{
		Pos: &pb.Vec3{X: x, Y: y, Z: z}, Teleport: teleport,
	}}}
}

func (ts *testServer) spawn() (x, y, z float32) {
	sx, sy, sz := ts.world.Spawn()
	return float32(sx), float32(sy), float32(sz)
}

// expectPong sends a ping and fails if anything else arrives first.
func expectPong(t *testing.T, c *websocket.Conn) {
	t.Helper()
	send(t, c, ping(1))
	if msg := recv(t, c); msg.GetPong() == nil {
		t.Fatalf("got %v, want Pong", msg)
	}
}

func TestImpossibleMoveTeleportsBackToLastGoodPosition(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	x, y, z := ts.spawn()
	send(t, c, move(x+0.5, y+0.5, z, 0))
	expectPong(t, c)

	send(t, c, move(x+500, y, z, 0))
	tp := recv(t, c).GetTeleport()
	if tp == nil || tp.Id != 1 || tp.Pos.X != x+0.5 || tp.Pos.Y != y+0.5 {
		t.Fatalf("teleport = %v", tp)
	}
}

func TestMovesFromBeforeATeleportAreIgnored(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	x, y, z := ts.spawn()
	send(t, c, move(x, y-5, z, 0)) // inside the ground
	if tp := recv(t, c).GetTeleport(); tp == nil || tp.Id != 1 {
		t.Fatalf("teleport = %v", tp)
	}
	send(t, c, move(x, y-5, z, 0))
	expectPong(t, c)

	send(t, c, move(x, y-5, z, 1))
	if tp := recv(t, c).GetTeleport(); tp == nil || tp.Id != 2 {
		t.Fatalf("teleport = %v", tp)
	}
}

func TestConsoleCannotMove(t *testing.T) {
	ts := startServer(t)
	c := ts.joinConsole(t)
	send(t, c, move(0, 0, 0, 0))
	expectDisconnect(t, c, pb.DisconnectCode_DISCONNECT_CODE_PROTOCOL)
}
