package mobs

import (
	"math"
	"time"

	"scabbage/internal/drops"
	"scabbage/internal/motion"
	"scabbage/internal/world"
)

type kind struct {
	height, halfWidth float64
	health            int
	walk              float64 // metres per second while wandering or chasing
	chases            bool
	damage            int  // a bite's harm
	boss              bool // woken at an altar; never melts
	steady            bool // stands its ground when hit
}

// metre turns sizes and speeds in metres into blocks.
const metre = world.Metre

// Body sizes must match client/src/player/mobs.ts.
var kinds = map[Kind]kind{
	Sheep:     {height: 1.3 * metre, halfWidth: 0.45 * metre, health: 1, walk: 1 * metre},
	Slime:     {height: 0.8 * metre, halfWidth: 0.4 * metre, health: 3, walk: 2.5 * metre, chases: true, damage: 2},
	KingSlime: {height: 2 * metre, halfWidth: 1 * metre, health: 24, walk: 2 * metre, chases: true, damage: 4, boss: true},
	Golem:     {height: 2.6 * metre, halfWidth: 0.6 * metre, health: 40, walk: 1.6 * metre, chases: true, damage: 6, boss: true, steady: true},
}

const (
	chaseRange = 16.0 * metre
	biteGap    = 0.5 * metre // between the mob's side and the player's center, horizontally
	biteEvery  = 1500 * time.Millisecond
	stepUp     = 1.0 * metre // highest ledge a mob hops onto
	maxDrop    = 3.0 * metre // deepest it will jump down
	maxStride  = 0.5         // blocks a mob moves between looks at the ground
)

// think picks what the mob does next: chase prey in sight, or keep wandering and resting.
func (h *Herd) think(m *Mob, now time.Time, players []Player) {
	if m.fleeing && now.Before(m.until) {
		return
	}
	m.fleeing = false
	if p, ok := m.prey(players); ok {
		m.Yaw, m.speed = math.Atan2(p.X-m.Pos.X, p.Z-m.Pos.Z), kinds[m.Kind].walk
		return
	}
	if now.Before(m.until) {
		return
	}
	if m.speed > 0 {
		m.speed, m.until = 0, now.Add(h.between(2*time.Second, 6*time.Second))
		return
	}
	m.Yaw, m.speed = h.rng.Float64()*2*math.Pi, kinds[m.Kind].walk
	m.until = now.Add(h.between(time.Second, 3*time.Second))
}

func (h *Herd) between(lo, hi time.Duration) time.Duration {
	return lo + time.Duration(h.rng.Int64N(int64(hi-lo)))
}

// prey is the nearest survival player a slime can see.
func (m *Mob) prey(players []Player) (motion.Vec, bool) {
	if !kinds[m.Kind].chases {
		return motion.Vec{}, false
	}
	best, found := chaseRange, false
	var at motion.Vec
	for _, p := range players {
		if d := motion.Distance(p.Pos, m.Pos); p.Prey && d < best {
			best, at, found = d, p.Pos, true
		}
	}
	return at, found
}

// flee runs away from `from` at speed for d.
func (m *Mob) flee(from motion.Vec, now time.Time, speed float64, d time.Duration) {
	m.Yaw = math.Atan2(m.Pos.X-from.X, m.Pos.Z-from.Z)
	m.speed, m.until, m.fleeing = speed, now.Add(d), true
}

// walk moves the mob along its heading in strides short enough not to skip over a block.
func (m *Mob) walk(dt float64, now time.Time, g Ground) {
	n := max(1, int(math.Ceil(m.speed*dt/maxStride)))
	for range n {
		m.stride(dt/float64(n), now, g)
	}
}

// stride moves the mob a little, hopping up ledges and stopping at walls, water and cliffs.
func (m *Mob) stride(dt float64, now time.Time, g Ground) {
	to := motion.Vec{X: m.Pos.X + math.Sin(m.Yaw)*m.speed*dt, Y: m.Pos.Y, Z: m.Pos.Z + math.Cos(m.Yaw)*m.speed*dt}
	top, ok := m.footing(to.X, to.Z, g)
	if !ok && m.speed > 0 {
		m.speed, m.until = 0, now // try another way
		return
	}
	to.Y = top
	m.Pos = to
}

// footing is where the mob would stand with its middle at x,z: on the highest ground under its
// body, if it can get there from where it is.
func (m *Mob) footing(x, z float64, g Ground) (float64, bool) {
	w := kinds[m.Kind].halfWidth
	top, ok := 0.0, true
	for cx := cell(x - w); cx <= cell(x+w); cx++ {
		for cz := cell(z - w); cz <= cell(z+w); cz++ {
			y := ground(cx, cz, m.Pos.Y, g)
			b, _ := g.Block(cx, cell(y), cz)
			ok = ok && b != world.Water
			top = max(top, y)
		}
	}
	if !ok || top-m.Pos.Y > stepUp+0.01 || m.Pos.Y-top > maxDrop || top >= world.MaxY {
		return top, false
	}
	for cx := cell(x - w); cx <= cell(x+w); cx++ {
		for cz := cell(z - w); cz <= cell(z+w); cz++ {
			if !clear(cx, cz, top, m.height(), g) {
				return top, false
			}
		}
	}
	return top, true
}

// ground is the top of whatever lies under feet at y in column x,z, looking from one ledge up.
func ground(x, z int32, y float64, g Ground) float64 {
	return drops.Rest(x, cell(y+stepUp-0.01), z, g.Boxes).Y
}

// clear reports whether nothing solid fills column x,z between bottom and bottom+height.
func clear(x, z int32, bottom, height float64, g Ground) bool {
	lo, hi := bottom+0.01, bottom+height-0.01
	for y := cell(lo); y <= cell(hi); y++ {
		for _, b := range g.Boxes(x, y, z) {
			if float64(y)+b.Min[1] < hi && float64(y)+b.Max[1] > lo {
				return false
			}
		}
	}
	return true
}

// Bite is a mob hurting the player at index Player.
type Bite struct{ Player, Damage int }

// bite lets a monster touching prey hurt them, now and then.
func (m *Mob) bite(now time.Time, players []Player) (Bite, bool) {
	k := kinds[m.Kind]
	if !k.chases || now.Before(m.biteAt) {
		return Bite{}, false
	}
	for i, p := range players {
		dy := p.Pos.Y - m.Pos.Y
		if p.Prey && math.Hypot(p.Pos.X-m.Pos.X, p.Pos.Z-m.Pos.Z) <= k.halfWidth+biteGap && dy > -motion.Height && dy < m.height() {
			m.biteAt = now.Add(biteEvery)
			return Bite{i, k.damage}, true
		}
	}
	return Bite{}, false
}

func cell(v float64) int32 { return int32(math.Floor(v)) }
