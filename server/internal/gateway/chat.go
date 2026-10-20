package gateway

import (
	"context"
	"fmt"
	"strings"
	"time"

	"scabbage/internal/chat"
	pb "scabbage/internal/pb/scabbage/v1"
)

const commandHelp = "Commands: /approve NAME, /deny NAME, /admin NAME, /unadmin NAME, /creative NAME, /survival NAME, /zone"

// sendChat runs a command or shares a line with everyone in the world.
func (s *session) sendChat(ctx context.Context, m *pb.SendChat) error {
	if s.console {
		return reject(pb.DisconnectCode_DISCONNECT_CODE_PROTOCOL, "the console has no voice")
	}
	text, ok := chat.Clean(m.Text)
	if !ok {
		return nil
	}
	if !s.talk.Take(time.Now()) {
		return s.notice(ctx, "Slow down a little!")
	}
	if cmd, ok := chat.ParseCommand(text); ok {
		return s.command(ctx, cmd)
	}
	s.log.Info("chat", "name", s.player.Name, "text", text)
	s.gw.broadcast(&pb.ServerMessage{Body: &pb.ServerMessage_Chat{Chat: &pb.Chat{
		Name: s.player.Name, Color: s.player.Color, Text: text,
	}}})
	return nil
}

// sendEmote shows a known emote to everyone else; others are dropped like spam.
func (s *session) sendEmote(m *pb.SendEmote) error {
	if s.console {
		return reject(pb.DisconnectCode_DISCONNECT_CODE_PROTOCOL, "the console has no face")
	}
	if _, known := pb.Emote_name[int32(m.Emote)]; !known || m.Emote == pb.Emote_EMOTE_UNSPECIFIED {
		return nil
	}
	if s.talk.Take(time.Now()) {
		s.gw.emote(s, m.Emote)
	}
	return nil
}

func (s *session) command(ctx context.Context, c chat.Command) error {
	if !s.player.Admin {
		return s.notice(ctx, "Only admins can use commands.")
	}
	switch c.Name {
	case "approve":
		return s.decideByName(ctx, c.Arg, true)
	case "deny":
		return s.decideByName(ctx, c.Arg, false)
	case "admin":
		return s.setAdminByName(ctx, c.Arg, true)
	case "unadmin":
		return s.setAdminByName(ctx, c.Arg, false)
	case "creative":
		return s.setModeByName(ctx, c.Arg, creative)
	case "survival":
		return s.setModeByName(ctx, c.Arg, survival)
	case "zone":
		return s.zoneCommand(ctx, c.Arg)
	default:
		return s.notice(ctx, commandHelp)
	}
}

// decideByName answers the waiting join request with this name, ignoring case.
func (s *session) decideByName(ctx context.Context, name string, approve bool) error {
	reqs, err := s.gw.cfg.Access.Pending()
	if err != nil {
		return err
	}
	for _, r := range reqs {
		if !strings.EqualFold(r.Name, name) {
			continue
		}
		if err := s.decideJoin(ctx, &pb.DecideJoin{RequestId: uint64(r.ID), Approve: approve}); err != nil {
			return err
		}
		verb := "Turned away"
		if approve {
			verb = "Let in"
		}
		return s.notice(ctx, fmt.Sprintf("%s %s.", verb, r.Name))
	}
	return s.notice(ctx, fmt.Sprintf("No one called %q is waiting.", name))
}

// notice sends a server message to this player only.
func (s *session) notice(ctx context.Context, text string) error {
	return s.write(ctx, &pb.ServerMessage{Body: &pb.ServerMessage_Chat{Chat: &pb.Chat{Text: text, Notice: true}}})
}

// emote shows s's emote above its avatar for everyone else.
func (g *Gateway) emote(s *session, e pb.Emote) {
	g.mu.Lock()
	defer g.mu.Unlock()
	a := g.players[s]
	if a == nil {
		return
	}
	msg := &pb.ServerMessage{Body: &pb.ServerMessage_EmoteShown{EmoteShown: &pb.EmoteShown{Entity: a.entity, Emote: e}}}
	for o := range g.players {
		if o != s {
			o.push(msg)
		}
	}
}
