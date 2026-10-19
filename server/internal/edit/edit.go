// Package edit decides which block breaks and placements players may make.
package edit

import (
	"math"
	"time"

	"scabbage/internal/motion"
	"scabbage/internal/world"
)

// Reach is the farthest a block's center may be from the eye: the client reaches 5 to a face,
// plus up to a block's half-diagonal and the eye lag between physics steps.
const Reach = 6.5

// eyeHeight must match client/src/player/physics.ts.
const eyeHeight = 1.62

// Request asks to set the block at X,Y,Z; Air breaks it.
type Request struct {
	X, Y, Z int32
	Block   world.Block
}

// Allowed reports whether a player with feet at `feet` may change a block that is now `current`.
// Solid placements must not overlap any body in `bodies`.
func Allowed(feet motion.Vec, r Request, current world.Block, bodies []motion.Vec) bool {
	if !inReach(feet, r) {
		return false
	}
	if r.Block == world.Air {
		return world.Info(current).Solid && current != world.Bedrock
	}
	if !r.Block.Valid() || r.Block == world.Bedrock || world.Info(current).Solid {
		return false
	}
	return !world.Info(r.Block).Solid || !overlapsAny(r, bodies)
}

func inReach(feet motion.Vec, r Request) bool {
	dx := float64(r.X) + 0.5 - feet.X
	dy := float64(r.Y) + 0.5 - (feet.Y + eyeHeight)
	dz := float64(r.Z) + 0.5 - feet.Z
	return dx*dx+dy*dy+dz*dz <= Reach*Reach
}

func overlapsAny(r Request, bodies []motion.Vec) bool {
	only := func(x, y, z int32) bool { return x == r.X && y == r.Y && z == r.Z }
	for _, b := range bodies {
		if motion.Blocked(b, only) {
			return true
		}
	}
	return false
}

// Edit rate limits: a held mouse button repeats about 4 times a second.
const (
	perSecond = 10
	burst     = 20
)

// Rate is a token bucket of edits for one player.
type Rate struct {
	tokens float64
	last   time.Time
}

// NewRate starts with a full bucket.
func NewRate(now time.Time) *Rate { return &Rate{tokens: burst, last: now} }

// Take spends one edit if any are left.
func (r *Rate) Take(now time.Time) bool {
	dt := max(now.Sub(r.last).Seconds(), 0)
	r.tokens, r.last = math.Min(r.tokens+dt*perSecond, burst), now
	if r.tokens < 1 {
		return false
	}
	r.tokens--
	return true
}
