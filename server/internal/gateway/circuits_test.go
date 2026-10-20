package gateway

import (
	"slices"
	"testing"
	"time"

	"github.com/coder/websocket"
	"google.golang.org/protobuf/proto"

	"scabbage/internal/logic"
	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/world"
)

func useBlock(x, y, z int32) *pb.ClientMessage {
	return &pb.ClientMessage{Body: &pb.ClientMessage_UseBlock{UseBlock: &pb.UseBlock{Pos: &pb.BlockPos{X: x, Y: y, Z: z}}}}
}

// circuit lays blocks in a row along +x beside spawn and returns the first one's position.
func (ts *testServer) circuit(t *testing.T, blocks ...world.Block) (x, y, z int32) {
	t.Helper()
	x, y, z = ts.besideSpawn(t, 0, blocks[0])
	for i, b := range blocks[1:] {
		if err := ts.world.SetBlock(x+int32(i)+1, y, z, b); err != nil {
			t.Fatal(err)
		}
	}
	return x, y, z
}

// waitFor pings until the server sends a message that is wanted.
func waitFor(t *testing.T, c *websocket.Conn, wanted func(*pb.ServerMessage) bool) *pb.ServerMessage {
	t.Helper()
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); {
		send(t, c, ping(1))
		if m := recv(t, c); wanted(m) {
			return m
		}
	}
	t.Fatal("the server never sent what we waited for")
	return nil
}

// waitForBlock pings until the server broadcasts b at x, y, z, alone or in a batch.
func waitForBlock(t *testing.T, c *websocket.Conn, x, y, z int32, b world.Block) {
	t.Helper()
	waitForBlocks(t, c, changed(x, y, z, b))
}

// waitForBlocks pings until the server has broadcast every one of want, in any messages.
func waitForBlocks(t *testing.T, c *websocket.Conn, want ...*pb.BlockChanged) {
	t.Helper()
	waitFor(t, c, func(m *pb.ServerMessage) bool {
		changes := m.GetBlockBatch().GetBlocks()
		if ch := m.GetBlockChanged(); ch != nil {
			changes = append(changes, ch)
		}
		want = slices.DeleteFunc(want, func(w *pb.BlockChanged) bool {
			return slices.ContainsFunc(changes, func(ch *pb.BlockChanged) bool { return proto.Equal(ch, w) })
		})
		return len(want) == 0
	})
}

func changed(x, y, z int32, b world.Block) *pb.BlockChanged {
	return &pb.BlockChanged{Pos: &pb.BlockPos{X: x, Y: y, Z: z}, Block: uint32(b)}
}

func TestSwitchesLightLampsDownTheWire(t *testing.T) {
	ts := startServer(t)
	admin, kid := ts.joinKid(t)
	x, y, z := ts.circuit(t, world.Switch, world.Wire, world.SignalLamp)

	send(t, kid, useBlock(x, y, z))
	waitForBlock(t, admin, x+2, y, z, world.SignalLamp+1)
	send(t, kid, useBlock(x, y, z))
	waitForBlock(t, kid, x+2, y, z, world.SignalLamp)
	if b, _ := ts.world.Block(x+1, y, z); b != world.Wire {
		t.Fatalf("wire = %v", b)
	}
}

func TestLongCircuitsSwitchInOneMessage(t *testing.T) {
	ts := startServer(t)
	admin := ts.joinAdmin(t)
	row := []world.Block{world.Switch}
	for range 30 {
		row = append(row, world.Wire, world.SignalLamp)
	}
	x, y, z := ts.circuit(t, row...)
	for dx := int32(1); dx < int32(len(row)); dx++ {
		if err := ts.world.SetBlock(x+dx, y, z+1, world.Wire); err != nil { // carries power past each lamp
			t.Fatal(err)
		}
	}

	send(t, admin, useBlock(x, y, z))
	batch := waitFor(t, admin, func(m *pb.ServerMessage) bool { return m.GetBlockBatch() != nil }).GetBlockBatch()
	if len(batch.Blocks) <= outQueueSize {
		t.Fatalf("%d changes in the batch", len(batch.Blocks))
	}
	send(t, admin, ping(1))
	expectPong(t, admin)
}

func TestPlacingWireCompletesACircuit(t *testing.T) {
	ts := startServer(t)
	admin := ts.joinAdmin(t)
	x, y, z := ts.circuit(t, world.Switch+1, world.Air, world.SignalLamp)

	send(t, admin, editBlock(x+1, y, z, world.Wire))
	waitForBlock(t, admin, x+2, y, z, world.SignalLamp+1)
}

func TestButtonsLetGo(t *testing.T) {
	ts := startServer(t)
	admin := ts.joinAdmin(t)
	x, y, z := ts.circuit(t, world.Button, world.SignalLamp)

	send(t, admin, useBlock(x, y, z))
	waitForBlock(t, admin, x+1, y, z, world.SignalLamp+1)
	waitForBlocks(t, admin, changed(x, y, z, world.Button), changed(x+1, y, z, world.SignalLamp))
}

func TestOnlySwitchesAndButtonsAreUsed(t *testing.T) {
	ts := startServer(t)
	admin := ts.joinAdmin(t)
	x, y, z := ts.circuit(t, world.Stone)

	send(t, admin, useBlock(x, y, z))
	expectBlockChanged(t, admin, x, y, z, world.Stone)
}

func TestStandingOnAPlatePressesIt(t *testing.T) {
	ts := startServer(t)
	admin := ts.joinAdmin(t)
	sx, sy, sz := ts.spawn()
	x, y, z := int32(sx), int32(sy), int32(sz)
	for i, b := range []world.Block{world.Plate, world.SignalLamp} {
		if err := ts.world.SetBlock(x+int32(i), y, z, b); err != nil {
			t.Fatal(err)
		}
	}
	waitForBlock(t, admin, x+1, y, z, world.SignalLamp+1)

	send(t, admin, move(sx-1, sy, sz, 0))
	waitForBlocks(t, admin, changed(x, y, z, world.Plate), changed(x+1, y, z, world.SignalLamp))
}

func TestDoorsOpenByHand(t *testing.T) {
	ts := startServer(t)
	admin := ts.joinAdmin(t)
	x, y, z := ts.circuit(t, world.Door)

	send(t, admin, useBlock(x, y, z))
	waitForBlock(t, admin, x, y, z, world.Door+1)
}

func TestTimersBlink(t *testing.T) {
	ts := startServer(t)
	admin := ts.joinAdmin(t)
	x, y, z := ts.circuit(t, world.Timer, world.SignalLamp)

	waitForBlock(t, admin, x+1, y, z, world.SignalLamp+1)
	waitForBlock(t, admin, x+1, y, z, world.SignalLamp)
}

func TestTeleportersCarryPlayersAlongTheirWire(t *testing.T) {
	ts := startServer(t)
	admin := ts.joinAdmin(t)
	sx, sy, sz := ts.spawn()
	x, y, z := int32(sx)-1, int32(sy), int32(sz)
	for i, b := range []world.Block{world.Teleporter, world.Wire, world.Wire, world.Teleporter} {
		if err := ts.world.SetBlock(x, y, z+int32(i), b); err != nil {
			t.Fatal(err)
		}
	}

	send(t, admin, move(sx-1, sy, sz, 0))
	tp := waitFor(t, admin, func(m *pb.ServerMessage) bool { return m.GetTeleport() != nil }).GetTeleport()
	if tp.Id != 1 || tp.Pos.X != sx-1 || tp.Pos.Y != sy || tp.Pos.Z != sz+3 {
		t.Fatalf("teleport = %v", tp)
	}
}

func TestPistonsKeepOutOfZonesTheyAreNotIn(t *testing.T) {
	ts := startServer(t)
	if err := ts.zones.Put("home", -5, -5, 5, 5); err != nil {
		t.Fatal(err)
	}
	w := circuitWorld{ts.gw}
	if !w.CanPush(logic.Pos{X: 10}, logic.Pos{X: 11}) || !w.CanPush(logic.Pos{X: 1}, logic.Pos{X: 2}) {
		t.Fatal("pistons push within the wilds and within their own zone")
	}
	if w.CanPush(logic.Pos{X: 6}, logic.Pos{X: 5}) || w.CanPush(logic.Pos{X: 5}, logic.Pos{X: 6}) {
		t.Fatal("pistons push neither into nor out of a zone")
	}
}
