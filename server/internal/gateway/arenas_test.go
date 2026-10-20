package gateway

import (
	"testing"

	"github.com/coder/websocket"

	"scabbage/internal/arenas"
	pb "scabbage/internal/pb/scabbage/v1"
)

// arenaCommand makes a zone called Field around a new admin, who then runs cmd.
func arenaCommand(t *testing.T, ts *testServer, cmd, reply string) *websocket.Conn {
	t.Helper()
	admin := ts.joinAdmin(t)
	send(t, admin, sendChat("/zone add Field 8"))
	recvWhere(t, admin, noticeOf("Zones updated"))
	send(t, admin, sendChat(cmd))
	recvWhere(t, admin, noticeOf(reply))
	return admin
}

func field(t *testing.T, ts *testServer) arenas.Arena {
	t.Helper()
	a, ok, err := ts.arenas.Get("Field")
	if err != nil || !ok {
		t.Fatalf("Get = %v, %v", ok, err)
	}
	return a
}

func TestAdminsMarkSpawnsWhereTheyStand(t *testing.T) {
	ts := startServer(t)
	arenaCommand(t, ts, "/arena spawn field red", "Field: last one standing, no time limit, 1 spawn (red), no goals")
	a := field(t, ts)
	if len(a.Spawns) != 1 || a.Spawns[0].Team != arenas.Red {
		t.Fatalf("arena %+v", a)
	}
	z, _ := ts.zones.Named("Field")
	if !z.Contains(a.Spawns[0].X, a.Spawns[0].Z) {
		t.Fatalf("spawn %+v is outside %+v", a.Spawns[0], z)
	}
}

func TestAdminsSetHowArenasAreWon(t *testing.T) {
	ts := startServer(t)
	arenaCommand(t, ts, "/arena win field points 3", "first to 3 points")
	if a := field(t, ts); a.Win != arenas.WinPoints || a.Points != 3 {
		t.Fatalf("arena %+v", a)
	}
}

func TestArenaTimers(t *testing.T) {
	ts := startServer(t)
	arenaCommand(t, ts, "/arena timer field 90", "90 seconds")
	if a := field(t, ts); a.Seconds != 90 {
		t.Fatalf("arena %+v", a)
	}
}

func TestArenasPlayBuiltInGames(t *testing.T) {
	ts := startServer(t)
	arenaCommand(t, ts, "/arena game field race", "Field (race): first to a goal, 300 seconds, no spawns, no goals. It needs a spawn.")
	if a := field(t, ts); a.Game != arenas.Race || a.Win != arenas.WinGoal {
		t.Fatalf("arena %+v", a)
	}
}

func TestReadyArenasSayNothingMissing(t *testing.T) {
	ts := startServer(t)
	arenaCommand(t, ts, "/arena spawn field", "Field: last one standing, no time limit, 1 spawn, no goals")
	if a := field(t, ts); a.Ready() != nil {
		t.Fatalf("arena %+v", a)
	}
}

func TestArenaMistakesAreExplained(t *testing.T) {
	cases := map[string]string{
		"/arena spawn nowhere":     `No zone is called "nowhere"`,
		"/arena spawn field pink":  "Teams are red, blue, green or yellow.",
		"/arena win field fastest": "Arenas are won by last, goal or points",
		"/arena timer field soon":  "Timers run 0 to 3600 seconds.",
		"/arena clear field":       "Field is not an arena.",
		"/arena jump":              "Arenas: /arena spawn",
		"/arena game field chess":  "Games are race, flags, tag, spleef, last or custom.",
		"/arena pvp field maybe":   "Say pvp on or off.",
	}
	for cmd, reply := range cases {
		t.Run(cmd, func(t *testing.T) { arenaCommand(t, startServer(t), cmd, reply) })
	}
}

func TestClearingAnArena(t *testing.T) {
	ts := startServer(t)
	admin := arenaCommand(t, ts, "/arena goal field", "1 goal")
	send(t, admin, sendChat("/arena clear field"))
	recvWhere(t, admin, noticeOf("Field is no longer an arena."))
	if _, ok, _ := ts.arenas.Get("Field"); ok {
		t.Fatal("the arena is still there")
	}
}

func TestArenaCommandsAreForAdmins(t *testing.T) {
	ts := startServer(t)
	_, kid := ts.joinKid(t)
	send(t, kid, sendChat("/arena list"))
	recvWhere(t, kid, noticeOf("Only admins"))
}

func TestEveryoneSeesArenaMarks(t *testing.T) {
	ts := startServer(t)
	admin, kid := ts.joinKid(t)
	zoneCommand(t, admin, kid, "/zone add Field 8", 1)
	send(t, admin, sendChat("/arena goal field blue"))
	marked := func(m *pb.ServerMessage) bool {
		z := m.GetZonesChanged().GetZones()
		return len(z) == 1 && len(z[0].Marks) == 1
	}
	if m := recvWhere(t, kid, marked).GetZonesChanged().Zones[0].Marks[0]; m.Team != "blue" || !m.Goal {
		t.Fatalf("kid sees %v", m)
	}
}
