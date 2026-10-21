package gateway

import (
	"errors"
	"regexp"
	"testing"

	"github.com/coder/websocket"

	"scabbage/internal/world"
)

var approveCode = regexp.MustCompile(`/world approve ([A-Z]{4})`)

// askWipe sends a /world request and returns the approval code every admin is told.
func askWipe(t *testing.T, c *websocket.Conn, cmd string) string {
	t.Helper()
	send(t, c, sendChat(cmd))
	m := approveCode.FindStringSubmatch(recvWhere(t, c, noticeOf("/world approve")).GetChat().GetText())
	if m == nil {
		t.Fatal("no approval code")
	}
	return m[1]
}

// promoteKid makes Kid an admin and reconnects them as one.
func (ts *testServer) promoteKid(t *testing.T, kid *websocket.Conn) *websocket.Conn {
	t.Helper()
	p, found, err := ts.gw.playerNamed("Kid")
	if err != nil || !found {
		t.Fatal("no Kid")
	}
	if _, err := ts.svc.SetAdmin(p.ID, true); err != nil {
		t.Fatal(err)
	}
	_ = kid.Close(websocket.StatusNormalClosure, "")
	c := dial(t, ts.url)
	send(t, c, hello(secret(2), nil))
	if w := recv(t, c).GetWelcome(); w == nil || !w.You.Admin {
		t.Fatalf("welcome %v", w)
	}
	return c
}

func TestWorldResetNeedsAnotherAdmin(t *testing.T) {
	ts := startServer(t)
	mom, kid := ts.joinKid(t)
	dad := ts.promoteKid(t, kid)
	setBlock(t, ts, 1, 300, 1, world.Brick)
	send(t, mom, sendChat("/zone add Home 8"))
	recvWhere(t, mom, noticeOf("Zones updated"))
	code := askWipe(t, mom, "/world reset")
	recvWhere(t, dad, noticeOf("Mom asks to start the world over"))

	send(t, mom, sendChat("/world approve "+code))
	recvWhere(t, mom, noticeOf("Another admin has to approve"))
	send(t, dad, sendChat("/world approve ZZZZ"))
	recvWhere(t, dad, noticeOf("doesn't match"))
	seed := ts.world.Meta().Seed
	send(t, dad, sendChat("/world approve "+code))
	recvWhere(t, mom, noticeOf("The world is starting over"))
	expectClosed(t, mom)
	expectClosed(t, dad)
	blockIs(t, ts, 1, 300, 1, world.Air)
	if _, ok := ts.zones.Named("Home"); ok || ts.world.Meta().Seed != seed {
		t.Fatalf("zone kept %v, seed %d → %d", ok, seed, ts.world.Meta().Seed)
	}
}

func TestTheOnlyAdminApprovesTheirOwnNewWorld(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	code := askWipe(t, c, "/world new 1234")
	send(t, c, sendChat("/world approve "+code))
	recvWhere(t, c, noticeOf("make a new world (seed 1234)"))
	expectClosed(t, c)
	if ts.world.Meta().Seed != 1234 {
		t.Fatalf("seed %d", ts.world.Meta().Seed)
	}
}

func TestWorldWipeStopsIfTheBackupFails(t *testing.T) {
	ts := startServer(t)
	ts.backup = func() error { return errors.New("disk full") }
	c := ts.joinAdmin(t)
	setBlock(t, ts, 1, 300, 1, world.Brick)
	code := askWipe(t, c, "/world reset")
	send(t, c, sendChat("/world approve "+code))
	recvWhere(t, c, noticeOf("backup failed, so nothing changed: disk full"))
	blockIs(t, ts, 1, 300, 1, world.Brick)
}

func TestWorldWipeCanBeDenied(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	code := askWipe(t, c, "/world reset")
	send(t, c, sendChat("/world deny"))
	recvWhere(t, c, noticeOf("Mom stopped the request"))
	send(t, c, sendChat("/world approve "+code))
	recvWhere(t, c, noticeOf("No one is asking"))
}

func TestWorldWipeKeepsInventoriesButNotCustomBlocksOrPlaces(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	tpAndWait(t, c, "30 290 30")
	s := ts.sessionOf(t, "Mom")
	ts.gw.mu.Lock()
	a := ts.gw.players[s]
	a.inv[0].Block, a.inv[0].Count = world.Torch, 5
	a.inv[1].Block, a.inv[1].Count = world.FirstCustom, 2
	a.checkpoint = checkpoint{at: world.Pos{X: 1, Y: 2, Z: 3}, set: true}
	ts.gw.mu.Unlock()
	code := askWipe(t, c, "/world reset")
	send(t, c, sendChat("/world approve "+code))
	expectClosed(t, c)
	waitPlayers(t, ts.hub, 0)
	v, err := ts.svc.Saved(s.player.ID)
	if err != nil {
		t.Fatal(err)
	}
	x, _, z := ts.world.Spawn()
	if v.X != x || v.Z != z || v.Checkpoint != nil {
		t.Fatalf("saved at %v,%v checkpoint %v", v.X, v.Z, v.Checkpoint)
	}
	if v.Inventory[0] != byte(world.Torch) || v.Inventory[2] != 5 || v.Inventory[5] != 0 {
		t.Fatalf("inventory %v", v.Inventory[:6])
	}
}

func TestOnlyAdminsWipeTheWorld(t *testing.T) {
	ts := startServer(t)
	_, kid := ts.joinKid(t)
	send(t, kid, sendChat("/world reset"))
	recvWhere(t, kid, noticeOf("Only admins"))
}
