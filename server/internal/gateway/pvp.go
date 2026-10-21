package gateway

import (
	"math"
	"strings"
	"time"

	"scabbage/internal/motion"
	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/world"
)

const (
	playerDamage = 2                 // a heart
	playerReach  = 5.5 * world.Metre // from the hitter's eye to the other's middle, with lag
)

// hitPlayer hurts another survival player in reach when both stand in the same PVP arena.
func (s *session) hitPlayer(m *pb.HitPlayer) error {
	if s.console {
		return reject(pb.DisconnectCode_DISCONNECT_CODE_PROTOCOL, "the console has no hands")
	}
	g := s.gw
	g.mu.Lock()
	defer g.mu.Unlock()
	a := g.players[s]
	now := time.Now()
	if a == nil || now.Sub(a.hitAt) < hitEvery {
		return nil
	}
	target, b := g.playerByEntityLocked(m.Entity)
	if b == nil || b == a || !g.mayHitLocked(a, b) {
		return nil
	}
	a.hitAt = now
	if r := a.round; r != nil && r == b.round && r.Tagging() {
		g.fallLocked(b) // one tag and you're out
		return nil
	}
	fainted := b.vitals.Hurt(playerDamage, now)
	target.push(vitalsMessage(b, pb.Harm_HARM_PLAYER))
	if fainted {
		target.faint()
	}
	return nil
}

func (g *Gateway) playerByEntityLocked(entity uint32) (*session, *avatar) {
	for s, a := range g.players {
		if a.entity == entity {
			return s, a
		}
	}
	return nil, nil
}

// mayHitLocked reports whether a can hit b: both in survival, in reach, inside one PVP arena.
func (g *Gateway) mayHitLocked(a, b *avatar) bool {
	if a.mode != survival || b.mode != survival {
		return false
	}
	eye := motion.Vec{X: a.pos.X, Y: a.pos.Y + motion.EyeHeight, Z: a.pos.Z}
	middle := motion.Vec{X: b.pos.X, Y: b.pos.Y + motion.Height/2, Z: b.pos.Z}
	if motion.Distance(eye, middle) > playerReach {
		return false
	}
	za, inA := g.cfg.Zones.At(int32(math.Floor(a.pos.X)), int32(math.Floor(a.pos.Z)))
	zb, inB := g.cfg.Zones.At(int32(math.Floor(b.pos.X)), int32(math.Floor(b.pos.Z)))
	return inA && inB && za.Name == zb.Name && g.arenas[strings.ToLower(za.Name)].arena.PVP
}
