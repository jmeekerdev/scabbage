package gateway

import (
	"context"
	"time"

	"scabbage/internal/motion"
	pb "scabbage/internal/pb/scabbage/v1"
)

// startMotion tracks the player from the world spawn.
func (s *session) startMotion() {
	x, y, z := s.gw.cfg.World.Spawn()
	s.body = motion.NewTracker(motion.DefaultLimits, motion.Vec{X: x, Y: y, Z: z}, time.Now())
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
	if s.body.Move(to, time.Now(), s.gw.cfg.World.Solid) {
		return nil
	}
	s.teleport++
	at := s.body.Pos()
	return s.write(ctx, &pb.ServerMessage{Body: &pb.ServerMessage_Teleport{Teleport: &pb.Teleport{
		Id: s.teleport, Pos: &pb.Vec3{X: float32(at.X), Y: float32(at.Y), Z: float32(at.Z)},
	}}})
}
