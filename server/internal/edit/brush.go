package edit

import (
	"math"
	"time"

	"scabbage/internal/motion"
	"scabbage/internal/world"
)

// MaxBrush is the widest brush, a cube as big as one block was before blocks shrank.
const MaxBrush = world.Metre

// Brush sets every block in a cube of Size blocks a side whose lowest corner is X,Y,Z.
type Brush struct {
	X, Y, Z, Size int32
	Block         world.Block
}

func (b Brush) size() int32 { return max(1, min(b.Size, MaxBrush)) }

// Cells lists one request per block in the brush.
func (b Brush) Cells() []Request {
	n := b.size()
	out := make([]Request, 0, n*n*n)
	for x := b.X; x < b.X+n; x++ {
		for y := b.Y; y < b.Y+n; y++ {
			for z := b.Z; z < b.Z+n; z++ {
				out = append(out, Request{X: x, Y: y, Z: z, Block: b.Block})
			}
		}
	}
	return out
}

// InReach reports whether a player with feet at `feet` can reach the middle of the brush, allowing
// for the client aiming at its nearest block instead.
func (b Brush) InReach(feet motion.Vec) bool {
	half := float64(b.size()) / 2
	dx := float64(b.X) + half - feet.X
	dy := float64(b.Y) + half - (feet.Y + motion.EyeHeight)
	dz := float64(b.Z) + half - feet.Z
	r := Reach + (half-0.5)*math.Sqrt(3)
	return dx*dx+dy*dy+dz*dz <= r*r
}

// DigTime is how long survival players dig a brush of `size` holding blocks: the slowest one's
// time for each block of the brush's width.
func DigTime(blocks []world.Block, size int32) time.Duration {
	slowest := uint16(0)
	for _, b := range blocks {
		slowest = max(slowest, world.Info(b).MineMs)
	}
	return time.Duration(slowest) * time.Duration(max(1, min(size, MaxBrush))) * time.Millisecond
}
