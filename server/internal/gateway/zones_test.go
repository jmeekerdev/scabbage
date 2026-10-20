package gateway

import (
	"testing"

	"github.com/coder/websocket"

	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/world"
)

func zonesWith(n int) func(*pb.ServerMessage) bool {
	return func(m *pb.ServerMessage) bool {
		return m.GetZonesChanged() != nil && len(m.GetZonesChanged().Zones) == n
	}
}

// zoneCommand runs an admin's /zone command and waits until both players see the new zones.
func zoneCommand(t *testing.T, admin, kid *websocket.Conn, cmd string, zones int) (adminSees, kidSees *pb.Zone) {
	t.Helper()
	send(t, admin, sendChat(cmd))
	first := func(c *websocket.Conn) *pb.Zone {
		return recvWhere(t, c, zonesWith(zones)).GetZonesChanged().GetZones()[0]
	}
	if zones == 0 {
		recvWhere(t, admin, zonesWith(0))
		recvWhere(t, kid, zonesWith(0))
		return nil, nil
	}
	return first(admin), first(kid)
}

func TestAdminsMakeZonesThatOnlyBuildersChange(t *testing.T) {
	ts := startServer(t)
	admin, kid := ts.joinKid(t)
	bx, by, bz := ts.besideSpawn(t, 0, world.Stone)

	mine, theirs := zoneCommand(t, admin, kid, "/zone add Home 8", 1)
	if mine.Name != "Home" || mine.X0 != -4 || mine.X1 != 3 || !mine.Build || theirs.Build {
		t.Fatalf("admin sees %v, kid sees %v", mine, theirs)
	}
	send(t, kid, editBlock(bx, by, bz, world.Air))
	recvWhere(t, kid, func(m *pb.ServerMessage) bool { return m.GetBlockChanged().GetBlock() == uint32(world.Stone) })

	if _, theirs = zoneCommand(t, admin, kid, "/zone allow home kid", 1); !theirs.Build {
		t.Fatalf("kid sees %v after being allowed", theirs)
	}
	send(t, kid, editBlock(bx, by, bz, world.Air))
	recvWhere(t, kid, func(m *pb.ServerMessage) bool { return m.GetBlockChanged() != nil })
	if b, _ := ts.world.Block(bx, by, bz); b != world.Air {
		t.Fatalf("block = %v, want air", b)
	}
}

func TestZonesSetTheModeOfPlayersInside(t *testing.T) {
	ts := startServer(t)
	admin, kid := ts.joinKid(t)
	zoneCommand(t, admin, kid, "/zone add Arena 8", 1)
	if _, theirs := zoneCommand(t, admin, kid, "/zone mode arena survival", 1); theirs.Mode != survival {
		t.Fatalf("kid sees %v", theirs)
	}
	if v := recvWhere(t, kid, func(m *pb.ServerMessage) bool { return m.GetVitals() != nil }).GetVitals(); v.Mode != survival {
		t.Fatalf("kid's vitals = %v", v)
	}
	zoneCommand(t, admin, kid, "/zone remove arena", 0)
	if v := recvWhere(t, kid, func(m *pb.ServerMessage) bool { return m.GetVitals() != nil }).GetVitals(); v.Mode != creative {
		t.Fatalf("kid's vitals = %v", v)
	}
}

func TestJoinersSeeTheZones(t *testing.T) {
	ts := startServer(t)
	admin := ts.joinAdmin(t)
	if err := ts.zones.Put("Farm", 0, 0, 9, 9); err != nil {
		t.Fatal(err)
	}
	kid := ts.requestJoin(t, secret(2), "Kid")
	send(t, admin, decide(recv(t, admin).GetPendingRequests().Requests[0].Id, true))
	if z := recvWhere(t, kid, zonesWith(1)).GetZonesChanged().Zones[0]; z.Name != "Farm" || z.Build {
		t.Fatalf("kid sees %v", z)
	}
}

func TestZoneCommandsAreForAdmins(t *testing.T) {
	ts := startServer(t)
	_, kid := ts.joinKid(t)
	send(t, kid, sendChat("/zone add Mine 8"))
	recvWhere(t, kid, noticeOf("Only admins"))
	if len(ts.zones.All()) != 0 {
		t.Fatal("kid made a zone")
	}
}
