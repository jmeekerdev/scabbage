package mobs

import (
	"time"

	"scabbage/internal/motion"
)

const (
	shedEvery = 6            // a king slime lets out a slime each time it loses this much health
	bossRoom  = 32.0 * metre // no two bosses wake this close together
)

// Summon wakes boss k at pos unless another boss is already up nearby.
func (h *Herd) Summon(k Kind, pos motion.Vec, now time.Time) (Mob, bool) {
	if !kinds[k].boss {
		return Mob{}, false
	}
	for _, m := range h.mobs {
		if m.Boss() && motion.Distance(m.Pos, pos) < bossRoom {
			return Mob{}, false
		}
	}
	return h.Add(k, pos, now), true
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
