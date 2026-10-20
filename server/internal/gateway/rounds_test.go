package gateway

import (
	"testing"

	"github.com/coder/websocket"

	"scabbage/internal/arenas"
	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/world"
)

func scoreboard(m *pb.ServerMessage) bool { return m.GetScoreboard() != nil }

// adminSays runs an admin command and waits for its reply.
func adminSays(t *testing.T, admin *websocket.Conn, cmd string) string {
	t.Helper()
	send(t, admin, sendChat(cmd))
	return recvWhere(t, admin, func(m *pb.ServerMessage) bool { return m.GetChat().GetNotice() }).GetChat().Text
}

// raceWithPortal makes a race in Field around spawn and puts a portal block just west of spawn,
// where the admin stands; it returns the portal's middle.
func raceWithPortal(t *testing.T, ts *testServer) (admin *websocket.Conn, x, y, z float32) {
	t.Helper()
	admin = ts.joinAdmin(t)
	sx, sy, sz := ts.spawn()
	px, py, pz := int32(sx)-1, int32(sy), int32(sz)
	if err := ts.world.SetBlock(px, py, pz, world.Portal); err != nil {
		t.Fatal(err)
	}
	a := arenas.Arena{Spawns: []arenas.Mark{{X: px + 3, Y: py, Z: pz}}, Goals: []arenas.Mark{{X: px + 5, Y: py, Z: pz}}}
	a.Play(arenas.Race)
	if err := ts.zones.Put("Field", px-8, pz-8, px+8, pz+8); err != nil {
		t.Fatal(err)
	}
	if err := ts.arenas.Put("Field", a); err != nil {
		t.Fatal(err)
	}
	ts.gw.zonesChanged()
	send(t, admin, move(sx-1, sy, sz, 0))
	return admin, sx - 1, sy, sz
}

func TestPortalsAreLinkedFromOnTop(t *testing.T) {
	ts := startServer(t)
	admin := ts.joinAdmin(t)
	adminSays(t, admin, "/zone add Field 8")
	if got := adminSays(t, admin, "/arena portal field"); got != "Stand on a portal block first." {
		t.Fatalf("reply %q", got)
	}
}

func TestAdminsLinkPortalsTheyStandOn(t *testing.T) {
	ts := startServer(t)
	admin, _, _, _ := raceWithPortal(t, ts)
	if got := adminSays(t, admin, "/arena portal field"); got != "Field (race): first to a goal, 300 seconds, 1 spawn, 1 goal, 1 portal" {
		t.Fatalf("reply %q", got)
	}
}

func TestSteppingOnAPortalJoinsTheRound(t *testing.T) {
	ts := startServer(t)
	admin, x, y, z := raceWithPortal(t, ts)
	adminSays(t, admin, "/arena portal field")
	send(t, admin, move(x+1, y, z, 0))
	send(t, admin, move(x, y, z, 0))
	b := waitFor(t, admin, scoreboard).GetScoreboard()
	if b.Zone != "Field" || b.Status != "Starting" || b.Seconds != 10 || len(b.Scores) != 1 || b.Scores[0].Name != "Mom" {
		t.Fatalf("board %v", b)
	}

	send(t, admin, sendChat("/leave"))
	var tp *pb.Teleport
	var hidden bool
	waitFor(t, admin, func(m *pb.ServerMessage) bool {
		if m.GetTeleport() != nil {
			tp = m.GetTeleport()
		}
		hidden = hidden || scoreboard(m) && m.GetScoreboard().Zone == ""
		return tp != nil && hidden
	})
	if tp.Pos.X != x || tp.Pos.Y != y || tp.Pos.Z != z {
		t.Fatalf("teleport %v", tp)
	}
}

func TestAnyoneCanLeave(t *testing.T) {
	ts := startServer(t)
	_, kid := ts.admitKid(t)
	send(t, kid, sendChat("/leave"))
	recvWhere(t, kid, noticeOf("You aren't in a game."))
}
