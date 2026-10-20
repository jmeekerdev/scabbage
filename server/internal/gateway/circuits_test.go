package gateway

import (
	"testing"
	"time"

	"github.com/coder/websocket"

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

// waitForBlock pings until the server broadcasts b at x, y, z.
func waitForBlock(t *testing.T, c *websocket.Conn, x, y, z int32, b world.Block) {
	t.Helper()
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); {
		send(t, c, ping(1))
		ch := recv(t, c).GetBlockChanged()
		if ch != nil && ch.Pos.X == x && ch.Pos.Y == y && ch.Pos.Z == z && ch.Block == uint32(b) {
			return
		}
	}
	t.Fatalf("block at %d,%d,%d never became %v", x, y, z, b)
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
	waitForBlock(t, admin, x, y, z, world.Button)
	waitForBlock(t, admin, x+1, y, z, world.SignalLamp)
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
	waitForBlock(t, admin, x, y, z, world.Plate)
	waitForBlock(t, admin, x+1, y, z, world.SignalLamp)
}
