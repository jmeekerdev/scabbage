package gateway

import (
	"context"
	"errors"
	"time"

	"scabbage/internal/access"
	pb "scabbage/internal/pb/scabbage/v1"
)

const (
	maxCodeUses     = 50
	defaultCodeTTL  = 24
	maxCodeTTLHours = 30 * 24
)

func (s *session) requireAdmin() error {
	if !s.player.Admin {
		return reject(pb.DisconnectCode_DISCONNECT_CODE_NOT_ADMIN, "admins only")
	}
	return nil
}

func (s *session) decideJoin(ctx context.Context, d *pb.DecideJoin) error {
	if err := s.requireAdmin(); err != nil {
		return err
	}
	err := s.gw.cfg.Access.Decide(int64(d.RequestId), d.Approve)
	if errors.Is(err, access.ErrNotFound) {
		return s.sendPending(ctx) // someone else decided first; refresh this admin's list
	}
	if err != nil {
		return err
	}
	s.log.Info("join request decided", "request", d.RequestId, "approve", d.Approve, "by", s.player.Name)
	if d.Approve {
		s.gw.PlayersChanged()
	}
	return nil
}

func (s *session) createJoinCode(ctx context.Context, c *pb.CreateJoinCode) error {
	if err := s.requireAdmin(); err != nil {
		return err
	}
	uses := clamp(int(c.Uses), 1, maxCodeUses)
	ttl := clamp(int(c.TtlHours), 1, maxCodeTTLHours)
	if c.TtlHours == 0 {
		ttl = defaultCodeTTL
	}
	code, err := s.gw.cfg.Access.CreateCode(uses, time.Duration(ttl)*time.Hour)
	if err != nil {
		return err
	}
	s.log.Info("join code created", "uses", uses, "ttl_hours", ttl, "by", s.player.Name)
	return s.write(ctx, &pb.ServerMessage{Body: &pb.ServerMessage_JoinCodeCreated{JoinCodeCreated: &pb.JoinCodeCreated{
		Code: code.Code, Uses: uint32(code.Uses), ExpiresMs: float64(code.Expires.UnixMilli()),
	}}})
}

func (s *session) sendPending(ctx context.Context) error {
	msg, err := s.gw.pendingMessage()
	if err != nil {
		return err
	}
	return s.write(ctx, msg)
}

func (g *Gateway) pendingMessage() (*pb.ServerMessage, error) {
	reqs, err := g.cfg.Access.Pending()
	if err != nil {
		return nil, err
	}
	list := &pb.PendingRequests{Requests: make([]*pb.PendingRequest, 0, len(reqs))}
	for _, r := range reqs {
		list.Requests = append(list.Requests, &pb.PendingRequest{
			Id: uint64(r.ID), Name: r.Name, Color: r.Color, Returning: r.Returning,
			CreatedMs: float64(r.Created.UnixMilli()),
		})
	}
	return &pb.ServerMessage{Body: &pb.ServerMessage_PendingRequests{PendingRequests: list}}, nil
}

func (g *Gateway) playerInfo(p access.Player, skin []byte) *pb.PlayerInfo {
	return &pb.PlayerInfo{Id: uint64(p.ID), Name: p.Name, Color: p.Color, Admin: p.Admin, Skin: skin, Mode: g.modeOf(p)}
}

func clamp(v, lo, hi int) int { return max(lo, min(v, hi)) }
