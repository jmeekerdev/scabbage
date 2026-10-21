package gateway

import (
	"math"
	"strings"

	"scabbage/internal/logic"
	"scabbage/internal/motion"
	"scabbage/internal/world"
)

// checkpoint is the pad a player comes back to after fainting, once they've stepped on one.
type checkpoint struct {
	at  world.Pos
	set bool
}

// reachCheckpoint remembers the checkpoint a player stepped on, telling them the first time.
func (g *Gateway) reachCheckpoint(s *session, p logic.Pos) {
	g.mu.Lock()
	defer g.mu.Unlock()
	a := g.players[s]
	cp := checkpoint{at: world.Pos(p), set: true}
	if a == nil || a.checkpoint == cp {
		return
	}
	a.checkpoint = cp
	s.push(noticeMessage("🚩 Checkpoint! If you faint, you'll come back here."))
}

// comeBackTo is where a fainted player wakes: their checkpoint, if it is still there and in the
// zone they fainted in (or both are outside every zone), else the world spawn.
func (g *Gateway) comeBackTo(s *session) motion.Vec {
	g.mu.Lock()
	var cp checkpoint
	var fell motion.Vec
	if a := g.players[s]; a != nil {
		cp, fell = a.checkpoint, a.pos
	}
	g.mu.Unlock()
	if cp.set && g.zoneName(cp.at.X, cp.at.Z) == g.zoneName(int32(math.Floor(fell.X)), int32(math.Floor(fell.Z))) {
		if b, err := g.cfg.World.Block(cp.at.X, cp.at.Y, cp.at.Z); err == nil && b == world.Checkpoint {
			return motion.Vec{X: float64(cp.at.X) + 0.5, Y: float64(cp.at.Y), Z: float64(cp.at.Z) + 0.5}
		}
	}
	x, y, z := g.cfg.World.Spawn()
	return motion.Vec{X: x, Y: y, Z: z}
}

// zoneName is the lowercased name of the zone over a column, or "" outside every zone.
func (g *Gateway) zoneName(x, z int32) string {
	if zone, ok := g.cfg.Zones.At(x, z); ok {
		return strings.ToLower(zone.Name)
	}
	return ""
}
