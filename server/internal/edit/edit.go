// Package edit decides which block breaks and placements players may make.
package edit

import (
	"scabbage/internal/motion"
	"scabbage/internal/world"
)

// Reach is the farthest a block's center may be from the eye: the client reaches 5 m to a face,
// plus up to a block's half-diagonal and the eye lag between physics steps.
const Reach = 6.5 * world.Metre

// Request asks to set the block at X,Y,Z; Air breaks it.
type Request struct {
	X, Y, Z int32
	Block   world.Block
}

// Allowed reports whether a block that is now `current` may change as r asks; reach is checked
// separately. Solid placements must not overlap any body in `bodies`.
func Allowed(r Request, current world.Block, bodies []motion.Vec) bool {
	if r.Block == world.Air {
		return world.Info(current).Targetable() && current != world.Bedrock
	}
	if !r.Block.Valid() || r.Block == world.Bedrock || !world.Placeable(r.Block) || world.Info(current).Targetable() {
		return false
	}
	return !world.Info(r.Block).Solid || !Occupied(r.X, r.Y, r.Z, bodies)
}

// InReach reports whether a player with feet at `feet` can reach the block at x, y, z.
func InReach(feet motion.Vec, x, y, z int32) bool {
	dx := float64(x) + 0.5 - feet.X
	dy := float64(y) + 0.5 - (feet.Y + motion.EyeHeight)
	dz := float64(z) + 0.5 - feet.Z
	return dx*dx+dy*dy+dz*dz <= Reach*Reach
}

// Occupied reports whether any of bodies stands partly inside the block at x, y, z.
func Occupied(x, y, z int32, bodies []motion.Vec) bool {
	only := func(bx, by, bz int32) []world.Box {
		if bx == x && by == y && bz == z {
			return []world.Box{world.FullBox}
		}
		return nil
	}
	for _, b := range bodies {
		if motion.Blocked(b, only) {
			return true
		}
	}
	return false
}

// Edit rate limits, per brush: a held mouse button repeats about 6 times a second.
const (
	PerSecond = 30
	Burst     = 60
)
