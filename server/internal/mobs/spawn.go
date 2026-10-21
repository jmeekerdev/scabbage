package mobs

import (
	"math"
	"time"

	"scabbage/internal/drops"
	"scabbage/internal/motion"
	"scabbage/internal/world"
)

const (
	spawnEvery      = time.Second
	near            = 16.0 // mobs appear no closer to anyone than this...
	spawnFar        = 32.0 // ...and no farther than this from someone
	leaveRange      = 64.0 // and vanish beyond this from everyone
	maxSheep        = 10
	monstersPerPrey = 4
	maxMonsters     = 16
	meltOdds        = 20 // a slime melts on one daytime step in this many
)

// leaves reports whether a mob goes away this step: far from everyone, or a slime in daylight.
func (h *Herd) leaves(m *Mob, players []Player, night bool) bool {
	if k := kinds[m.Kind]; k.chases && !k.boss && !night && h.rng.IntN(meltOdds) == 0 {
		return true
	}
	for _, p := range players {
		if math.Hypot(p.Pos.X-m.Pos.X, p.Pos.Z-m.Pos.Z) <= leaveRange {
			return false
		}
	}
	return true
}

// spawn now and then tries to add a slime near prey at night, or a sheep on grass by day.
func (h *Herd) spawn(now time.Time, players []Player, g Ground, night bool) (Mob, bool) {
	if !h.spawns || now.Sub(h.spawnAt) < spawnEvery || len(players) == 0 {
		return Mob{}, false
	}
	h.spawnAt = now
	k, around, ok := h.wanted(players, night)
	if !ok {
		return Mob{}, false
	}
	pos, ok := h.spot(k, around, players, g)
	if !ok {
		return Mob{}, false
	}
	return h.Add(k, pos, now), true
}

// wanted is which kind should spawn, and around whom, while there is room for more.
func (h *Herd) wanted(players []Player, night bool) (Kind, motion.Vec, bool) {
	if !night {
		p := players[h.rng.IntN(len(players))]
		return Sheep, p.Pos, h.count(Sheep) < maxSheep
	}
	var prey []motion.Vec
	for _, p := range players {
		if p.Prey {
			prey = append(prey, p.Pos)
		}
	}
	if len(prey) == 0 || h.count(Slime) >= min(monstersPerPrey*len(prey), maxMonsters) {
		return 0, motion.Vec{}, false
	}
	return Slime, prey[h.rng.IntN(len(prey))], true
}

func (h *Herd) count(k Kind) int {
	n := 0
	for _, m := range h.mobs {
		if m.Kind == k {
			n++
		}
	}
	return n
}

// spot picks a random surface a fair way from `around` and checks a k fits there.
func (h *Herd) spot(k Kind, around motion.Vec, players []Player, g Ground) (motion.Vec, bool) {
	angle, dist := h.rng.Float64()*2*math.Pi, near+h.rng.Float64()*(spawnFar-near)
	x, z := cell(around.X+math.Sin(angle)*dist), cell(around.Z+math.Cos(angle)*dist)
	pos := drops.Rest(x, world.MaxY-1, z, g.Boxes)
	for _, p := range players {
		if math.Hypot(p.Pos.X-pos.X, p.Pos.Z-pos.Z) < near {
			return pos, false
		}
	}
	m := Mob{Kind: k, Pos: pos}
	if _, ok := m.footing(x, z, g); !ok {
		return pos, false
	}
	under, _ := g.Block(x, cell(pos.Y)-1, z)
	return pos, k != Sheep || under == world.Grass
}
