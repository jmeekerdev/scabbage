package gateway

import (
	"testing"

	"github.com/coder/websocket"

	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/world"
)

// sessionOf finds the in-world session of the player called name.
func (ts *testServer) sessionOf(t *testing.T, name string) *session {
	t.Helper()
	ts.gw.mu.Lock()
	defer ts.gw.mu.Unlock()
	for s := range ts.gw.players {
		if s.player.Name == name {
			return s
		}
	}
	t.Fatalf("%s isn't in the world", name)
	return nil
}

// walkOnto steps a player from a /tp east along x in half blocks, to x.
func walkOnto(t *testing.T, c *websocket.Conn, tp *pb.Teleport, x float32) {
	t.Helper()
	for at := tp.Pos.X + 0.5; at <= x; at += 0.5 {
		send(t, c, move(at, tp.Pos.Y, tp.Pos.Z, tp.Id))
	}
}

func isTeleportTo(x, y, z float32) func(*pb.ServerMessage) bool {
	return func(m *pb.ServerMessage) bool {
		p := m.GetTeleport().GetPos()
		return p != nil && p.X == x && p.Y == y && p.Z == z
	}
}

// checkpointFloor builds a stone floor high up from x 10 to 12 with a checkpoint at its east end.
func checkpointFloor(t *testing.T, ts *testServer) {
	t.Helper()
	for x := int32(10); x <= 12; x++ {
		setBlock(t, ts, x, 299, 10, world.Stone)
	}
	setBlock(t, ts, 12, 300, 10, world.Checkpoint)
}

func TestFaintedPlayersComeBackToTheirCheckpoint(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	checkpointFloor(t, ts)
	walkOnto(t, c, tpAndWait(t, c, "10 300 10"), 12.5)
	recvWhere(t, c, noticeOf("Checkpoint!"))

	tpAndWait(t, c, "10 300 10")
	ts.sessionOf(t, "Mom").faint()
	recvWhere(t, c, isTeleportTo(12.5, 300, 10.5))

	setBlock(t, ts, 12, 300, 10, world.Air) // a broken checkpoint sends them to the spawn
	ts.sessionOf(t, "Mom").faint()
	x, y, z := ts.spawn()
	recvWhere(t, c, isTeleportTo(x, y, z))
}

func TestCheckpointsOnlyCountInTheirOwnZone(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	checkpointFloor(t, ts)
	walkOnto(t, c, tpAndWait(t, c, "10 300 10"), 12.5)
	recvWhere(t, c, noticeOf("Checkpoint!"))
	tpAndWait(t, c, "40 300 40")
	send(t, c, sendChat("/zone add Far 8"))
	recvWhere(t, c, noticeOf("Zones updated"))
	ts.sessionOf(t, "Mom").faint()
	x, y, z := ts.spawn()
	recvWhere(t, c, isTeleportTo(x, y, z))
}

func TestCheckpointsAreSaved(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	checkpointFloor(t, ts)
	walkOnto(t, c, tpAndWait(t, c, "10 300 10"), 12.5)
	recvWhere(t, c, noticeOf("Checkpoint!"))
	s := ts.sessionOf(t, "Mom")
	ts.gw.mu.Lock()
	saved := ts.gw.players[s].snapshot().record()
	ts.gw.mu.Unlock()
	if cp := saved.Checkpoint; cp == nil || *cp != [3]int32{12, 300, 10} {
		t.Fatalf("saved checkpoint %v", cp)
	}
}
