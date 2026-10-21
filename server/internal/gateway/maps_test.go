package gateway

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"scabbage/internal/access"
	"scabbage/internal/maps"
	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/world"
)

func isMapExported(m *pb.ServerMessage) bool { return m.GetMapExported() != nil }

func importMap(name string, data []byte) *pb.ClientMessage {
	return &pb.ClientMessage{Body: &pb.ClientMessage_ImportMap{ImportMap: &pb.ImportMap{Name: name, Data: data}}}
}

// tpAndWait moves an admin's player with /tp and waits until the server has them there.
func tpAndWait(t *testing.T, c *websocket.Conn, where string) *pb.Teleport {
	t.Helper()
	send(t, c, sendChat("/tp "+where))
	return recvWhere(t, c, isTeleport).GetTeleport()
}

func setBlock(t *testing.T, ts *testServer, x, y, z int32, b world.Block) {
	t.Helper()
	if err := ts.world.SetBlock(x, y, z, b); err != nil {
		t.Fatal(err)
	}
}

func blockIs(t *testing.T, ts *testServer, x, y, z int32, want world.Block) {
	t.Helper()
	if b, _ := ts.world.Block(x, y, z); b != want {
		t.Fatalf("block at %d,%d,%d is %v, want %v", x, y, z, b, want)
	}
}

func TestMapResetPutsTheSavedBlocksBack(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	tpAndWait(t, c, "0 90 0")
	send(t, c, sendChat("/zone add Castle 8")) // -4..3
	recvWhere(t, c, noticeOf("Zones updated"))
	setBlock(t, ts, 2, 80, 2, world.Chest)
	send(t, c, sendChat("/map save Castle"))
	recvWhere(t, c, noticeOf("Saved the map Castle"))

	setBlock(t, ts, 2, 80, 2, world.Chest+1) // looted
	setBlock(t, ts, 1, 70, 1, world.Brick)
	setBlock(t, ts, 5, 70, 5, world.Brick) // outside the zone
	send(t, c, sendChat("/map reset Castle"))
	batch := recvWhere(t, c, func(m *pb.ServerMessage) bool { return m.GetBlockBatch() != nil }).GetBlockBatch()
	if len(batch.Blocks) != 2 {
		t.Fatalf("reset sent %d blocks", len(batch.Blocks))
	}
	recvWhere(t, c, noticeOf("starts over"))
	blockIs(t, ts, 2, 80, 2, world.Chest)
	blockIs(t, ts, 1, 70, 1, world.Air)
	blockIs(t, ts, 5, 70, 5, world.Brick)
}

func TestMapStartIsWherePlayersGoOnReset(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	tpAndWait(t, c, "0 90 0")
	send(t, c, sendChat("/zone add Castle 8"))
	recvWhere(t, c, noticeOf("Zones updated"))
	send(t, c, sendChat("/map start Castle"))
	recvWhere(t, c, noticeOf("Save the map first"))
	send(t, c, sendChat("/map save Castle"))
	recvWhere(t, c, noticeOf("Saved"))
	tpAndWait(t, c, "2 85 -3")
	send(t, c, sendChat("/map start Castle"))
	recvWhere(t, c, noticeOf("start here"))
	tpAndWait(t, c, "0 90 0")
	send(t, c, sendChat("/map reset Castle"))
	if p := recvWhere(t, c, isTeleport).GetTeleport().Pos; p.X != 2.5 || p.Y != 85 || p.Z != -2.5 {
		t.Fatalf("sent to %v", p)
	}
	tpAndWait(t, c, "40 90 40")
	send(t, c, sendChat("/map start Castle"))
	recvWhere(t, c, noticeOf("inside Castle"))
}

func TestMapResetPutsScriptsBack(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	tpAndWait(t, c, "0 90 0")
	send(t, c, sendChat("/zone add Castle 8"))
	recvWhere(t, c, noticeOf("Zones updated"))
	at := world.Pos{X: 1, Y: 80, Z: 1}
	setBlock(t, ts, at.X, at.Y, at.Z, world.ScriptBlock)
	ts.saveScript(t, at, `say("hello")`)
	send(t, c, sendChat("/map save Castle"))
	recvWhere(t, c, noticeOf("1 script"))

	setBlock(t, ts, at.X, at.Y, at.Z, world.Air)
	ts.deleteScript(t, at)
	stray := world.Pos{X: -2, Y: 80, Z: -2}
	setBlock(t, ts, stray.X, stray.Y, stray.Z, world.ScriptBlock)
	ts.saveScript(t, stray, `say("new")`)
	send(t, c, sendChat("/map reset Castle"))
	recvWhere(t, c, noticeOf("starts over"))
	if got := ts.scriptAt(t, at); got != `say("hello")` {
		t.Fatalf("script = %q", got)
	}
	if got := ts.scriptAt(t, stray); got != "" {
		t.Fatalf("script added after saving stayed: %q", got)
	}
}

func TestMapExportThenImportElsewhere(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	c.SetReadLimit(maps.MaxFileBytes)
	tpAndWait(t, c, "0 90 0")
	send(t, c, sendChat("/zone add Castle 8"))
	recvWhere(t, c, noticeOf("Zones updated"))
	send(t, c, sendChat("/zone mode Castle survival"))
	recvWhere(t, c, noticeOf("Zones updated"))
	setBlock(t, ts, 0, 100, 0, world.GoldBlock)
	send(t, c, sendChat("/map save Castle"))
	recvWhere(t, c, noticeOf("Saved"))
	send(t, c, sendChat("/map export Castle"))
	file := recvWhere(t, c, isMapExported).GetMapExported()
	if file.Name != "Castle" || len(file.Data) == 0 {
		t.Fatalf("exported %q, %d bytes", file.Name, len(file.Data))
	}

	send(t, c, importMap("Copy", file.Data))
	recvWhere(t, c, noticeOf("would cover the zone Castle"))
	tpAndWait(t, c, "30 90 30")
	send(t, c, importMap("Copy", file.Data))
	recvWhere(t, c, noticeOf("Placed Copy"))
	blockIs(t, ts, 30, 100, 30, world.GoldBlock)
	z, ok := ts.zones.Named("Copy")
	if !ok || z.X0 != 26 || z.X1 != 33 || z.Mode != access.ModeSurvival {
		t.Fatalf("zone = %+v, %v", z, ok)
	}
	if _, ok, _ := ts.gw.cfg.Maps.Get("Copy"); !ok {
		t.Fatal("the imported map wasn't saved for resets")
	}
	send(t, c, importMap("Copy", file.Data))
	recvWhere(t, c, noticeOf("already a zone called Copy"))
	send(t, c, importMap("Junk", []byte("not a map")))
	recvWhere(t, c, noticeOf("isn't a map file"))
}

func TestOnlyAdminsImportMaps(t *testing.T) {
	ts := startServer(t)
	_, kid := ts.joinKid(t)
	send(t, kid, importMap("Copy", []byte("x")))
	recvWhere(t, kid, noticeOf("Only admins"))
}

func TestMapSamplePlacesAStarterAdventure(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	tpAndWait(t, c, "0 40 0")
	send(t, c, sendChat("/map sample Quest"))
	recvWhere(t, c, noticeOf("Placed Quest")) // written straight away, before the queued broadcasts
	stale := recvWhere(t, c, func(m *pb.ServerMessage) bool { return m.GetChunksStale() != nil }).GetChunksStale()
	if n := len(stale.Chunks); n == 0 || n > 2*2*8 {
		t.Fatalf("%d chunks to fetch again", n) // 24×16 blocks around 0,0 touch 2 by 2 columns of 8
	}
	z, ok := ts.zones.Named("Quest")
	if !ok {
		t.Fatal("no zone for the sample")
	}
	scripts, err := ts.gw.scripts.Inside(z.X0, z.Z0, z.X1, z.Z1)
	if err != nil || len(scripts) == 0 {
		t.Fatalf("sample has %d scripts (%v)", len(scripts), err)
	}
	for _, at := range scripts {
		ts.gw.editMu.Lock()
		saved, _, _ := ts.gw.scripts.Source(at)
		ts.gw.editMu.Unlock()
		if saved.Err != "" {
			t.Errorf("script at %v: %s", at, saved.Err)
		}
	}

	// The hidden switch behind the house opens its gate.
	g := int32(39)
	sw := world.Pos{X: z.X0 + 20, Y: g + 1, Z: z.Z0 + 13}
	gate := world.Pos{X: z.X0 + 16, Y: g + 1, Z: z.Z0 + 7}
	blockIs(t, ts, gate.X, gate.Y, gate.Z, world.Cobblestone)
	tpAndWait(t, c, fmt.Sprintf("%d %d %d", sw.X-2, sw.Y, sw.Z))
	send(t, c, useBlock(sw.X, sw.Y, sw.Z))
	recvWhere(t, c, func(m *pb.ServerMessage) bool { return strings.Contains(m.GetChat().GetText(), "opens") })
	blockIs(t, ts, gate.X, gate.Y, gate.Z, world.Air)
	blockIs(t, ts, gate.X, gate.Y+1, gate.Z, world.Air)
}

func (ts *testServer) saveScript(t *testing.T, at world.Pos, src string) {
	t.Helper()
	ts.gw.editMu.Lock()
	defer ts.gw.editMu.Unlock()
	if err := ts.gw.scripts.Save(at, src, 1, time.Now()); err != nil {
		t.Fatal(err)
	}
}

func (ts *testServer) deleteScript(t *testing.T, at world.Pos) {
	t.Helper()
	ts.gw.editMu.Lock()
	defer ts.gw.editMu.Unlock()
	if err := ts.gw.scripts.Delete(at); err != nil {
		t.Fatal(err)
	}
}

func (ts *testServer) scriptAt(t *testing.T, at world.Pos) string {
	t.Helper()
	ts.gw.editMu.Lock()
	defer ts.gw.editMu.Unlock()
	saved, _, err := ts.gw.scripts.Source(at)
	if err != nil {
		t.Fatal(err)
	}
	return saved.Source
}

func TestMapRemovePutsBackTheLandUnderAPlacedMap(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	setBlock(t, ts, 3, 60, 3, world.ScriptBlock) // someone's build where the map will go
	ts.saveScript(t, world.Pos{X: 3, Y: 60, Z: 3}, `say("mine")`)
	setBlock(t, ts, 5, 45, -2, world.Brick)
	before, err := ts.world.Copy(world.Area{X0: -12, Z0: -8, X1: 11, Z1: 7})
	if err != nil {
		t.Fatal(err)
	}
	tpAndWait(t, c, "0 40 0")
	send(t, c, sendChat("/map sample Quest"))
	recvWhere(t, c, noticeOf("Placed Quest"))
	blockIs(t, ts, 3, 60, 3, world.Air)
	send(t, c, sendChat("/zone remove Quest"))
	recvWhere(t, c, noticeOf("/map remove Quest puts that land back"))

	send(t, c, sendChat("/map remove Quest"))
	recvWhere(t, c, noticeOf("Removed Quest"))
	after, _ := ts.world.Copy(world.Area{X0: -12, Z0: -8, X1: 11, Z1: 7})
	if !slices.Equal(after, before) {
		t.Fatal("the land under the map didn't come back")
	}
	if got := ts.scriptAt(t, world.Pos{X: 3, Y: 60, Z: 3}); got != `say("mine")` {
		t.Fatalf("script under the map = %q", got)
	}
	if scripts, _ := ts.gw.scripts.Inside(-12, -8, 11, 7); len(scripts) != 1 {
		t.Fatalf("%d scripts left; the map's should be gone", len(scripts))
	}
	if _, ok := ts.zones.Named("Quest"); ok {
		t.Fatal("the zone stayed")
	}
}

func TestMapRemoveNeedsLandThatWasKept(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	tpAndWait(t, c, "0 90 0")
	send(t, c, sendChat("/zone add Castle 8"))
	recvWhere(t, c, noticeOf("Zones updated"))
	send(t, c, sendChat("/map save Castle"))
	recvWhere(t, c, noticeOf("Saved"))
	send(t, c, sendChat("/map remove Castle"))
	recvWhere(t, c, noticeOf("only for maps placed"))
	if _, ok := ts.zones.Named("Castle"); !ok {
		t.Fatal("the zone went away")
	}
}
