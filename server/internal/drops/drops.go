// Package drops tracks items lying in the world until someone picks them up.
package drops

import (
	"math"
	"slices"
	"time"

	"scabbage/internal/motion"
	"scabbage/internal/world"
)

const (
	Lifetime    = 5 * time.Minute
	PickupDelay = 500 * time.Millisecond // long enough to see the item pop out
	Max         = 256                    // the oldest drop vanishes beyond this
	reach       = 1.75 * world.Metre     // from the body's middle to the item
)

type Drop struct {
	ID    uint64
	Block world.Block
	Count int
	Pos   motion.Vec
	Born  time.Time
}

// Field is every drop in the world. It is not safe for concurrent use.
type Field struct {
	last  uint64
	drops []Drop // oldest first
}

// Rest is where an item let go at block x,y,z comes to lie: on top of whatever is below.
func Rest(x, y, z int32, boxes motion.BoxesFunc) motion.Vec {
	top := 0.0
	for c := y; c >= 0; c-- {
		if b := boxes(x, c, z); len(b) > 0 {
			top = float64(c) + highest(b)
			break
		}
	}
	return motion.Vec{X: float64(x) + 0.5, Y: top, Z: float64(z) + 0.5}
}

func highest(boxes []world.Box) float64 {
	h := 0.0
	for _, b := range boxes {
		h = max(h, b.Max[1])
	}
	return h
}

// Spawn adds a drop of one b, returning it and the ids of any pushed out to make room.
func (f *Field) Spawn(b world.Block, pos motion.Vec, now time.Time) (Drop, []uint64) {
	return f.SpawnMany(b, 1, pos, now)
}

// SpawnMany adds a pile of n b as one drop, returning it and the ids of any pushed out to make room.
func (f *Field) SpawnMany(b world.Block, n int, pos motion.Vec, now time.Time) (Drop, []uint64) {
	var gone []uint64
	if len(f.drops) >= Max {
		gone = append(gone, f.drops[0].ID)
		f.drops = slices.Delete(f.drops, 0, 1)
	}
	f.last++
	d := Drop{ID: f.last, Block: b, Count: n, Pos: pos, Born: now}
	f.drops = append(f.drops, d)
	return d, gone
}

// Collect offers nearby, settled drops to take, which says how many of each it took; drops taken
// whole are removed.
func (f *Field) Collect(feet motion.Vec, now time.Time, take func(Drop) int) []uint64 {
	middle := motion.Vec{X: feet.X, Y: feet.Y + motion.Height/2, Z: feet.Z}
	return f.remove(func(d *Drop) bool {
		if now.Sub(d.Born) < PickupDelay || !near(middle, d.Pos) {
			return false
		}
		d.Count -= take(*d)
		return d.Count <= 0
	})
}

// Expire removes drops that have lain around for their whole Lifetime.
func (f *Field) Expire(now time.Time) []uint64 {
	return f.remove(func(d *Drop) bool { return now.Sub(d.Born) >= Lifetime })
}

func (f *Field) remove(gone func(*Drop) bool) []uint64 {
	var ids []uint64
	kept := f.drops[:0]
	for i := range f.drops {
		if d := &f.drops[i]; gone(d) {
			ids = append(ids, d.ID)
		} else {
			kept = append(kept, *d)
		}
	}
	f.drops = kept
	return ids
}

// Resettle lets drops in the column x,z fall after the ground under them changed.
func (f *Field) Resettle(x, z int32, boxes motion.BoxesFunc) []Drop {
	var moved []Drop
	for i := range f.drops {
		d := &f.drops[i]
		if cell(d.Pos.X) != x || cell(d.Pos.Z) != z {
			continue
		}
		if to := Rest(x, cell(d.Pos.Y), z, boxes); to != d.Pos {
			d.Pos = to
			moved = append(moved, *d)
		}
	}
	return moved
}

func (f *Field) All() []Drop { return slices.Clone(f.drops) }

func (f *Field) Len() int { return len(f.drops) }

func near(a, b motion.Vec) bool {
	dx, dy, dz := a.X-b.X, a.Y-b.Y, a.Z-b.Z
	return dx*dx+dy*dy+dz*dz <= reach*reach
}

func cell(v float64) int32 { return int32(math.Floor(v)) }
