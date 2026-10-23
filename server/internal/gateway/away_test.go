package gateway

import (
	"testing"

	pb "scabbage/internal/pb/scabbage/v1"
)

func setAway(away bool) *pb.ClientMessage {
	return &pb.ClientMessage{Body: &pb.ClientMessage_SetAway{SetAway: &pb.SetAway{Away: away}}}
}

func TestOthersSeeWhoIsAway(t *testing.T) {
	ts := startServer(t)
	admin, kid := ts.admitKid(t)
	k := onlyPlayer(t, adminSeesJoin(t, admin), "Kid")
	recvJoined(t, kid)

	for _, away := range []bool{true, false} {
		send(t, kid, setAway(away))
		got := recvWhere(t, admin, func(m *pb.ServerMessage) bool { return m.GetPlayerAway() != nil }).GetPlayerAway()
		if got.Entity != k.Entity || got.Away != away {
			t.Fatalf("away = %v, want %v for entity %d", got, away, k.Entity)
		}
	}
}

func TestNewcomersSeeWhoIsAway(t *testing.T) {
	ts := startServer(t)
	admin := ts.joinAdmin(t)
	send(t, admin, setAway(true))
	send(t, admin, ping(1))
	recvWhere(t, admin, func(m *pb.ServerMessage) bool { return m.GetPong() != nil })

	kid := ts.requestJoin(t, secret(2), "Kid")
	list := recvWhere(t, admin, func(m *pb.ServerMessage) bool { return m.GetPendingRequests() != nil }).GetPendingRequests()
	send(t, admin, decide(list.Requests[0].Id, true))
	recvWhere(t, kid, func(m *pb.ServerMessage) bool { return m.GetWelcome() != nil })
	if mom := onlyPlayer(t, recvJoined(t, kid), "Mom"); !mom.Away {
		t.Fatalf("mom = %v, want away", mom)
	}
}

func TestAwayPlayersCannotBeHit(t *testing.T) {
	ts := startServer(t)
	admin, kid, entity := arenaAtSpawn(t, ts, "/zone mode field survival", "/arena game field last")
	send(t, kid, setAway(true))
	recvWhere(t, admin, func(m *pb.ServerMessage) bool { return m.GetPlayerAway() != nil })
	if hitAndSettle(t, admin, kid, entity) {
		t.Fatal("an away player was hurt")
	}
}

func TestMobsLeaveAwayPlayersBe(t *testing.T) {
	a := &avatar{mode: survival}
	if !a.prey() {
		t.Fatal("a survival player should be prey")
	}
	a.away = true
	if a.prey() {
		t.Fatal("an away player should not be prey")
	}
}

func TestMobsRestWhileEveryoneIsAway(t *testing.T) {
	g := &Gateway{players: map[*session]*avatar{{}: {away: true}, {}: {away: false}}}
	if g.everyoneAwayLocked() {
		t.Fatal("one player is still here")
	}
	for _, a := range g.players {
		a.away = true
	}
	if !g.everyoneAwayLocked() {
		t.Fatal("everyone is away")
	}
}

func TestAgentsSeeWhoIsAway(t *testing.T) {
	ts := startServer(t)
	admin, kid := ts.joinKid(t)
	send(t, kid, setAway(true))
	recvWhere(t, admin, func(m *pb.ServerMessage) bool { return m.GetPlayerAway() != nil })
	for _, p := range ts.gw.onlinePlayers().(map[string][]agentPlayer)["players"] {
		if p.Away != (p.Name == "Kid") {
			t.Fatalf("%s away = %v", p.Name, p.Away)
		}
	}
}
