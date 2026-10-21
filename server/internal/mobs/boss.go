package mobs

import (
	"math"
	"time"

	"scabbage/internal/motion"
)

const (
	shedEvery = 6            // a king slime lets out a slime each time it loses this much health
	bossRoom  = 32.0 * metre // no two bosses wake this close together
)

// Summon wakes boss k at pos unless another boss is already up nearby.
func (h *Herd) Summon(k Kind, pos motion.Vec, g Ground, now time.Time) (Mob, bool) {
	if !kinds[k].boss {
		return Mob{}, false
	}
	for _, m := range h.mobs {
		if m.Boss() && motion.Distance(m.Pos, pos) < bossRoom {
			return Mob{}, false
		}
	}
	return h.Add(k, roomFor(k, pos, g), now), true
}

// roomFor is the nearest spot to pos where a boss of kind k fits, or pos itself if none is close.
func roomFor(k Kind, pos motion.Vec, g Ground) motion.Vec {
	m := Mob{Kind: k, Pos: pos}
	reach := int(math.Ceil(2 * kinds[k].halfWidth))
	for r := range reach + 1 {
		for dx := -r; dx <= r; dx++ {
			for dz := -r; dz <= r; dz++ {
				if max(dx, -dx, dz, -dz) != r {
					continue // only the ring r blocks out
				}
				x, z := pos.X+float64(dx), pos.Z+float64(dz)
				if y, ok := m.footing(x, z, g); ok {
					return motion.Vec{X: x, Y: y, Z: z}
				}
			}
		}
	}
	return pos
}

// BossName is what players call boss k.
func BossName(k Kind) string {
	switch k {
	case KingSlime:
		return "King Slime"
	case Golem:
		return "Stone Golem"
	}
	return ""
}
