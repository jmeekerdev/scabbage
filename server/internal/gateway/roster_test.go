package gateway

import (
	"testing"

	"github.com/coder/websocket"

	pb "scabbage/internal/pb/scabbage/v1"
)

// admitKid admits a second player, leaving the roster messages unread.
func (ts *testServer) admitKid(t *testing.T) (admin, kid *websocket.Conn) {
	t.Helper()
	admin = ts.joinAdmin(t)
	kid = ts.requestJoin(t, secret(2), "Kid")
	list := recv(t, admin).GetPendingRequests()
	send(t, admin, decide(list.Requests[0].Id, true))
	if recv(t, kid).GetWelcome() == nil {
		t.Fatal("expected Welcome")
	}
	waitPlayers(t, ts.hub, 2)
	return admin, kid
}

func recvJoined(t *testing.T, c *websocket.Conn) *pb.PlayersJoined {
	t.Helper()
	msg := recv(t, c)
	if j := msg.GetPlayersJoined(); j != nil {
		return j
	}
	t.Fatalf("got %v, want PlayersJoined", msg)
	return nil
}

// adminSeesJoin reads the admin's PlayersJoined, emptied pending list and new player list, which race.
func adminSeesJoin(t *testing.T, admin *websocket.Conn) *pb.PlayersJoined {
	t.Helper()
	var joined *pb.PlayersJoined
	for range 3 {
		msg := recv(t, admin)
		if j := msg.GetPlayersJoined(); j != nil {
			joined = j
		} else if msg.GetPendingRequests() == nil && msg.GetPlayerList() == nil {
			t.Fatalf("got %v, want PlayersJoined, PendingRequests or PlayerList", msg)
		}
	}
	if joined == nil {
		t.Fatal("no PlayersJoined")
	}
	return joined
}

func onlyPlayer(t *testing.T, j *pb.PlayersJoined, name string) *pb.OtherPlayer {
	t.Helper()
	if len(j.Players) != 1 || j.Players[0].Name != name {
		t.Fatalf("joined = %v, want just %s", j.Players, name)
	}
	return j.Players[0]
}

func TestJoiningPlayersSeeEachOther(t *testing.T) {
	ts := startServer(t)
	admin, kid := ts.admitKid(t)
	x, y, z := ts.spawn()

	mom := onlyPlayer(t, recvJoined(t, kid), "Mom")
	if mom.Color != 7 || mom.Pose.Pos.X != x || mom.Pose.Pos.Y != y || mom.Pose.Pos.Z != z {
		t.Fatalf("mom = %v", mom)
	}
	k := onlyPlayer(t, adminSeesJoin(t, admin), "Kid")
	if k.Entity == mom.Entity {
		t.Fatalf("entities collide: %d", k.Entity)
	}
}

func TestMovesReachOthersOnTheNextTick(t *testing.T) {
	ts := startServer(t)
	admin, kid := ts.joinKid(t)
	x, y, z := ts.spawn()

	m := move(x+0.25, y, z, 0)
	m.GetMove().Yaw = 1.5
	send(t, admin, m)
	moved := recv(t, kid).GetPlayersMoved()
	if moved == nil || len(moved.Poses) != 1 {
		t.Fatalf("moved = %v", moved)
	}
	if p := moved.Poses[0]; p.Pos.X != x+0.25 || p.Yaw != 1.5 {
		t.Fatalf("pose = %v", p)
	}
	expectPong(t, admin) // never told about its own move
}

func TestLeavingIsSentToOthers(t *testing.T) {
	ts := startServer(t)
	admin, kid := ts.admitKid(t)
	k := onlyPlayer(t, adminSeesJoin(t, admin), "Kid")
	recvJoined(t, kid)

	_ = kid.Close(websocket.StatusNormalClosure, "")
	if left := recv(t, admin).GetPlayerLeft(); left == nil || left.Entity != k.Entity {
		t.Fatalf("left = %v", left)
	}
}
