package gateway

import (
	"testing"

	"github.com/coder/websocket"

	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/vitals"
)

func hitPlayer(entity uint32) *pb.ClientMessage {
	return &pb.ClientMessage{Body: &pb.ClientMessage_HitPlayer{HitPlayer: &pb.HitPlayer{Entity: entity}}}
}

func hurtByPlayer(m *pb.ServerMessage) bool { return m.GetVitals().GetHurt() == pb.Harm_HARM_PLAYER }

// arenaAtSpawn puts the admin and a kid in a Field zone at spawn, set up by the admin's commands;
// it returns the kid's entity.
func arenaAtSpawn(t *testing.T, ts *testServer, cmds ...string) (admin, kid *websocket.Conn, entity uint32) {
	t.Helper()
	admin, kid = ts.admitKid(t)
	entity = adminSeesJoin(t, admin).Players[0].Entity
	recvJoined(t, kid)
	zoneCommand(t, admin, kid, "/zone add Field 8", 1)
	for _, cmd := range cmds {
		zoneCommand(t, admin, kid, cmd, 1)
	}
	return admin, kid, entity
}

// hitAndSettle has the admin hit someone and waits until the kid has read everything it caused.
func hitAndSettle(t *testing.T, admin, kid *websocket.Conn, entity uint32) (hurt bool) {
	t.Helper()
	send(t, admin, hitPlayer(entity))
	send(t, admin, ping(1))
	recvWhere(t, admin, func(m *pb.ServerMessage) bool { return m.GetPong() != nil })
	send(t, kid, ping(2))
	m := recvWhere(t, kid, func(m *pb.ServerMessage) bool { return m.GetPong() != nil || hurtByPlayer(m) })
	return hurtByPlayer(m)
}

func TestFightingGamesLetSurvivalPlayersHitEachOther(t *testing.T) {
	ts := startServer(t)
	admin, kid, entity := arenaAtSpawn(t, ts, "/zone mode field survival", "/arena game field tag")
	if !hitAndSettle(t, admin, kid, entity) {
		t.Fatal("the kid was not hurt")
	}
}

func TestHitsHurtAHeart(t *testing.T) {
	ts := startServer(t)
	admin, kid, entity := arenaAtSpawn(t, ts, "/zone mode field survival", "/arena game field last")
	send(t, admin, hitPlayer(entity))
	if v := recvWhere(t, kid, hurtByPlayer).GetVitals(); v.Health != vitals.Max-playerDamage {
		t.Fatalf("vitals = %v", v)
	}
}

func TestNoHittingOutsideFightingGames(t *testing.T) {
	cases := map[string][]string{
		"plain zone":   {"/zone mode field survival"},
		"race":         {"/zone mode field survival", "/arena game field race"},
		"creative tag": {"/zone mode field creative", "/arena game field tag"},
		"opted out":    {"/zone mode field survival", "/arena game field tag", "/arena pvp field off"},
	}
	for name, cmds := range cases {
		t.Run(name, func(t *testing.T) {
			ts := startServer(t)
			admin, kid, entity := arenaAtSpawn(t, ts, cmds...)
			if hitAndSettle(t, admin, kid, entity) {
				t.Fatal("the kid was hurt")
			}
		})
	}
}

func TestCustomArenasOptIn(t *testing.T) {
	ts := startServer(t)
	admin, kid, entity := arenaAtSpawn(t, ts, "/zone mode field survival", "/arena pvp field on")
	if !hitAndSettle(t, admin, kid, entity) {
		t.Fatal("the kid was not hurt")
	}
}

func TestPlayersSeeWhichZonesAllowHitting(t *testing.T) {
	ts := startServer(t)
	admin, kid, _ := arenaAtSpawn(t, ts)
	if _, z := zoneCommand(t, admin, kid, "/arena game field flags", 1); !z.Pvp {
		t.Fatalf("kid sees %v", z)
	}
}
