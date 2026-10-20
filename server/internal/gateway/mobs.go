package gateway

import (
	"time"

	"scabbage/internal/drops"
	"scabbage/internal/mobs"
	"scabbage/internal/motion"
	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/world"
)

// hitEvery is the quickest one player can hit again.
const hitEvery = 250 * time.Millisecond

func mobProto(m mobs.Mob) *pb.Mob {
	return &pb.Mob{
		Id: m.ID, Kind: pb.MobKind(m.Kind), Yaw: float32(m.Yaw), Sheared: m.Sheared, Hurt: m.Hurt,
		Pos: &pb.Vec3{X: float32(m.Pos.X), Y: float32(m.Pos.Y), Z: float32(m.Pos.Z)},
	}
}

// mobsChanged lists mobs that appeared or moved and ids that went; nil when there are none.
func mobsChanged(moved []mobs.Mob, gone []uint32) *pb.ServerMessage {
	if len(moved) == 0 && len(gone) == 0 {
		return nil
	}
	out := &pb.MobsChanged{Gone: gone}
	for _, m := range moved {
		out.Mobs = append(out.Mobs, mobProto(m))
	}
	return &pb.ServerMessage{Body: &pb.ServerMessage_MobsChanged{MobsChanged: out}}
}

// stepMobsLocked moves the herd every mobs.Every, tells everyone and lets slimes bite.
func (g *Gateway) stepMobsLocked(now time.Time) {
	if now.Sub(g.mobsAt) < mobs.Every {
		return
	}
	g.mobsAt = now
	sessions := make([]*session, 0, len(g.players))
	seen := make([]mobs.Player, 0, len(g.players))
	for s, a := range g.players {
		sessions = append(sessions, s)
		seen = append(seen, mobs.Player{Pos: a.pos, Prey: a.mode == survival})
	}
	c := g.cfg.Mobs.Step(now, seen, g.cfg.World)
	if msg := mobsChanged(c.Moved, c.Gone); msg != nil {
		g.broadcastLocked(msg)
	}
	for _, i := range c.Bitten {
		g.biteLocked(sessions[i], now)
	}
}

func (g *Gateway) biteLocked(s *session, now time.Time) {
	a := g.players[s]
	fainted := a.vitals.Hurt(mobs.BiteDamage, now)
	s.push(vitalsMessage(a, pb.Harm_HARM_MOB))
	if fainted {
		s.faint()
	}
}

// hitMob hits a mob in reach; a sheep's wool drops for survival players.
func (s *session) hitMob(m *pb.HitMob) error {
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
	eye := motion.Vec{X: a.pos.X, Y: a.pos.Y + motion.EyeHeight, Z: a.pos.Z}
	hit, ok := g.cfg.Mobs.Hit(m.Mob, eye, now)
	if !ok {
		return nil
	}
	a.hitAt = now
	if hit.Gone {
		g.broadcastLocked(mobsChanged(nil, []uint32{hit.Mob.ID}))
	} else {
		g.broadcastLocked(mobsChanged([]mobs.Mob{hit.Mob}, nil))
	}
	if hit.Drop != world.Air && a.mode == survival {
		d, gone := g.drops.Spawn(hit.Drop, hit.Mob.Pos, now)
		g.changedLocked([]drops.Drop{d}, gone)
	}
	return nil
}

// mobsAroundLocked is every mob, for a player who just arrived; nil when there are none.
func (g *Gateway) mobsAroundLocked() *pb.ServerMessage {
	return mobsChanged(g.cfg.Mobs.All(), nil)
}
