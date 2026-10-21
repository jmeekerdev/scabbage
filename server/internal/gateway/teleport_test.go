package gateway

import (
	"testing"

	pb "scabbage/internal/pb/scabbage/v1"
)

func TestParseTp(t *testing.T) {
	for arg, want := range map[string]tpTarget{
		"Dad":          {name: "Dad"},
		"10 -20":       {x: 10.5, z: -19.5, ground: true},
		"10 64 -20":    {x: 10.5, y: 64, z: -19.5},
		"  3   4   5 ": {x: 3.5, y: 4, z: 5.5},
	} {
		if got, ok := parseTp(arg); !ok || got != want {
			t.Errorf("parseTp(%q) = %+v, %v; want %+v", arg, got, ok, want)
		}
	}
	for _, bad := range []string{"", "1", "1 2 3 4", "a b", "1 x 3", "NaN 1 2"} {
		if _, ok := parseTp(bad); ok {
			t.Errorf("parseTp(%q) should fail", bad)
		}
	}
}

func isTeleport(m *pb.ServerMessage) bool { return m.GetTeleport() != nil }

func TestTpMovesAdminsOnly(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	send(t, c, sendChat("/tp 20 30 -10"))
	if p := recvWhere(t, c, isTeleport).GetTeleport().Pos; p.X != 20.5 || p.Y != 30 || p.Z != -9.5 {
		t.Fatalf("teleported to %v", p)
	}
	send(t, c, sendChat("/tp 3 4"))
	if p := recvWhere(t, c, isTeleport).GetTeleport().Pos; p.X != 3.5 || p.Y < 1 {
		t.Fatalf("teleported to %v", p)
	}
	send(t, c, sendChat("/tp Nobody"))
	recvWhere(t, c, noticeOf("No one called"))
	send(t, c, sendChat("/tp 99999 5 0"))
	recvWhere(t, c, noticeOf("outside the world"))
}

func TestOnlyAdminsTp(t *testing.T) {
	ts := startServer(t)
	_, kid := ts.joinKid(t)
	send(t, kid, sendChat("/tp 1 2 3"))
	recvWhere(t, kid, noticeOf("Only admins"))
}
