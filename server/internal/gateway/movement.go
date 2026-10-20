package gateway

import (
	"context"
	"time"

	"scabbage/internal/edit"
	"scabbage/internal/motion"
	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/rate"
)

// startMotion tracks the player from where they start.
func (s *session) startMotion() {
	now := time.Now()
	s.body = motion.NewTracker(motion.DefaultLimits, s.saved.pos, now)
	s.edits = rate.New(now, edit.PerSecond, edit.Burst)
}

// move accepts a plausible move, or sends the player back to their last accepted position.
func (s *session) move(ctx context.Context, m *pb.Move) error {
	if s.console {
		return reject(pb.DisconnectCode_DISCONNECT_CODE_PROTOCOL, "the console has no body")
	}
	if m.Teleport != s.teleport {
		return nil // sent before the client saw our latest teleport
	}
	p := m.GetPos()
	to := motion.Vec{X: float64(p.GetX()), Y: float64(p.GetY()), Z: float64(p.GetZ())}
	if s.body.Move(to, time.Now(), s.gw.cfg.World.Boxes) {
		s.gw.movePlayer(s, to, m)
		if s.gw.land(s, to.Y, s.gw.footing(to)) {
			s.respawn()
			return nil
		}
		s.stepOnPad(to)
		return nil
	}
	s.teleport++
	at := s.body.Pos()
	return s.write(ctx, &pb.ServerMessage{Body: &pb.ServerMessage_Teleport{Teleport: &pb.Teleport{
		Id: s.teleport, Pos: &pb.Vec3{X: float32(at.X), Y: float32(at.Y), Z: float32(at.Z)},
	}}})
}
