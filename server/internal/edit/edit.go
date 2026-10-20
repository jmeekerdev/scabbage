// Package edit decides which block breaks and placements players may make.
package edit

import (
	"scabbage/internal/motion"
	"scabbage/internal/world"
)

// Reach is the farthest a block's center may be from the eye: the client reaches 5 to a face,
// plus up to a block's half-diagonal and the eye lag between physics steps.
const Reach = 6.5

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
		return world.Info(current).Targetable() && current != world.Bedrock
	}
	if !r.Block.Valid() || r.Block == world.Bedrock || world.Info(current).Targetable() {
		return false
	}
	return !world.Info(r.Block).Solid || !overlapsAny(r, bodies)
}

func inReach(feet motion.Vec, r Request) bool {
	dx := float64(r.X) + 0.5 - feet.X
	dy := float64(r.Y) + 0.5 - (feet.Y + motion.EyeHeight)
	dz := float64(r.Z) + 0.5 - feet.Z
	return dx*dx+dy*dy+dz*dz <= Reach*Reach
}

func overlapsAny(r Request, bodies []motion.Vec) bool {
	only := func(x, y, z int32) []world.Box {
		if x == r.X && y == r.Y && z == r.Z {
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

// Edit rate limits: a held mouse button repeats about 4 times a second.
const (
	PerSecond = 10
	Burst     = 20
)
