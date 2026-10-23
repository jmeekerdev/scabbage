package gateway

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"scabbage/internal/agent"
	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/plan"
)

const hutPlan = `{"name":"hut","about":"a tiny hut","palette":{"p":"planks","g":"glass"},
	"layers":[["ppp","ppp"],["p_p","pgp"],["ppp","ppp"]]}`

// propose has the agent propose a plan, the tab hold it for /yes, and returns the command the tab got.
func (ts *testServer) propose(t *testing.T, c *websocket.Conn, o agent.Owner, planJSON string) *pb.AgentCommand {
	t.Helper()
	done := make(chan error, 1)
	go func() {
		_, err := ts.call(t, o, "propose", `{"plan":`+planJSON+`,"x":0,"y":10,"z":0,"facing":"north"}`)
		done <- err
	}()
	cmd := recvWhere(t, c, isAgentCommand).GetAgentCommand()
	send(t, c, agentReply(cmd.Id, `{"job":`+jsonNumber(cmd.Id)+`,"started":"Building a hut.","awaiting":true}`))
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	return cmd
}

func TestProposeAlwaysWaitsAndSendsThePlanToTheTab(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	o := ts.owner(t, "Mom")
	o.SkipConfirm = true
	cmd := ts.propose(t, c, o, hutPlan)
	if cmd.Tool != "build" || !cmd.Confirm {
		t.Fatalf("command = %v, want a build waiting for /yes", cmd)
	}
	var args struct {
		Plan   plan.Plan
		Facing string
	}
	if err := json.Unmarshal([]byte(cmd.Json), &args); err != nil || args.Plan.Layers[1][1] != "pgp" || args.Facing != "north" {
		t.Fatalf("args = %s (%v)", cmd.Json, err)
	}
	if s, _ := ts.call(t, o, "status", `{}`); s["state"] != "awaiting_confirmation" {
		t.Fatalf("status = %v", s)
	}
}

func TestProposeRefusesBadPlansSayingWhy(t *testing.T) {
	ts := startServer(t)
	ts.joinAdmin(t)
	o := ts.owner(t, "Mom")
	bad := strings.Replace(hutPlan, `"pgp"`, `"pzp"`, 1)
	_, err := ts.call(t, o, "propose", `{"plan":`+bad+`,"x":0,"y":10,"z":0}`)
	refused(t, err, "layer 2 row 2 uses z")
	_, err = ts.call(t, o, "propose", `{"plan":"nothing","x":0,"y":10,"z":0}`)
	refused(t, err, "no plan")
}

func TestSavePlanSavesTheLastProposal(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	o := ts.owner(t, "Mom")
	_, err := ts.call(t, o, "save_plan", `{}`)
	refused(t, err, "propose")
	ts.propose(t, c, o, hutPlan)
	if _, err := ts.call(t, o, "save_plan", `{"name":"Little-Hut"}`); err != nil {
		t.Fatal(err)
	}
	list, _ := ts.call(t, o, "plans", `{}`)
	plans, _ := list["plans"].([]any)
	if len(plans) != 1 || plans[0].(map[string]any)["name"] != "little-hut" {
		t.Fatalf("plans = %v", list)
	}
	got, err := ts.call(t, o, "plan", `{"name":"little-hut"}`)
	if err != nil || got["about"] != "a tiny hut" || got["layers"] == nil {
		t.Fatalf("plan = %v, %v", got, err)
	}
}

func TestBuildFindsSavedPlansByName(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	o := ts.owner(t, "Mom")
	o.SkipConfirm = true
	ts.savePlan(t, o.PlayerID, hutPlan)
	go func() { _, _ = ts.call(t, o, "build", `{"template":"hut","x":0,"y":10,"z":0,"material":"stone"}`) }()
	cmd := recvWhere(t, c, isAgentCommand).GetAgentCommand()
	if cmd.Tool != "build" || cmd.Confirm || !strings.Contains(cmd.Json, `"layers"`) || !strings.Contains(cmd.Json, `"stone"`) {
		t.Fatalf("command = %v", cmd)
	}
	send(t, c, agentReply(cmd.Id, `{"error":"x"}`))
	_, err := ts.call(t, o, "build", `{"template":"hut","x":0,"y":10,"z":0,"w":5}`)
	refused(t, err, "can't be resized")
	_, err = ts.call(t, o, "build", `{"template":"castle","x":0,"y":10,"z":0}`)
	refused(t, err, "no template or plan called castle")
}

func (ts *testServer) savePlan(t *testing.T, owner int64, planJSON string) {
	t.Helper()
	var p plan.Plan
	if err := json.Unmarshal([]byte(planJSON), &p); err != nil {
		t.Fatal(err)
	}
	if err := p.Check(isPlanBlock); err != nil {
		t.Fatal(err)
	}
	if err := ts.gw.cfg.Plans.Save(owner, p); err != nil {
		t.Fatal(err)
	}
}

func TestPlanCommandsSaveShareRenameAndDelete(t *testing.T) {
	ts := startServer(t)
	admin, kid := ts.joinKid(t)
	mom := ts.owner(t, "Mom")
	ts.propose(t, admin, mom, hutPlan)
	send(t, admin, sendChat("/plan save hut"))
	recvWhere(t, admin, noticeOf("Saved hut"))
	send(t, admin, sendChat("/plan share hut"))
	recvWhere(t, admin, noticeOf("Shared hut"))

	ts.savePlan(t, ts.owner(t, "Kid").PlayerID, hutPlan)
	send(t, kid, sendChat("/plan share hut"))
	recvWhere(t, kid, noticeOf("Another shared plan"))
	send(t, kid, sendChat("/plan rename hut shack"))
	recvWhere(t, kid, noticeOf("Renamed hut to shack"))
	send(t, kid, sendChat("/plans"))
	recvWhere(t, kid, noticeOf("shack"))

	send(t, kid, sendChat("/plan delete shack"))
	recvWhere(t, kid, noticeOf("Deleted shack"))
	send(t, admin, sendChat("/plan delete hut"))
	recvWhere(t, admin, noticeOf("Deleted hut"))
	send(t, kid, sendChat("/plan nonsense"))
	recvWhere(t, kid, noticeOf("/plan save NAME"))
}

func TestAdminsDeleteOthersSharedPlans(t *testing.T) {
	ts := startServer(t)
	admin, _ := ts.joinKid(t)
	kid := ts.owner(t, "Kid").PlayerID
	ts.savePlan(t, kid, hutPlan)
	if err := ts.gw.cfg.Plans.Share(kid, "hut", true); err != nil {
		t.Fatal(err)
	}
	send(t, admin, sendChat("/plan delete hut"))
	recvWhere(t, admin, noticeOf("Deleted Kid's shared plan hut"))
}

func TestPlanRequestSendsTheSavedPlan(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	ts.savePlan(t, ts.owner(t, "Mom").PlayerID, hutPlan)
	ask := func(name string) *pb.Plan {
		send(t, c, &pb.ClientMessage{Body: &pb.ClientMessage_PlanRequest{PlanRequest: &pb.PlanRequest{Name: name}}})
		return recvWhere(t, c, func(m *pb.ServerMessage) bool { return m.GetPlan() != nil }).GetPlan()
	}
	if got := ask("HUT"); got.Name != "HUT" || !strings.Contains(got.Json, `"pgp"`) || got.Error != "" {
		t.Fatalf("plan = %v", got)
	}
	if got := ask("barn"); got.Error == "" || got.Json != "" {
		t.Fatalf("plan = %v", got)
	}
}

func TestChatToolShowsRecentLines(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	send(t, c, sendChat("make the roof steeper"))
	recvWhere(t, c, func(m *pb.ServerMessage) bool { return m.GetChat() != nil })
	out, err := ts.call(t, ts.owner(t, "Mom"), "chat", `{}`)
	if err != nil {
		t.Fatal(err)
	}
	lines, _ := out["lines"].([]any)
	if len(lines) != 1 || lines[0].(map[string]any)["text"] != "make the roof steeper" || lines[0].(map[string]any)["name"] != "Mom" {
		t.Fatalf("chat = %v", out)
	}
	waited := make(chan map[string]any, 1)
	go func() {
		out, _ := ts.call(t, ts.owner(t, "Mom"), "chat", `{"after":1,"wait":true}`)
		waited <- out
	}()
	time.Sleep(20 * time.Millisecond)
	send(t, c, sendChat("taller please"))
	select {
	case out := <-waited:
		lines, _ := out["lines"].([]any)
		if len(lines) != 1 || lines[0].(map[string]any)["text"] != "taller please" || out["last"] != 2.0 {
			t.Fatalf("chat after waiting = %v", out)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("chat wait never returned")
	}
}

func TestAiMessagesReachOnlyTheirPlayersAgent(t *testing.T) {
	ts := startServer(t)
	admin, kid := ts.joinKid(t)
	mom := ts.owner(t, "Mom")
	send(t, admin, sendChat("/ai build a treehouse"))
	recvWhere(t, admin, noticeOf("isn't listening"))
	out, err := ts.call(t, mom, "listen", `{"seconds":1}`)
	msgs, _ := out["messages"].([]any)
	if err != nil || len(msgs) != 1 || msgs[0].(map[string]any)["text"] != "build a treehouse" {
		t.Fatalf("listen = %v, %v", out, err)
	}
	if out, _ := ts.call(t, ts.owner(t, "Kid"), "listen", `{"seconds":1}`); len(out["messages"].([]any)) != 0 {
		t.Fatalf("the kid's agent heard Mom: %v", out)
	}

	heard := make(chan map[string]any, 1)
	go func() { out, _ := ts.call(t, mom, "listen", `{}`); heard <- out }()
	time.Sleep(20 * time.Millisecond)
	send(t, admin, sendChat("/ai taller"))
	recvWhere(t, admin, noticeOf("To your agent: taller"))
	if out := <-heard; out["messages"].([]any)[0].(map[string]any)["text"] != "taller" {
		t.Fatalf("listen = %v", out)
	}

	if _, err := ts.call(t, mom, "say", `{"text":"which trees?","private":true}`); err != nil {
		t.Fatal(err)
	}
	if ch := recvWhere(t, admin, func(m *pb.ServerMessage) bool { return m.GetChat() != nil && !m.GetChat().Notice }).GetChat(); ch.Text != "🤖 which trees?" {
		t.Fatalf("chat = %v", ch)
	}
	send(t, admin, sendChat("everyone sees this"))
	if ch := recvWhere(t, kid, func(m *pb.ServerMessage) bool { return m.GetChat() != nil }).GetChat(); ch.Text != "everyone sees this" {
		t.Fatalf("the kid saw %q first: /ai or private say leaked", ch.Text)
	}
}
