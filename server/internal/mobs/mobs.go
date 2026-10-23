// Package mobs runs the sheep that graze near players, the slimes that come out at night and
// the bosses woken at dungeon altars.
package mobs

import (
	"math"
	"math/rand/v2"
	"slices"
	"time"

	"scabbage/internal/motion"
	"scabbage/internal/weather"
	"scabbage/internal/world"
)

type Kind uint8

const (
	Sheep Kind = iota + 1
	Slime
	KingSlime
	Golem
)

const (
	Every    = 100 * time.Millisecond // how often the herd steps
	Regrow   = 3 * time.Minute        // a sheared sheep's wool
	reach    = 5.5 * meter            // from the hitter's eye to the mob's middle, with lag
	firstID  = 1 << 31                // above player entity ids, so the two never meet
	maxDelta = 0.5                    // seconds simulated in one step at most
)

// Mob is one animal or monster. Only the herd changes it.
type Mob struct {
	ID      uint32
	Kind    Kind
	Pos     motion.Vec // feet center
	Yaw     float64    // faces (sin Yaw, cos Yaw) in x,z
	Sheared bool
	Hurt    bool // set on the copy a hit returns

	health  int
	speed   float64
	until   time.Time // the current walk, rest or flight ends
	fleeing bool
	biteAt  time.Time // earliest next bite
	regrow  time.Time
}

func (m *Mob) height() float64 { return kinds[m.Kind].height }

func (m Mob) Health() int    { return m.health }
func (m Mob) MaxHealth() int { return kinds[m.Kind].health }
func (m Mob) Boss() bool     { return kinds[m.Kind].boss }

// Player is someone mobs can see; slimes only go after Prey (survival players).
type Player struct {
	Pos  motion.Vec
	Prey bool
}

// Ground is the world as mobs walk on it.
type Ground interface {
	Block(x, y, z int32) (world.Block, error)
	Boxes(x, y, z int32) []world.Box
}

// Herd is every mob in the world. It is not safe for concurrent use.
type Herd struct {
	spawns  bool
	rng     *rand.Rand
	mobs    []*Mob
	last    uint32
	stepAt  time.Time
	spawnAt time.Time
}

// New returns an empty herd; without spawns, only Add brings mobs in.
func New(seed uint64, spawns bool) *Herd {
	return &Herd{spawns: spawns, rng: rand.New(rand.NewPCG(seed, seed^0x5eed)), last: firstID}
}

// Add puts a mob at pos, resting for a moment, and returns it.
func (h *Herd) Add(k Kind, pos motion.Vec, now time.Time) Mob {
	h.last++
	m := &Mob{ID: h.last, Kind: k, Pos: pos, health: kinds[k].health, until: now.Add(h.between(2*time.Second, 6*time.Second))}
	h.mobs = append(h.mobs, m)
	return *m
}

func (h *Herd) All() []Mob {
	out := make([]Mob, len(h.mobs))
	for i, m := range h.mobs {
		out[i] = *m
	}
	return out
}

func (h *Herd) Len() int { return len(h.mobs) }

// Clear removes every mob, e.g. once no one is left to see them.
func (h *Herd) Clear() { h.mobs = nil }

// Changes is what one step did.
type Changes struct {
	Moved  []Mob
	Gone   []uint32
	Bitten []Bite
}

// Step moves every mob on by the time since the last step, spawning and removing some;
// night says whether it is night, which admins can change apart from the wall clock.
func (h *Herd) Step(now time.Time, night bool, players []Player, g Ground) Changes {
	dt := min(max(now.Sub(h.stepAt).Seconds(), 0), maxDelta)
	h.stepAt = now
	var c Changes
	h.mobs = slices.DeleteFunc(h.mobs, func(m *Mob) bool {
		leaves := h.leaves(m, players, night)
		if leaves {
			c.Gone = append(c.Gone, m.ID)
		}
		return leaves
	})
	for _, m := range h.mobs {
		if h.live(m, now, dt, players, g, &c) {
			c.Moved = append(c.Moved, *m)
		}
	}
	if m, ok := h.spawn(now, players, g, night); ok {
		c.Moved = append(c.Moved, m)
	}
	return c
}

// live runs one mob for dt seconds and reports whether others need to see the change.
func (h *Herd) live(m *Mob, now time.Time, dt float64, players []Player, g Ground, c *Changes) bool {
	pos, yaw, sheared := m.Pos, m.Yaw, m.Sheared
	h.think(m, now, players)
	m.walk(dt, now, g)
	if b, ok := m.bite(now, players); ok {
		c.Bitten = append(c.Bitten, b)
	}
	if m.Sheared && !now.Before(m.regrow) {
		m.Sheared = false
	}
	return m.Pos != pos || m.Yaw != yaw || m.Sheared != sheared
}

// Hit is what hitting a mob did: what it dropped, whether it popped and any mobs it let out.
type Hit struct {
	Mob     Mob
	Drop    world.Block // Air for nothing
	Gone    bool
	Spawned []Mob
}

// Hit lets someone whose eye is at `from` hit mob id; false when it isn't there or is out of reach.
func (h *Herd) Hit(id uint32, from motion.Vec, now time.Time) (Hit, bool) {
	i := slices.IndexFunc(h.mobs, func(m *Mob) bool { return m.ID == id })
	if i < 0 {
		return Hit{}, false
	}
	m := h.mobs[i]
	middle := motion.Vec{X: m.Pos.X, Y: m.Pos.Y + m.height()/2, Z: m.Pos.Z}
	if motion.Distance(from, middle) > reach+kinds[m.Kind].halfWidth {
		return Hit{}, false
	}
	var out Hit
	switch m.Kind {
	case Sheep:
		m.flee(from, now, 3*meter, 2*time.Second)
		if !m.Sheared {
			m.Sheared, m.regrow, out.Drop = true, now.Add(Regrow), world.WhiteWool
		}
	default:
		if !kinds[m.Kind].steady {
			m.flee(from, now, 5*meter, 300*time.Millisecond)
		}
		if m.health--; m.health <= 0 {
			h.mobs = slices.Delete(h.mobs, i, i+1)
			out.Gone = true
		} else if m.Kind == KingSlime && m.health%shedEvery == 0 {
			out.Spawned = append(out.Spawned, h.Add(Slime, m.Pos, now))
		}
	}
	out.Mob = *m
	out.Mob.Hurt = true
	return out, true
}

// Night mirrors client/src/render/sky.ts: one day lasts DayLength, starting at sunrise.
func Night(now time.Time) bool {
	angle := float64(now.UnixMilli()%DayLength.Milliseconds()) / float64(DayLength.Milliseconds()) * 2 * math.Pi
	return math.Sin(angle) < -0.1
}

// DayLength is one whole day and night.
const DayLength = weather.DayLength
