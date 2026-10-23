package gateway

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

const chatKept = 20 // lines the chat tool can show

// chatLog keeps the last lines said in chat, so an agent can hear the player, and wait for them.
type chatLog struct {
	mu      sync.Mutex
	lines   []chatLine
	last    uint64
	changed chan struct{} // closed when a player says something, then replaced
}

type chatLine struct {
	ID         uint64 `json:"id"`
	Name       string `json:"name"`
	Text       string `json:"text"`
	Agent      bool   `json:"agent,omitempty"` // said by an agent with say
	SecondsAgo int    `json:"seconds_ago"`
	at         time.Time
}

// chatSince is the chat tool's answer: lines after the asked-for id, and the newest id.
type chatSince struct {
	Lines []chatLine `json:"lines"`
	Last  uint64     `json:"last"`
}

// agentChat is the chat tool: lines after `after`, first waiting up to statusWait for a player to speak.
func (g *Gateway) agentChat(ctx context.Context, args json.RawMessage) (any, error) {
	var a struct {
		After uint64 `json:"after"`
		Wait  bool   `json:"wait"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	if a.Wait {
		g.chats.wait(ctx, a.After, statusWait)
	}
	return g.chats.since(a.After, time.Now()), nil
}

func (l *chatLog) add(name, text string, agent bool, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.last++
	l.lines = append(l.lines, chatLine{ID: l.last, Name: name, Text: text, Agent: agent, at: now})
	if len(l.lines) > chatKept {
		l.lines = l.lines[len(l.lines)-chatKept:]
	}
	if !agent && l.changed != nil {
		close(l.changed)
		l.changed = nil
	}
}

// since lists the kept lines after id, oldest first.
func (l *chatLog) since(id uint64, now time.Time) chatSince {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := chatSince{Lines: []chatLine{}, Last: l.last}
	for _, c := range l.lines {
		if c.ID > id {
			c.SecondsAgo = int(now.Sub(c.at).Seconds())
			out.Lines = append(out.Lines, c)
		}
	}
	return out
}

// wait waits up to max for a player's line after id, saying whether one came.
func (l *chatLog) wait(ctx context.Context, id uint64, max time.Duration) bool {
	deadline := time.After(max)
	for {
		l.mu.Lock()
		if l.playerSaidAfter(id) {
			l.mu.Unlock()
			return true
		}
		if l.changed == nil {
			l.changed = make(chan struct{})
		}
		changed := l.changed
		l.mu.Unlock()
		select {
		case <-changed:
		case <-deadline:
			return false
		case <-ctx.Done():
			return false
		}
	}
}

func (l *chatLog) playerSaidAfter(id uint64) bool {
	for _, c := range l.lines {
		if c.ID > id && !c.Agent {
			return true
		}
	}
	return false
}
