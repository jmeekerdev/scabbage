package gateway

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"scabbage/internal/access"
	"scabbage/internal/motion"
	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/vitals"
	"scabbage/internal/world"
)

const (
	creative = pb.GameMode_GAME_MODE_CREATIVE
	survival = pb.GameMode_GAME_MODE_SURVIVAL
)

// modeOf resolves a player's game mode, falling back to the server's.
func (g *Gateway) modeOf(p access.Player) pb.GameMode {
	m := p.Mode
	if m == access.ModeDefault {
		m = g.cfg.Mode
	}
	if m == access.ModeSurvival {
		return survival
	}
	return creative
}

// storedMode is how a mode from the wire is kept; false for modes this server doesn't know.
func storedMode(m pb.GameMode) (access.Mode, bool) {
	switch m {
	case pb.GameMode_GAME_MODE_UNSPECIFIED:
		return access.ModeDefault, true
	case creative:
		return access.ModeCreative, true
	case survival:
		return access.ModeSurvival, true
	}
	return "", false
}

func modeName(m pb.GameMode) string {
	if m == survival {
		return "survival"
	}
	return "creative"
}

func vitalsMessage(a *avatar, hurt pb.Harm) *pb.ServerMessage {
	return &pb.ServerMessage{Body: &pb.ServerMessage_Vitals{Vitals: &pb.Vitals{
		Mode: a.mode, Health: uint32(a.vitals.Health), Hurt: hurt,
	}}}
}

// footing tells what holds up a body with its feet at p: water at its feet or chest, or the ground.
func (g *Gateway) footing(p motion.Vec) vitals.Footing {
	w := g.cfg.World
	x, z := int32(math.Floor(p.X)), int32(math.Floor(p.Z))
	for _, y := range []float64{p.Y, p.Y + 1} {
		if b, err := w.Block(x, int32(math.Floor(y)), z); err == nil && b == world.Water {
			return vitals.Water
		}
	}
	if motion.Standing(p, w.Boxes) {
		return vitals.Ground
	}
	return vitals.Air
}

// land follows an accepted move for falls in survival and reports whether the player fainted.
func (g *Gateway) land(s *session, y float64, f vitals.Footing) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	a := g.players[s]
	if a == nil || a.mode != survival {
		return false
	}
	damage := a.vitals.Fall(y, f)
	if damage == 0 {
		return false
	}
	fainted := a.vitals.Hurt(damage, time.Now())
	s.push(vitalsMessage(a, pb.Harm_HARM_FALL))
	return fainted
}

// respawn puts a player who fainted back where their minigame says, at their checkpoint or at the
// world spawn, with full health.
func (s *session) respawn() {
	if !s.gw.fallInRound(s) {
		s.teleportTo(s.gw.comeBackTo(s), nil)
	}
	s.gw.revive(s)
}

// teleportTo moves the player to at and tells their client, with how they were facing when restoring a save.
func (s *session) teleportTo(at motion.Vec, restore *pb.Restore) {
	s.teleport++
	s.pad = blockAt(at) // arriving on a pad doesn't step on it
	s.body.Reset(at, time.Now())
	s.gw.place(s, &pb.Teleport{Id: s.teleport, Pos: vec3(at), Restore: restore})
}

func (g *Gateway) place(s *session, t *pb.Teleport) {
	g.mu.Lock()
	defer g.mu.Unlock()
	a := g.players[s]
	if a == nil {
		return
	}
	a.pos, a.moved = motion.Vec{X: float64(t.Pos.X), Y: float64(t.Pos.Y), Z: float64(t.Pos.Z)}, true
	if t.Restore != nil {
		a.yaw, a.pitch, a.flying = t.Restore.Yaw, t.Restore.Pitch, t.Restore.Flying
	}
	s.push(&pb.ServerMessage{Body: &pb.ServerMessage_Teleport{Teleport: t}})
}

func vec3(v motion.Vec) *pb.Vec3 { return &pb.Vec3{X: float32(v.X), Y: float32(v.Y), Z: float32(v.Z)} }

func (g *Gateway) revive(s *session) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if a := g.players[s]; a != nil {
		a.vitals.Revive(time.Now())
		s.push(vitalsMessage(a, pb.Harm_HARM_UNSPECIFIED))
	}
}

// healLocked lets survival players slowly heal; call with g.mu held.
func (g *Gateway) healLocked(now time.Time) {
	for s, a := range g.players {
		if a.mode == survival && a.vitals.Heal(now) {
			s.push(vitalsMessage(a, pb.Harm_HARM_UNSPECIFIED))
		}
	}
}

// setGameMode changes a player's mode, starting them afresh at full health.
func (s *session) setGameMode(ctx context.Context, m *pb.SetGameMode) error {
	if err := s.requireAdmin(); err != nil {
		return err
	}
	mode, ok := storedMode(m.Mode)
	if !ok {
		return nil
	}
	p, err := s.gw.cfg.Access.SetMode(int64(m.PlayerId), mode)
	if errors.Is(err, access.ErrNoPlayer) {
		return s.sendPlayers(ctx)
	}
	if err != nil {
		return err
	}
	now := s.gw.modeOf(p)
	s.log.Info("game mode changed", "player", p.Name, "mode", modeName(now), "by", s.player.Name)
	s.gw.changeMode(p.ID, now)
	s.gw.PlayersChanged()
	return s.notice(ctx, fmt.Sprintf("%s plays %s now.", p.Name, modeName(now)))
}

// setModeByName runs /creative and /survival.
func (s *session) setModeByName(ctx context.Context, name string, m pb.GameMode) error {
	p, found, err := s.gw.playerNamed(name)
	if err != nil || !found {
		return s.noOneCalled(ctx, name, err)
	}
	return s.setGameMode(ctx, &pb.SetGameMode{PlayerId: uint64(p.ID), Mode: m})
}

func (g *Gateway) changeMode(playerID int64, m pb.GameMode) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for s, a := range g.players {
		if s.player.ID == playerID {
			a.own = m
			g.refreshModeLocked(s, a, true)
		}
	}
}
