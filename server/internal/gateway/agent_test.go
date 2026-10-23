package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"scabbage/internal/agent"
	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/world"
)

func (ts *testServer) owner(t *testing.T, name string) agent.Owner {
	t.Helper()
	p, found, err := ts.gw.playerNamed(name)
	if err != nil || !found {
		t.Fatalf("no player %s: %v", name, err)
	}
	return agent.Owner{PlayerID: p.ID, Name: p.Name, Admin: p.Admin}
}

// call runs a tool and decodes its JSON answer.
func (ts *testServer) call(t *testing.T, o agent.Owner, tool, args string) (map[string]any, error) {
	t.Helper()
	out, err := ts.gw.Call(context.Background(), o, tool, json.RawMessage(args))
	if err != nil {
		return nil, err
	}
	b, _ := json.Marshal(out)
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m, nil
}

func refused(t *testing.T, err error, says string) {
	t.Helper()
	var r agent.Refusal
	if !errors.As(err, &r) || !strings.Contains(r.Error(), says) {
		t.Fatalf("err = %v, want a refusal saying %q", err, says)
	}
}

func isAgentCommand(m *pb.ServerMessage) bool { return m.GetAgentCommand() != nil }

func agentReply(id uint64, js string) *pb.ClientMessage {
	return &pb.ClientMessage{Body: &pb.ClientMessage_AgentReply{AgentReply: &pb.AgentReply{Id: id, Json: js}}}
}

func agentProgress(id uint64, state pb.AgentJobState, percent uint32, note string) *pb.ClientMessage {
	return &pb.ClientMessage{Body: &pb.ClientMessage_AgentProgress{AgentProgress: &pb.AgentProgress{
		Id: id, State: state, Percent: percent, Note: note,
	}}}
}

func TestAdminsLinkAgentsAndSeeTheTokenOnce(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	send(t, c, sendChat("/agent link laptop"))
	linked := recvWhere(t, c, func(m *pb.ServerMessage) bool { return m.GetAgentLinked() != nil }).GetAgentLinked()
	if linked.Label != "laptop" || !strings.HasPrefix(linked.Token, agent.TokenPrefix) {
		t.Fatalf("linked = %v", linked)
	}
	o, err := ts.agents.Owner(linked.Token)
	if err != nil || o.Name != "Mom" {
		t.Fatalf("owner = %+v, %v", o, err)
	}
	send(t, c, sendChat("/agent list"))
	recvWhere(t, c, noticeOf("laptop"))
	send(t, c, sendChat("/agent revoke laptop"))
	recvWhere(t, c, noticeOf("Revoked"))
	if _, err := ts.agents.Owner(linked.Token); !errors.Is(err, agent.ErrBadToken) {
		t.Fatal("revoked token still works")
	}
}

func TestPlayersNeedAnAdminToLinkAgents(t *testing.T) {
	ts := startServer(t)
	admin, kid := ts.joinKid(t)
	send(t, kid, sendChat("/agent link"))
	expectNotice(t, kid, "admin")
	send(t, kid, sendChat("/agent allow Kid"))
	expectNotice(t, kid, "Only admins")
	send(t, admin, sendChat("/agent allow kid"))
	recvWhere(t, admin, noticeOf("Kid may"))
	send(t, kid, sendChat("/agent link"))
	tok := recvWhere(t, kid, func(m *pb.ServerMessage) bool { return m.GetAgentLinked() != nil }).GetAgentLinked().Token
	send(t, admin, sendChat("/agent deny Kid"))
	recvWhere(t, admin, noticeOf("Kid can't"))
	if _, err := ts.agents.Owner(tok); !errors.Is(err, agent.ErrBadToken) {
		t.Fatal("denying agents left the kid's token working")
	}
}

func TestToolsNeedTheTabOpen(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	o := ts.owner(t, "Mom")
	_ = c.Close(websocket.StatusNormalClosure, "")
	waitPlayers(t, ts.hub, 0)
	_, err := ts.call(t, o, "whereami", `{}`)
	refused(t, err, "open Scabbage")
	if blocks, err := ts.call(t, o, "blocks", `{}`); err != nil || len(blocks["blocks"].([]any)) == 0 {
		t.Fatalf("blocks = %v, %v", blocks, err)
	}
}

func TestWhereAmIAndPlayers(t *testing.T) {
	ts := startServer(t)
	ts.joinAdmin(t)
	o := ts.owner(t, "Mom")
	x, y, z := ts.spawn()
	at, err := ts.call(t, o, "whereami", `{}`)
	if err != nil {
		t.Fatal(err)
	}
	if at["x"] != float64(int(x*10))/10 || at["y"] != float64(int(y*10))/10 || at["z"] != float64(int(z*10))/10 ||
		at["mode"] != "creative" || at["facing"] != "north" {
		t.Fatalf("whereami = %v (spawn %v %v %v)", at, x, y, z)
	}
	list, err := ts.call(t, o, "players", `{}`)
	if err != nil || len(list["players"].([]any)) != 1 || list["players"].([]any)[0].(map[string]any)["name"] != "Mom" {
		t.Fatalf("players = %v, %v", list, err)
	}
}

func TestBlocksListsWhatPlayersCanPlace(t *testing.T) {
	ts := startServer(t)
	ts.joinAdmin(t)
	out, _ := ts.call(t, ts.owner(t, "Mom"), "blocks", `{}`)
	blocks := map[string]map[string]any{}
	for _, b := range out["blocks"].([]any) {
		blocks[b.(map[string]any)["name"].(string)] = b.(map[string]any)
	}
	if blocks["planks"] == nil || blocks["air"] != nil || blocks["bedrock"] != nil || blocks["tripwire_on"] != nil {
		t.Fatalf("blocks = %v", out["blocks"])
	}
	if blocks["planks_stairs_east"]["shape"] != "stairs_east" || blocks["glass"]["see_through"] != true ||
		blocks["lamp"]["glows"] != true || blocks["planks"]["shape"] != nil {
		t.Fatalf("details: %v %v %v", blocks["planks_stairs_east"], blocks["glass"], blocks["lamp"])
	}
}

func TestLookCountsAndMapsABox(t *testing.T) {
	ts := startServer(t)
	ts.joinAdmin(t)
	x, y, z := ts.spawn()
	bx, by, bz := int32(x), int32(y)+20, int32(z)
	for dx := range int32(2) {
		if err := ts.world.SetBlock(bx+dx, by, bz, world.Planks); err != nil {
			t.Fatal(err)
		}
	}
	args, _ := json.Marshal(map[string]any{
		"from": map[string]int32{"x": bx, "y": by, "z": bz}, "to": map[string]int32{"x": bx + 2, "y": by + 1, "z": bz},
	})
	out, err := ts.call(t, ts.owner(t, "Mom"), "look", string(args))
	if err != nil {
		t.Fatal(err)
	}
	if out["counts"].(map[string]any)["planks"] != 2.0 || out["counts"].(map[string]any)["air"] != 4.0 {
		t.Fatalf("counts = %v", out["counts"])
	}
	heights, tops := out["heights"].([]any)[0].([]any), out["tops"].([]any)[0].([]any)
	if heights[0] != float64(by) || tops[1] != "planks" || heights[2] != nil || tops[2] != nil {
		t.Fatalf("heights = %v, tops = %v", heights, tops)
	}
	big := `{"from":{"x":0,"y":0,"z":0},"to":{"x":100,"y":100,"z":100}}`
	_, err = ts.call(t, ts.owner(t, "Mom"), "look", big)
	refused(t, err, "32,768")
}

func TestJobToolsRelayToTheTabAndReportStatus(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	o := ts.owner(t, "Mom")
	type result struct {
		out map[string]any
		err error
	}
	done := make(chan result, 1)
	go func() {
		out, err := ts.call(t, o, "fill", `{"from":{"x":0,"y":0,"z":0},"to":{"x":1,"y":1,"z":1},"block":"planks"}`)
		done <- result{out, err}
	}()
	cmd := recvWhere(t, c, isAgentCommand).GetAgentCommand()
	if cmd.Tool != "fill" || !strings.Contains(cmd.Json, `"planks"`) {
		t.Fatalf("command = %v", cmd)
	}
	send(t, c, agentReply(cmd.Id, `{"job":`+jsonNumber(cmd.Id)+`,"started":"Filling 8 blocks."}`))
	r := <-done
	if r.err != nil || r.out["started"] != "Filling 8 blocks." {
		t.Fatalf("fill = %v, %v", r.out, r.err)
	}
	status, _ := ts.call(t, o, "status", `{}`)
	if status["state"] != "running" || status["tool"] != "fill" {
		t.Fatalf("status = %v", status)
	}
	waited := make(chan map[string]any, 1)
	go func() {
		out, _ := ts.call(t, o, "status", `{"wait":true}`)
		waited <- out
	}()
	time.Sleep(20 * time.Millisecond)
	send(t, c, agentProgress(cmd.Id, pb.AgentJobState_AGENT_JOB_STATE_DONE, 100, "Done: 8 blocks."))
	select {
	case s := <-waited:
		if s["state"] != "done" || s["percent"] != 100.0 || s["note"] != "Done: 8 blocks." {
			t.Fatalf("status after waiting = %v", s)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("status wait never returned")
	}
}

func TestTheTabCanRefuseAndMustAnswerInTime(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	o := ts.owner(t, "Mom")
	errs := make(chan error, 1)
	go func() { _, err := ts.call(t, o, "goto", `{}`); errs <- err }()
	cmd := recvWhere(t, c, isAgentCommand).GetAgentCommand()
	send(t, c, agentReply(cmd.Id, `{"error":"Use goto with x, y and z."}`))
	refused(t, <-errs, "x, y and z")

	defer func(was time.Duration) { agentTimeout = was }(agentTimeout)
	agentTimeout = 50 * time.Millisecond
	_, err := ts.call(t, o, "stop", `{}`)
	refused(t, err, "didn't answer")
}

func TestSayChatsAsThePlayerWithARobot(t *testing.T) {
	ts := startServer(t)
	admin, kid := ts.joinKid(t)
	if _, err := ts.call(t, ts.owner(t, "Mom"), "say", `{"text":"  hello\nall "}`); err != nil {
		t.Fatal(err)
	}
	for _, c := range []*websocket.Conn{admin, kid} {
		ch := recvWhere(t, c, func(m *pb.ServerMessage) bool { return m.GetChat() != nil && !m.GetChat().Notice }).GetChat()
		if ch.Name != "Mom" || ch.Text != "🤖 hello all" {
			t.Fatalf("chat = %v", ch)
		}
	}
}

func jsonNumber(n uint64) string {
	b, _ := json.Marshal(n)
	return string(b)
}
