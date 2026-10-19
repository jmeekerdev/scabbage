// Package motion checks that players' self-reported moves are physically plausible.
package motion

import (
	"math"
	"time"

	"scabbage/internal/world"
)

// Vec is a position in blocks.
type Vec struct{ X, Y, Z float64 }

// Player body size; must match client/src/player/physics.ts.
const (
	HalfWidth = 0.3
	Height    = 1.8
)

// slack shrinks the body when checking overlap, absorbing float32 rounding on the wire.
const slack = 0.01

// pathStep is the longest gap between positions checked along a move.
const pathStep = 0.25

// StepHeight is how far the client lifts a walking body onto slabs and stairs.
const StepHeight = 0.5

// Limits caps speeds in blocks per second; Burst is how much unused time can be saved up.
type Limits struct {
	Horizontal, Up, Down float64
	Burst                time.Duration
}

// DefaultLimits leave headroom over the client's fly speed, jump and terminal fall speed.
var DefaultLimits = Limits{Horizontal: 14, Up: 12, Down: 60, Burst: time.Second}

// BoxesFunc returns what players collide with in the block at x,y,z.
type BoxesFunc func(x, y, z int32) []world.Box

// Tracker holds a player's last accepted position and how far they may still move.
type Tracker struct {
	limits      Limits
	pos         Vec
	last        time.Time
	h, up, down float64 // remaining distance budgets
}

// NewTracker starts tracking at pos with full budgets.
func NewTracker(l Limits, pos Vec, now time.Time) *Tracker {
	t := &Tracker{limits: l}
	t.Reset(pos, now)
	return t
}

// Pos returns the last accepted position.
func (t *Tracker) Pos() Vec { return t.pos }

// Reset places the player without checks, e.g. after a teleport.
func (t *Tracker) Reset(pos Vec, now time.Time) {
	t.pos, t.last = pos, now
	b := t.limits.Burst.Seconds()
	t.h, t.up, t.down = t.limits.Horizontal*b, t.limits.Up*b, t.limits.Down*b
}

// Move accepts the move to `to` if it is within budget and never passes through solid blocks.
func (t *Tracker) Move(to Vec, now time.Time, solid BoxesFunc) bool {
	if !finite(to) {
		return false
	}
	h, up, down := t.refilled(now)
	d := Vec{to.X - t.pos.X, to.Y - t.pos.Y, to.Z - t.pos.Z}
	h -= math.Hypot(d.X, d.Z)
	up -= math.Max(d.Y, 0)
	down -= math.Max(-d.Y, 0)
	if h < 0 || up < 0 || down < 0 || pathBlocked(t.pos, to, solid) {
		return false
	}
	t.pos, t.last, t.h, t.up, t.down = to, now, h, up, down
	return true
}

func (t *Tracker) refilled(now time.Time) (h, up, down float64) {
	dt := now.Sub(t.last).Seconds()
	if dt < 0 {
		dt = 0
	}
	l, b := t.limits, t.limits.Burst.Seconds()
	refill := func(have, rate float64) float64 { return math.Min(have+rate*dt, rate*b) }
	return refill(t.h, l.Horizontal), refill(t.up, l.Up), refill(t.down, l.Down)
}

// pathBlocked checks the body at evenly spaced points from just past `from` up to `to`.
// Points before `to` may clip a step's corner, since the client lifts bodies onto steps.
func pathBlocked(from, to Vec, solid BoxesFunc) bool {
	dist := math.Sqrt((to.X-from.X)*(to.X-from.X) + (to.Y-from.Y)*(to.Y-from.Y) + (to.Z-from.Z)*(to.Z-from.Z))
	steps := int(math.Ceil(dist / pathStep))
	for i := 1; i <= steps; i++ {
		f := float64(i) / float64(steps)
		p := Vec{from.X + (to.X-from.X)*f, from.Y + (to.Y-from.Y)*f, from.Z + (to.Z-from.Z)*f}
		if Blocked(p, solid) && (i == steps || Blocked(Vec{p.X, p.Y + StepHeight, p.Z}, solid)) {
			return true
		}
	}
	return steps == 0 && Blocked(to, solid)
}

// Blocked reports whether a body with its feet centered at p overlaps any collision box.
func Blocked(p Vec, solid BoxesFunc) bool {
	lo := [3]float64{p.X - HalfWidth + slack, p.Y + slack, p.Z - HalfWidth + slack}
	hi := [3]float64{p.X + HalfWidth - slack, p.Y + Height - slack, p.Z + HalfWidth - slack}
	for y := cell(lo[1]); y <= cell(hi[1]); y++ {
		for z := cell(lo[2]); z <= cell(hi[2]); z++ {
			for x := cell(lo[0]); x <= cell(hi[0]); x++ {
				if overlapsAny(lo, hi, [3]int32{x, y, z}, solid(x, y, z)) {
					return true
				}
			}
		}
	}
	return false
}

// overlapsAny reports whether the box lo..hi overlaps any of a cell's boxes.
func overlapsAny(lo, hi [3]float64, cell [3]int32, boxes []world.Box) bool {
	for _, b := range boxes {
		inside := true
		for a := range 3 {
			c := float64(cell[a])
			inside = inside && lo[a] < c+b.Max[a] && hi[a] > c+b.Min[a]
		}
		if inside {
			return true
		}
	}
	return false
}

func cell(v float64) int32 { return int32(math.Floor(v)) }

func finite(v Vec) bool {
	for _, c := range []float64{v.X, v.Y, v.Z} {
		if math.IsNaN(c) || math.IsInf(c, 0) {
			return false
		}
	}
	return true
}
