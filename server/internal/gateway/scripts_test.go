package gateway

import (
	"fmt"
	"strings"
	"testing"

	"github.com/coder/websocket"

	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/world"
)

func openScript(x, y, z int32) *pb.ClientMessage {
	return &pb.ClientMessage{Body: &pb.ClientMessage_OpenScript{OpenScript: &pb.OpenScript{Pos: &pb.BlockPos{X: x, Y: y, Z: z}}}}
}

func saveScript(x, y, z int32, src string) *pb.ClientMessage {
	return &pb.ClientMessage{Body: &pb.ClientMessage_SaveScript{SaveScript: &pb.SaveScript{
		Pos: &pb.BlockPos{X: x, Y: y, Z: z}, Source: src,
	}}}
}

func scriptShown(t *testing.T, c *websocket.Conn) *pb.ScriptShown {
	t.Helper()
	return waitFor(t, c, func(m *pb.ServerMessage) bool { return m.GetScriptShown() != nil }).GetScriptShown()
}

func scriptSaid(t *testing.T, c *websocket.Conn, text string) {
	t.Helper()
	waitFor(t, c, func(m *pb.ServerMessage) bool {
		ch := m.GetChat()
		return ch != nil && ch.Name == "Script" && ch.Text == text
	})
}

// scriptBlock saves src into a new script block beside spawn and returns where it is.
func (ts *testServer) scriptBlock(t *testing.T, c *websocket.Conn, src string, rest ...world.Block) (x, y, z int32) {
	t.Helper()
	x, y, z = ts.circuit(t, append([]world.Block{world.ScriptBlock}, rest...)...)
	send(t, c, saveScript(x, y, z, src))
	if got := scriptShown(t, c); !got.Saved || got.Error != "" || !got.Editable {
		t.Fatalf("save answered %v", got)
	}
	return x, y, z
}

func TestScriptBlocksRunWhenUsed(t *testing.T) {
	ts := startServer(t)
	admin := ts.joinAdmin(t)
	x, y, z := ts.scriptBlock(t, admin, `function on_touch(p) say("hello " .. p) end`)

	send(t, admin, useBlock(x, y, z))
	waitFor(t, admin, func(m *pb.ServerMessage) bool {
		ch := m.GetChat()
		return ch != nil && ch.Name == "Script" && strings.HasPrefix(ch.Text, "hello ")
	})
	send(t, admin, openScript(x, y, z))
	if got := scriptShown(t, admin); !strings.Contains(got.Source, "on_touch") || got.Saved {
		t.Fatalf("opened %v", got)
	}
}

func TestScriptBlocksRunWhenPowered(t *testing.T) {
	ts := startServer(t)
	admin := ts.joinAdmin(t)
	x, y, z := ts.circuit(t, world.Switch, world.Wire)
	if err := ts.world.SetBlock(x+2, y, z, world.ScriptBlock); err != nil {
		t.Fatal(err)
	}
	send(t, admin, saveScript(x+2, y, z, `function on_power(on) if on then say("on") end end`))
	scriptShown(t, admin)

	send(t, admin, useBlock(x, y, z))
	scriptSaid(t, admin, "on")
}

func TestScriptsChangeBlocksNearby(t *testing.T) {
	ts := startServer(t)
	admin := ts.joinAdmin(t)
	x, y, z := ts.circuit(t, world.ScriptBlock)
	send(t, admin, saveScript(x, y, z, fmt.Sprintf(`function on_touch() set_block(%d, %d, %d, "lamp") end`, x+1, y, z)))
	scriptShown(t, admin)

	send(t, admin, useBlock(x, y, z))
	waitForBlock(t, admin, x+1, y, z, world.Lamp)
}

func TestScriptErrorsReachWhoeverTouchedThem(t *testing.T) {
	ts := startServer(t)
	admin := ts.joinAdmin(t)
	x, y, z := ts.scriptBlock(t, admin, `function on_touch() set_block(0, 0, 0, "unobtainium") end`)

	send(t, admin, useBlock(x, y, z))
	waitFor(t, admin, func(m *pb.ServerMessage) bool {
		return m.GetChat().GetNotice() && strings.Contains(m.GetChat().GetText(), "no block called that")
	})
}

func TestBreakingAScriptBlockForgetsItsScript(t *testing.T) {
	ts := startServer(t)
	admin := ts.joinAdmin(t)
	x, y, z := ts.scriptBlock(t, admin, `x = 1`)

	send(t, admin, editBlock(x, y, z, world.Air))
	waitForBlock(t, admin, x, y, z, world.Air)
	send(t, admin, editBlock(x, y, z, world.ScriptBlock))
	waitForBlock(t, admin, x, y, z, world.ScriptBlock)
	send(t, admin, openScript(x, y, z))
	if got := scriptShown(t, admin); got.Source != "" {
		t.Fatalf("a new block kept the old script: %v", got)
	}
}

func TestOnlyBuildersSaveScripts(t *testing.T) {
	ts := startServer(t)
	admin, kid := ts.joinKid(t)
	x, y, z := ts.scriptBlock(t, admin, `x = 1`)
	zoneCommand(t, admin, kid, "/zone add Home 8", 1)

	send(t, kid, saveScript(x, y, z, `x = 2`))
	send(t, kid, openScript(x, y, z))
	if got := scriptShown(t, kid); got.Source != "x = 1" || got.Editable {
		t.Fatalf("kid sees %v", got)
	}
}
