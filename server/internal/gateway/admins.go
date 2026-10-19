package gateway

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"scabbage/internal/access"
	pb "scabbage/internal/pb/scabbage/v1"
)

// setAdmin grants or removes another player's admin rights; they reconnect to pick it up.
func (s *session) setAdmin(ctx context.Context, m *pb.SetAdmin) error {
	if err := s.requireAdmin(); err != nil {
		return err
	}
	if int64(m.PlayerId) == s.player.ID {
		return s.notice(ctx, "You can't change your own admin rights. Ask another admin.")
	}
	p, err := s.gw.cfg.Access.SetAdmin(int64(m.PlayerId), m.Admin)
	if errors.Is(err, access.ErrNoPlayer) {
		return s.sendPlayers(ctx)
	}
	if err != nil {
		return err
	}
	s.log.Info("admin rights changed", "player", p.Name, "admin", p.Admin, "by", s.player.Name)
	s.gw.reconnect(p.ID)
	s.gw.PlayersChanged()
	if p.Admin {
		return s.notice(ctx, p.Name+" is an admin now.")
	}
	return s.notice(ctx, p.Name+" is no longer an admin.")
}

// setAdminByName runs /admin and /unadmin.
func (s *session) setAdminByName(ctx context.Context, name string, admin bool) error {
	p, found, err := s.gw.playerNamed(name)
	if err != nil || !found {
		return s.noOneCalled(ctx, name, err)
	}
	return s.setAdmin(ctx, &pb.SetAdmin{PlayerId: uint64(p.ID), Admin: admin})
}

// playerNamed finds an approved player, ignoring case.
func (g *Gateway) playerNamed(name string) (access.Player, bool, error) {
	players, err := g.cfg.Access.Players()
	if err != nil {
		return access.Player{}, false, err
	}
	for _, p := range players {
		if strings.EqualFold(p.Name, name) {
			return p, true, nil
		}
	}
	return access.Player{}, false, nil
}

// noOneCalled reports a failed name lookup, passing on any error behind it.
func (s *session) noOneCalled(ctx context.Context, name string, err error) error {
	if err != nil {
		return err
	}
	return s.notice(ctx, fmt.Sprintf("No one is called %q.", name))
}

// reconnect drops every connection of a player so their client comes back with fresh rights.
func (g *Gateway) reconnect(playerID int64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for s := range g.players {
		if s.player.ID == playerID {
			s.cancel()
		}
	}
	for s := range g.admins {
		if s.player.ID == playerID {
			s.cancel()
		}
	}
}

// PlayersChanged pushes the player list to every connected admin.
func (g *Gateway) PlayersChanged() {
	msg, err := g.playersMessage()
	if err != nil {
		g.cfg.Logger.Error("listing players", "err", err)
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	for s := range g.admins {
		s.push(msg)
	}
}

func (s *session) sendPlayers(ctx context.Context) error {
	msg, err := s.gw.playersMessage()
	if err != nil {
		return err
	}
	return s.write(ctx, msg)
}

func (g *Gateway) playersMessage() (*pb.ServerMessage, error) {
	players, err := g.cfg.Access.Players()
	if err != nil {
		return nil, err
	}
	list := &pb.PlayerList{Players: make([]*pb.PlayerInfo, 0, len(players))}
	for _, p := range players {
		list.Players = append(list.Players, g.playerInfo(p, nil))
	}
	return &pb.ServerMessage{Body: &pb.ServerMessage_PlayerList{PlayerList: list}}, nil
}
