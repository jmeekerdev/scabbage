package world

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"sync"
)

var (
	// ErrOutside means a position is beyond the world border or height limits.
	ErrOutside = errors.New("outside the world")
	// ErrUnknownBlock means a block ID isn't registered.
	ErrUnknownBlock = errors.New("unknown block")
)

// World caches loaded chunks and tracks which need saving. It is safe for concurrent use.
type World struct {
	mu      sync.Mutex
	meta    Meta
	store   *Store
	gen     Generator
	chunks  map[ChunkPos]*Chunk
	dirty   map[ChunkPos]struct{}
	ticking map[Pos]struct{} // loaded blocks the server steps, like timers
}

// Pos is a block position.
type Pos struct{ X, Y, Z int32 }

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
		chunks: map[ChunkPos]*Chunk{}, dirty: map[ChunkPos]struct{}{}, ticking: map[Pos]struct{}{}}, nil
}

// Wipe forgets every change and starts the world over from seed, as if it were new.
func (w *World) Wipe(seed int64) error {
	gen, err := NewGenerator(w.meta.Generator, seed)
	if err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.store.wipe(w.meta.ID, seed); err != nil {
		return err
	}
	w.meta.Seed, w.gen = seed, gen
	w.chunks, w.dirty, w.ticking = map[ChunkPos]*Chunk{}, map[ChunkPos]struct{}{}, map[Pos]struct{}{}
	return nil
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

// EncodedChunk returns chunk p in wire format; chunks outside the world are air.
func (w *World) EncodedChunk(p ChunkPos) ([]byte, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.Inside(p) {
		return Encode(NewChunk(Air)), nil
	}
	c, err := w.chunk(p)
	if err != nil {
		return nil, err
	}
	return Encode(c), nil
}

// Spawn returns the center of the column at the world's middle, just above its top block.
func (w *World) Spawn() (x, y, z float64) {
	top, err := w.Top(0, 0)
	if err != nil {
		top = MaxY
	}
	return 0.5, float64(top), 0.5
}

// Top is the y just above column x,z's highest block, where someone could stand: also above
// the solid blocks in the columns on each side, which a body is wide enough to reach into.
func (w *World) Top(x, z int32) (int32, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	top, err := w.columnTop(x, z, func(b Block) bool { return b != Air })
	if err != nil {
		return 0, err
	}
	solid := func(b Block) bool { return Info(b).Solid }
	for dx := int32(-1); dx <= 1; dx++ {
		for dz := int32(-1); dz <= 1; dz++ {
			if t, err := w.columnTop(x+dx, z+dz, solid); err == nil {
				top = max(top, t)
			}
		}
	}
	return top, nil
}

// columnTop is the y just above column x,z's highest block that counts.
func (w *World) columnTop(x, z int32, counts func(Block) bool) (int32, error) {
	top := int32(MaxY - 1)
	for ; top > MinY; top-- {
		b, err := w.block(x, top, z)
		if err != nil {
			return 0, err
		}
		if counts(b) {
			break
		}
	}
	return top + 1, nil
}

// FarTile sketches chunk column tx,tz as generated, without players' changes, for drawing it
// far away; ok is false outside the border.
func (w *World) FarTile(tx, tz int32) (tile FarTile, ok bool) {
	if !w.Inside(ChunkPos{tx, 0, tz}) {
		return FarTile{}, false
	}
	w.mu.Lock()
	gen := w.gen
	w.mu.Unlock()
	return gen.FarTile(tx, tz), true
}

// Chunk returns a loaded chunk, reading it from storage or generating it as needed.
// The chunk must not be used once other goroutines may change the world.
func (w *World) Chunk(p ChunkPos) (*Chunk, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.chunk(p)
}

func (w *World) chunk(p ChunkPos) (*Chunk, error) {
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
	w.indexTicking(p, c)
	return c, nil
}

// indexTicking remembers the ticking blocks in a freshly loaded chunk.
func (w *World) indexTicking(p ChunkPos, c *Chunk) {
	if c.Uniform() {
		return
	}
	ox, oy, oz := p.Origin()
	for i, b := range c.blocks {
		if Info(b).Ticks {
			w.ticking[Pos{ox + int32(i%Size), oy + int32(i/(Size*Size)), oz + int32(i/Size%Size)}] = struct{}{}
		}
	}
}

// Ticking lists the loaded blocks the server steps, in a stable order.
func (w *World) Ticking() []Pos {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]Pos, 0, len(w.ticking))
	for p := range w.ticking {
		out = append(out, p)
	}
	slices.SortFunc(out, func(a, b Pos) int {
		return cmp.Or(cmp.Compare(a.X, b.X), cmp.Compare(a.Y, b.Y), cmp.Compare(a.Z, b.Z))
	})
	return out
}

// Block returns the block at world coordinates.
func (w *World) Block(x, y, z int32) (Block, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.block(x, y, z)
}

func (w *World) block(x, y, z int32) (Block, error) {
	p, lx, ly, lz := ChunkOf(x, y, z)
	c, err := w.chunk(p)
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
	w.mu.Lock()
	defer w.mu.Unlock()
	p, lx, ly, lz := ChunkOf(x, y, z)
	c, err := w.chunk(p)
	if err != nil {
		return err
	}
	c.Set(lx, ly, lz, b)
	w.dirty[p] = struct{}{}
	if Info(b).Ticks {
		w.ticking[Pos{x, y, z}] = struct{}{}
	} else {
		delete(w.ticking, Pos{x, y, z})
	}
	return nil
}

// Boxes is what players collide with at x,y,z; the border and the floor are walls, the sky open.
func (w *World) Boxes(x, y, z int32) []Box {
	if y >= MaxY {
		return nil
	}
	if y < MinY || !w.blockInside(x, z) {
		return shapeBoxes[Cube]
	}
	b, err := w.Block(x, y, z)
	if err != nil {
		return shapeBoxes[Cube]
	}
	return Info(b).Boxes()
}

func (w *World) blockInside(x, z int32) bool {
	r := w.meta.Radius
	return r == Unbounded || (x >= -r && x < r && z >= -r && z < r)
}

// Flush saves every dirty chunk in one transaction.
func (w *World) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.flush()
}

func (w *World) flush() error {
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
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.flush(); err != nil {
		return err
	}
	w.chunks = map[ChunkPos]*Chunk{}
	clear(w.ticking)
	return nil
}

// Evict saves dirty chunks, then drops the loaded chunks keep rejects, returning how many.
func (w *World) Evict(keep func(ChunkPos) bool) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.flush(); err != nil {
		return 0, err
	}
	dropped := 0
	for p := range w.chunks {
		if !keep(p) {
			delete(w.chunks, p)
			dropped++
		}
	}
	if dropped > 0 {
		w.forgetTicking()
	}
	return dropped, nil
}

// forgetTicking drops ticking blocks whose chunks are no longer loaded.
func (w *World) forgetTicking() {
	for p := range w.ticking {
		if c, _, _, _ := ChunkOf(p.X, p.Y, p.Z); w.chunks[c] == nil {
			delete(w.ticking, p)
		}
	}
}

// Loaded returns the number of chunks in memory.
func (w *World) Loaded() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.chunks)
}

// Dirty returns the number of chunks waiting to be saved.
func (w *World) Dirty() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.dirty)
}
