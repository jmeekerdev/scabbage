package world

import (
	"errors"
	"fmt"
)

var (
	// ErrOutside means a position is beyond the world border or height limits.
	ErrOutside = errors.New("outside the world")
	// ErrUnknownBlock means a block ID isn't registered.
	ErrUnknownBlock = errors.New("unknown block")
)

// World caches loaded chunks and tracks which need saving. It is not safe for concurrent use;
// the simulation loop owns it.
type World struct {
	meta   Meta
	store  *Store
	gen    Generator
	chunks map[ChunkPos]*Chunk
	dirty  map[ChunkPos]struct{}
}

// Open loads the named world, creating it from d the first time.
func Open(s *Store, name string, d Defaults) (*World, error) {
	meta, err := s.world(name, d)
	if err != nil {
		return nil, err
	}
	gen, err := NewGenerator(meta.Generator, meta.Seed)
	if err != nil {
		return nil, err
	}
	return &World{meta: meta, store: s, gen: gen,
		chunks: map[ChunkPos]*Chunk{}, dirty: map[ChunkPos]struct{}{}}, nil
}

// Meta returns the world's stored identity.
func (w *World) Meta() Meta { return w.meta }

// Inside reports whether chunk p overlaps the playable area.
func (w *World) Inside(p ChunkPos) bool {
	x, y, z := p.Origin()
	if y < MinY || y >= MaxY {
		return false
	}
	r := w.meta.Radius
	return r == Unbounded || (x < r && x+Size > -r && z < r && z+Size > -r)
}

// Chunk returns a loaded chunk, reading it from storage or generating it as needed.
func (w *World) Chunk(p ChunkPos) (*Chunk, error) {
	if c, ok := w.chunks[p]; ok {
		return c, nil
	}
	if !w.Inside(p) {
		return nil, ErrOutside
	}
	data, err := w.store.loadChunk(w.meta.ID, p)
	if err != nil {
		return nil, err
	}
	var c *Chunk
	if data == nil {
		c = w.gen.Generate(p)
		w.dirty[p] = struct{}{} // keep generated land even if the generator changes later
	} else if c, err = Decode(data); err != nil {
		return nil, fmt.Errorf("chunk %v: %w", p, err)
	}
	w.chunks[p] = c
	return c, nil
}

// Block returns the block at world coordinates.
func (w *World) Block(x, y, z int32) (Block, error) {
	p, lx, ly, lz := ChunkOf(x, y, z)
	c, err := w.Chunk(p)
	if err != nil {
		return Air, err
	}
	return c.Get(lx, ly, lz), nil
}

// SetBlock changes a block inside the border and marks its chunk for saving.
func (w *World) SetBlock(x, y, z int32, b Block) error {
	if !b.Valid() {
		return ErrUnknownBlock
	}
	if !w.blockInside(x, z) {
		return ErrOutside
	}
	p, lx, ly, lz := ChunkOf(x, y, z)
	c, err := w.Chunk(p)
	if err != nil {
		return err
	}
	c.Set(lx, ly, lz, b)
	w.dirty[p] = struct{}{}
	return nil
}

func (w *World) blockInside(x, z int32) bool {
	r := w.meta.Radius
	return r == Unbounded || (x >= -r && x < r && z >= -r && z < r)
}

// Flush saves every dirty chunk in one transaction.
func (w *World) Flush() error {
	if len(w.dirty) == 0 {
		return nil
	}
	encoded := make(map[ChunkPos][]byte, len(w.dirty))
	for p := range w.dirty {
		encoded[p] = Encode(w.chunks[p])
	}
	if err := w.store.saveChunks(w.meta.ID, encoded); err != nil {
		return err
	}
	clear(w.dirty)
	return nil
}

// Unload saves dirty chunks and drops every chunk from memory.
func (w *World) Unload() error {
	if err := w.Flush(); err != nil {
		return err
	}
	w.chunks = map[ChunkPos]*Chunk{}
	return nil
}

// Loaded returns the number of chunks in memory.
func (w *World) Loaded() int { return len(w.chunks) }

// Dirty returns the number of chunks waiting to be saved.
func (w *World) Dirty() int { return len(w.dirty) }
