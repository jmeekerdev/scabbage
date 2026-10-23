package gateway

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"scabbage/internal/agent"
)

const (
	inboxKept   = 20               // unread /ai messages kept per player
	listenGrace = time.Minute      // an agent that listened this recently counts as listening
	listenMax   = 2 * time.Minute  // the longest listen may wait
	listenWait  = 20 * time.Second // how long listen waits when not told
)

// inbox is the player's /ai inbox, made on first use.
func (g *Gateway) inbox(playerID int64) *aiInbox {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.inboxes == nil {
		g.inboxes = map[int64]*aiInbox{}
	}
	if g.inboxes[playerID] == nil {
		g.inboxes[playerID] = &aiInbox{}
	}
	return g.inboxes[playerID]
}

// aiCommand runs /ai: a message for the player's own agent, which only they see.
func (s *session) aiCommand(ctx context.Context, text string) error {
	if s.gw.cfg.Agents == nil {
		return s.notice(ctx, "Agents are off on this server.")
	}
	if text == "" {
		return s.notice(ctx, "/ai MESSAGE asks your agent for something, e.g. /ai build a treehouse here.")
	}
	note := "To your agent: " + text
	if !s.gw.inbox(s.player.ID).post(text, time.Now()) {
		note += " Your agent isn't listening right now; it gets this when it next calls listen. " +
			"Ask it to listen for your /ai messages."
	}
	return s.notice(ctx, note)
}

// listen is the listen tool: the player's /ai messages, first waiting for one.
func (g *Gateway) listen(ctx context.Context, o agent.Owner, args json.RawMessage) (any, error) {
	var a struct {
		Seconds *int `json:"seconds"`
	}
	if err := decodeArgs(args, &a); err != nil {
		return nil, err
	}
	wait := listenWait
	if a.Seconds != nil {
		wait = min(max(time.Duration(*a.Seconds)*time.Second, 0), listenMax)
	}
	return map[string]any{"messages": g.inbox(o.PlayerID).take(ctx, wait, time.Now())}, nil
}

// aiInbox holds a player's /ai messages until their agent listens.
type aiInbox struct {
	mu        sync.Mutex
	msgs      []aiMessage
	arrived   chan struct{} // closed when a message comes, then replaced
	waiting   int           // listen calls waiting now
	lastHeard time.Time     // when an agent last listened
}

type aiMessage struct {
	Text       string `json:"text"`
	SecondsAgo int    `json:"seconds_ago"`
	at         time.Time
}

// post keeps a message for the agent, saying whether one is listening.
func (b *aiInbox) post(text string, now time.Time) (heard bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.msgs = append(b.msgs, aiMessage{Text: text, at: now})
	if len(b.msgs) > inboxKept {
		b.msgs = b.msgs[len(b.msgs)-inboxKept:]
	}
	if b.arrived != nil {
		close(b.arrived)
		b.arrived = nil
	}
	return b.waiting > 0 || (!b.lastHeard.IsZero() && now.Sub(b.lastHeard) < listenGrace)
}

// take hands over the waiting messages, first waiting up to wait for one to come.
func (b *aiInbox) take(ctx context.Context, wait time.Duration, now time.Time) []aiMessage {
	b.mu.Lock()
	b.lastHeard = now
	if len(b.msgs) == 0 && wait > 0 {
		if b.arrived == nil {
			b.arrived = make(chan struct{})
		}
		arrived := b.arrived
		b.waiting++
		b.mu.Unlock()
		select {
		case <-arrived:
		case <-time.After(wait):
		case <-ctx.Done():
		}
		b.mu.Lock()
		b.waiting--
		b.lastHeard = time.Now()
	}
	defer b.mu.Unlock()
	out := b.msgs
	b.msgs = nil
	for i := range out {
		out[i].SecondsAgo = max(0, int(now.Sub(out[i].at).Seconds()))
	}
	if out == nil {
		out = []aiMessage{}
	}
	return out
}
