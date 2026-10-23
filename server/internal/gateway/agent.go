package gateway

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"scabbage/internal/agent"
	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/rate"
)

// agentTimeout bounds how long a relayed tool waits for the player's tab to answer.
var agentTimeout = 10 * time.Second

// statusWait is the longest status waits for a job to end, short of common HTTP client timeouts.
const statusWait = 20 * time.Second

// relayed tools are run by the player's tab, through its autopilot.
var relayed = map[string]bool{
	"goto": true, "clear": true, "fill": true, "place": true, "build": true, "stop": true, "undo": true,
}

// Call runs an agent's tool for its owner: the server answers what it knows, the tab the rest.
func (g *Gateway) Call(ctx context.Context, o agent.Owner, tool string, args json.RawMessage) (any, error) {
	switch tool {
	case "blocks":
		return placeableBlocks(), nil
	case "players":
		return g.onlinePlayers(), nil
	}
	s := g.tabOf(o.PlayerID)
	if s == nil {
		return nil, agent.Refusal("open Scabbage in a browser first, and join the world as " + o.Name)
	}
	switch tool {
	case "whereami":
		return g.whereAmI(s)
	case "look":
		return g.look(args)
	case "status":
		return s.agent.status(ctx, args)
	case "say":
		return g.agentSay(s, args)
	}
	if relayed[tool] {
		return s.agent.relay(ctx, s, agentCommand{g.agentIDs.Add(1), tool, args, o.MustConfirm()})
	}
	return nil, agent.Refusal("no tool called " + tool)
}

// tabOf is the player's newest in-world session, or nil when they have no tab open.
func (g *Gateway) tabOf(playerID int64) *session {
	g.mu.Lock()
	defer g.mu.Unlock()
	var newest *session
	for s, a := range g.players {
		if s.player.ID == playerID && (newest == nil || a.entity > g.players[newest].entity) {
			newest = s
		}
	}
	return newest
}

// agentJob is the last job an agent started in a tab, as the tab last told it.
type agentJob struct {
	ID      uint64 `json:"job"`
	Tool    string `json:"tool"`
	State   string `json:"state"`
	Percent uint32 `json:"percent"`
	Note    string `json:"note,omitempty"`
}

// agentLink is one session's side of the relay: calls waiting on the tab, and its job.
type agentLink struct {
	mu     sync.Mutex
	calls  map[uint64]agentCall
	job    agentJob
	ended  chan struct{} // closed when the running job ends
	closed bool
	talk   *rate.Bucket // say's chat allowance, made on first use
}

type agentCall struct {
	tool  string
	reply chan string
}

func newAgentLink() *agentLink { return &agentLink{calls: map[uint64]agentCall{}} }

// agentCommand is a tool call for the tab; confirm holds any job it starts for the player's /yes.
type agentCommand struct {
	id      uint64
	tool    string
	args    json.RawMessage
	confirm bool
}

// relay sends a tool to the tab and waits for its answer: a JSON object, or a refusal.
func (l *agentLink) relay(ctx context.Context, s *session, c agentCommand) (any, error) {
	reply := make(chan string, 1)
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil, agent.Refusal("the tab just closed")
	}
	l.calls[c.id] = agentCall{c.tool, reply}
	l.mu.Unlock()
	defer l.forget(c.id)
	s.push(&pb.ServerMessage{Body: &pb.ServerMessage_AgentCommand{AgentCommand: &pb.AgentCommand{
		Id: c.id, Tool: c.tool, Json: string(c.args), Confirm: c.confirm,
	}}})
	select {
	case js := <-reply:
		return tabAnswer(js)
	case <-time.After(agentTimeout):
		return nil, agent.Refusal("the game didn't answer; keep the Scabbage tab open and in front")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (l *agentLink) forget(id uint64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.calls, id)
}

// tabAnswer turns the tab's reply into a result, or a refusal when it says "error".
func tabAnswer(js string) (any, error) {
	var out map[string]any
	if err := json.Unmarshal([]byte(js), &out); err != nil {
		return nil, agent.Refusal("the game sent a garbled answer")
	}
	if msg, ok := out["error"].(string); ok {
		return nil, agent.Refusal(msg)
	}
	return out, nil
}

// replied hands the tab's answer to the waiting call, first noting any job it started.
func (l *agentLink) replied(m *pb.AgentReply) {
	l.mu.Lock()
	defer l.mu.Unlock()
	call, ok := l.calls[m.Id]
	if !ok {
		return
	}
	delete(l.calls, m.Id) // answered once, so close can't answer it again
	var started struct {
		Job      uint64 `json:"job"`
		Awaiting bool   `json:"awaiting"`
	}
	if json.Unmarshal([]byte(m.Json), &started) == nil && started.Job == m.Id {
		l.endJob()
		l.job = agentJob{ID: m.Id, Tool: call.tool, State: "running"}
		if started.Awaiting {
			l.job.State = awaiting
		}
		l.ended = make(chan struct{})
	}
	call.reply <- m.Json
}

// progressed records how the tab says the job is going.
func (l *agentLink) progressed(m *pb.AgentProgress) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if m.Id != l.job.ID || !l.job.live() {
		return
	}
	was := l.job.State
	l.job.Percent, l.job.Note = min(m.Percent, 100), m.Note
	switch m.State {
	case pb.AgentJobState_AGENT_JOB_STATE_RUNNING:
		l.job.State = "running"
	case pb.AgentJobState_AGENT_JOB_STATE_AWAITING_CONFIRMATION:
		l.job.State = awaiting
	case pb.AgentJobState_AGENT_JOB_STATE_DONE:
		l.job.State = "done"
	case pb.AgentJobState_AGENT_JOB_STATE_STOPPED:
		l.job.State = "stopped"
	}
	if !l.job.live() {
		l.endJob()
	} else if was == awaiting && l.job.State != awaiting {
		l.endJob() // the player said yes: wake anyone waiting, then wait again for the end
		l.ended = make(chan struct{})
	}
}

// awaiting is the state of a job held for the player's /yes.
const awaiting = "awaiting_confirmation"

// live reports whether the job may still change the world: running, or waiting for /yes.
func (j agentJob) live() bool { return j.State == "running" || j.State == awaiting }

// endJob wakes anyone waiting on the job; call with mu held.
func (l *agentLink) endJob() {
	if l.ended != nil {
		close(l.ended)
		l.ended = nil
	}
}

// status reports the job, first waiting up to statusWait for it to end, or for the player to answer.
func (l *agentLink) status(ctx context.Context, args json.RawMessage) (any, error) {
	var a struct {
		Wait bool `json:"wait"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	l.mu.Lock()
	ended := l.ended
	l.mu.Unlock()
	if a.Wait && ended != nil {
		select {
		case <-ended:
		case <-time.After(statusWait):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.job.ID == 0 {
		return map[string]string{"state": "idle"}, nil
	}
	return l.job, nil
}

// close fails the calls still waiting, as the tab is gone.
func (l *agentLink) close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.closed = true
	for _, c := range l.calls {
		c.reply <- `{"error":"the tab closed before answering"}`
	}
	l.calls = map[uint64]agentCall{}
	if l.job.live() {
		l.job.State = "stopped"
	}
	l.endJob()
}
