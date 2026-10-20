package gateway

import (
	"testing"

	"github.com/coder/websocket"

	"scabbage/internal/access"
	"scabbage/internal/inventory"
	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/world"
)

func flyTo(x, y, z float32) *pb.ClientMessage {
	m := move(x, y, z, 0)
	m.GetMove().Flying, m.GetMove().Yaw = true, 0.5
	return m
}

// rejoin closes the admin's connection once it is saved and returns the welcome on coming back.
func (ts *testServer) rejoin(t *testing.T, c *websocket.Conn) *pb.Welcome {
	t.Helper()
	_ = c.Close(websocket.StatusNormalClosure, "")
	waitPlayers(t, ts.hub, 0)
	c = dial(t, ts.url)
	send(t, c, hello(secret(1), nil))
	return recv(t, c).GetWelcome()
}

func (ts *testServer) saved(t *testing.T) access.Save {
	t.Helper()
	v, err := ts.svc.Saved(1)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestRejoiningStartsWhereYouLeft(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	x, y, z := ts.spawn()
	send(t, c, flyTo(x+1, y+2, z))
	expectPong(t, c)
	s := ts.rejoin(t, c).GetStart()
	if s.GetId() != 0 || s.GetPos().X != x+1 || s.GetPos().Y != y+2 || !s.GetRestore().Flying || s.GetRestore().Yaw != 0.5 {
		t.Fatalf("start = %v", s)
	}
}

func TestASaveInsideBlocksStartsAtTheSpawn(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	x, y, z := ts.world.Spawn()
	_ = c.Close(websocket.StatusNormalClosure, "")
	waitPlayers(t, ts.hub, 0)
	if err := ts.svc.Store(1, access.Save{Placed: true, X: x, Y: y - 5, Z: z, Health: 7}); err != nil {
		t.Fatal(err)
	}
	c = dial(t, ts.url)
	send(t, c, hello(secret(1), nil))
	if w := recv(t, c).GetWelcome(); w.Start != nil || w.Vitals.Health != 7 {
		t.Fatalf("welcome start %v, vitals %v", w.Start, w.Vitals)
	}
}

func TestFaintedSavesComeBackHealthy(t *testing.T) {
	ts := startServer(t)
	_ = ts.joinAdmin(t).Close(websocket.StatusNormalClosure, "")
	waitPlayers(t, ts.hub, 0)
	if err := ts.svc.Store(1, access.Save{Health: 0}); err != nil {
		t.Fatal(err)
	}
	c := dial(t, ts.url)
	send(t, c, hello(secret(1), nil))
	if w := recv(t, c).GetWelcome(); w.Vitals.Health != 20 {
		t.Fatalf("vitals %v", w.Vitals)
	}
}

// inRoundFlying steps the admin onto a linked race portal while flying.
func inRoundFlying(t *testing.T, ts *testServer) (admin *websocket.Conn, x, y, z float32) {
	t.Helper()
	admin, x, y, z = raceWithPortal(t, ts)
	adminSays(t, admin, "/arena portal field")
	send(t, admin, flyTo(x+1, y, z))
	send(t, admin, flyTo(x, y, z))
	waitFor(t, admin, scoreboard)
	return admin, x, y, z
}

func TestGamesSaveYouOnTheWayIn(t *testing.T) {
	ts := startServer(t)
	_, x, _, _ := inRoundFlying(t, ts)
	if v := ts.saved(t); !v.Placed || float32(v.X) != x || !v.Flying {
		t.Fatalf("saved %+v", v)
	}
}

func TestLeavingAGameRestoresYourSave(t *testing.T) {
	ts := startServer(t)
	admin, x, y, z := inRoundFlying(t, ts)
	send(t, admin, move(x+1, y, z, 0))
	ts.gw.mu.Lock()
	for _, a := range ts.gw.players {
		a.inv[0] = inventory.Stack{Block: world.Log, Count: 5}
	}
	ts.gw.mu.Unlock()

	send(t, admin, sendChat("/leave"))
	var tp *pb.Teleport
	var inv *pb.Inventory
	waitFor(t, admin, func(m *pb.ServerMessage) bool {
		tp = firstSeen(tp, m.GetTeleport())
		inv = firstSeen(inv, m.GetInventory())
		return tp != nil && inv != nil
	})
	if tp.Pos.X != x || !tp.GetRestore().Flying || tp.GetRestore().Yaw != 0.5 || inv.Slots[0].Count != 0 {
		t.Fatalf("teleport %v, inventory %v", tp, inv.Slots[0])
	}
}

func TestLeavingTheWorldMidGameKeepsTheSaveFromBefore(t *testing.T) {
	ts := startServer(t)
	admin, x, y, z := inRoundFlying(t, ts)
	send(t, admin, move(x+1, y, z, 0))
	expectPong(t, admin)
	if s := ts.rejoin(t, admin).GetStart(); s.GetPos().X != x || !s.GetRestore().Flying {
		t.Fatalf("start = %v", s)
	}
}

// firstSeen keeps the first non-nil value seen.
func firstSeen[T any](have, got *T) *T {
	if have != nil {
		return have
	}
	return got
}
