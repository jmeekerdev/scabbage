package gateway

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"scabbage/internal/agent"
	pb "scabbage/internal/pb/scabbage/v1"
)

const agentHelp = "Agents: /agent link [label] connects one (Claude Code or any MCP client), /agent list, " +
	"/agent revoke LABEL|all, /ai MESSAGE talks to yours; admins: /agent allow NAME, /agent deny NAME, /agent confirm on|off"

// agentCommand runs /agent: linking, listing and revoking agents, and admins turning them on.
func (s *session) agentCommand(ctx context.Context, arg string) error {
	if s.gw.cfg.Agents == nil {
		return s.notice(ctx, "Agents aren't set up on this server.")
	}
	sub, rest, _ := strings.Cut(arg, " ")
	rest = strings.TrimSpace(rest)
	switch strings.ToLower(sub) {
	case "link":
		return s.linkAgent(ctx, rest)
	case "list":
		return s.listAgents(ctx)
	case "revoke":
		return s.revokeAgent(ctx, rest)
	case "allow", "deny":
		return s.allowAgents(ctx, rest, strings.EqualFold(sub, "allow"))
	case "confirm":
		return s.confirmAgents(ctx, rest)
	}
	return s.notice(ctx, agentHelp)
}

func (s *session) linkAgent(ctx context.Context, label string) error {
	if label == "" {
		label = agent.DefaultLabel
	}
	if ok, err := s.mayUseAgents(); err != nil || !ok {
		if err != nil {
			return err
		}
		return s.notice(ctx, "Ask an admin to turn agents on for you with /agent allow "+s.player.Name+".")
	}
	token, err := s.gw.cfg.Agents.Create(s.player.ID, label)
	if errors.Is(err, agent.ErrBadLabel) || errors.Is(err, agent.ErrTooMany) {
		return s.notice(ctx, "Can't link: "+err.Error()+".")
	}
	if err != nil {
		return err
	}
	s.log.Info("agent linked", "name", s.player.Name, "label", label)
	return s.write(ctx, &pb.ServerMessage{Body: &pb.ServerMessage_AgentLinked{AgentLinked: &pb.AgentLinked{
		Label: label, Token: token,
	}}})
}

func (s *session) mayUseAgents() (bool, error) {
	if s.player.Admin {
		return true, nil
	}
	return s.gw.cfg.Agents.Allowed(s.player.ID)
}

func (s *session) listAgents(ctx context.Context) error {
	tokens, err := s.gw.cfg.Agents.List(s.player.ID)
	if err != nil {
		return err
	}
	if len(tokens) == 0 {
		return s.notice(ctx, "No agents linked. /agent link connects one.")
	}
	lines := make([]string, 0, len(tokens))
	for _, t := range tokens {
		used := "never used"
		if !t.LastUsed.IsZero() {
			used = "last used " + t.LastUsed.Format("Jan 2 15:04")
		}
		lines = append(lines, fmt.Sprintf("%s (%s)", t.Label, used))
	}
	return s.notice(ctx, "Agents: "+strings.Join(lines, ", ")+".")
}

func (s *session) revokeAgent(ctx context.Context, label string) error {
	if label == "" {
		return s.notice(ctx, "Use /agent revoke LABEL, or /agent revoke all.")
	}
	n, err := s.gw.cfg.Agents.Revoke(s.player.ID, label)
	if err != nil {
		return err
	}
	if n == 0 {
		return s.notice(ctx, fmt.Sprintf("No agent called %q. /agent list shows them.", label))
	}
	s.log.Info("agent revoked", "name", s.player.Name, "label", label)
	return s.notice(ctx, fmt.Sprintf("Revoked %d agent token%s.", n, plural(n)))
}

// allowAgents turns agents on or off for a player; turning them off drops their tokens.
func (s *session) allowAgents(ctx context.Context, name string, on bool) error {
	if !s.player.Admin {
		return s.notice(ctx, "Only admins can turn agents on or off.")
	}
	p, found, err := s.gw.playerNamed(name)
	if err != nil || !found {
		return s.noOneCalled(ctx, name, err)
	}
	if err := s.gw.cfg.Agents.Allow(p.ID, on); err != nil {
		return err
	}
	s.log.Info("agents allowed", "player", p.Name, "on", on, "by", s.player.Name)
	if on {
		return s.notice(ctx, p.Name+" may link agents now, with /agent link.")
	}
	return s.notice(ctx, p.Name+" can't use agents now; their links are revoked.")
}

// confirmAgents sets whether an admin's own agents' jobs wait for their /yes.
func (s *session) confirmAgents(ctx context.Context, arg string) error {
	if !s.player.Admin {
		return s.notice(ctx, "Your agents' jobs always wait for you to say /yes.")
	}
	on := strings.EqualFold(arg, "on")
	if !on && !strings.EqualFold(arg, "off") {
		return s.notice(ctx, "Use /agent confirm on or /agent confirm off.")
	}
	if err := s.gw.cfg.Agents.SetConfirm(s.player.ID, on); err != nil {
		return err
	}
	s.log.Info("agent confirm", "name", s.player.Name, "on", on)
	if on {
		return s.notice(ctx, "Your agents' jobs will wait for you to say /yes.")
	}
	return s.notice(ctx, "Your agents will build without asking; /stop and /undo still work.")
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
