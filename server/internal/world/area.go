package world

import (
	"cmp"
	"fmt"
	"slices"
)

// Area is every block in the columns from X0, Z0 to X1, Z1, corners included, from MinY up to MaxY.
type Area struct{ X0, Z0, X1, Z1 int32 }

// Change is a block a paste replaced.
type Change struct {
	Pos
	Block, Was Block
}

func (a Area) Width() int32 { return a.X1 - a.X0 + 1 }
func (a Area) Depth() int32 { return a.Z1 - a.Z0 + 1 }
func (a Area) Volume() int  { return int(a.Width()) * int(a.Depth()) * (MaxY - MinY) }

// Index is where block x,y,z sits in the area's list: x first, then z, then y.
func (a Area) Index(x, y, z int32) int {
	return int(x-a.X0) + int(a.Width())*(int(z-a.Z0)+int(a.Depth())*int(y-MinY))
}

// Copy reads every block in the area.
func (w *World) Copy(a Area) ([]Block, error) {
	out := make([]Block, a.Volume())
	w.mu.Lock()
	defer w.mu.Unlock()
	err := w.eachInArea(a, func(c *Chunk, at Pos, lx, ly, lz int) {
		out[a.Index(at.X, at.Y, at.Z)] = c.Get(lx, ly, lz)
	})
	return out, err
}

// Paste sets every block in the area from a list Copy made, returning what changed in a stable order.
func (w *World) Paste(a Area, blocks []Block) ([]Change, error) {
	if len(blocks) != a.Volume() {
		return nil, fmt.Errorf("pasting %d blocks into an area of %d", len(blocks), a.Volume())
	}
	for _, b := range blocks {
		if !b.Valid() {
			return nil, ErrUnknownBlock
		}
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	var changes []Change
	err := w.eachInArea(a, func(c *Chunk, at Pos, lx, ly, lz int) {
		b, was := blocks[a.Index(at.X, at.Y, at.Z)], c.Get(lx, ly, lz)
		if b == was {
			return
		}
		c.Set(lx, ly, lz, b)
		w.dirty[chunkHolding(at)] = struct{}{}
		if Info(b).Ticks {
			w.ticking[at] = struct{}{}
		} else {
			delete(w.ticking, at)
		}
		changes = append(changes, Change{at, b, was})
	})
	slices.SortFunc(changes, func(p, q Change) int {
		return cmp.Or(cmp.Compare(p.X, q.X), cmp.Compare(p.Y, q.Y), cmp.Compare(p.Z, q.Z))
	})
	return changes, err
}

// InBorder reports whether the whole area lies inside the world border.
func (w *World) InBorder(a Area) bool { return w.blockInside(a.X0, a.Z0) && w.blockInside(a.X1, a.Z1) }

// eachInArea calls fn for every block in the area, a chunk at a time; call with mu held.
func (w *World) eachInArea(a Area, fn func(c *Chunk, at Pos, lx, ly, lz int)) error {
	if !w.InBorder(a) {
		return ErrOutside
	}
	lo, _, _, _ := ChunkOf(a.X0, MinY, a.Z0)
	hi, _, _, _ := ChunkOf(a.X1, MaxY-1, a.Z1)
	for cx := lo.X; cx <= hi.X; cx++ {
		for cz := lo.Z; cz <= hi.Z; cz++ {
			for cy := lo.Y; cy <= hi.Y; cy++ {
				if err := w.eachInChunk(a, ChunkPos{cx, cy, cz}, fn); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (w *World) eachInChunk(a Area, p ChunkPos, fn func(c *Chunk, at Pos, lx, ly, lz int)) error {
	c, err := w.chunk(p)
	if err != nil {
		return err
	}
	ox, oy, oz := p.Origin()
	for ly := range Size {
		for lz := range Size {
			for lx := range Size {
				at := Pos{ox + int32(lx), oy + int32(ly), oz + int32(lz)}
				if at.X >= a.X0 && at.X <= a.X1 && at.Z >= a.Z0 && at.Z <= a.Z1 {
					fn(c, at, lx, ly, lz)
				}
			}
		}
	}
	return nil
}

// chunkHolding is the chunk holding block p.
func chunkHolding(p Pos) ChunkPos {
	c, _, _, _ := ChunkOf(p.X, p.Y, p.Z)
	return c
}
