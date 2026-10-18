package world

// Size is a chunk's edge length in blocks; Volume is blocks per chunk.
const (
	Size   = 16
	Volume = Size * Size * Size
)

// Vertical limits in blocks (MaxY exclusive).
const (
	MinY = 0
	MaxY = 128
)

// ChunkPos is a chunk coordinate (block coordinate divided by Size).
type ChunkPos struct{ X, Y, Z int32 }

// ChunkOf returns the chunk holding block x,y,z and the block's local position inside it.
func ChunkOf(x, y, z int32) (pos ChunkPos, lx, ly, lz int) {
	return ChunkPos{x >> 4, y >> 4, z >> 4}, int(x & 15), int(y & 15), int(z & 15)
}

// Origin returns the world coordinate of the chunk's minimum corner.
func (p ChunkPos) Origin() (x, y, z int32) { return p.X * Size, p.Y * Size, p.Z * Size }

// Chunk is a 16³ block volume. A uniform chunk stores only its fill block.
type Chunk struct {
	blocks *[Volume]Block
	fill   Block
}

// NewChunk returns a chunk filled with b.
func NewChunk(b Block) *Chunk { return &Chunk{fill: b} }

func index(x, y, z int) int { return (y*Size+z)*Size + x }

// Get returns the block at local coordinates (each 0–15).
func (c *Chunk) Get(x, y, z int) Block {
	if c.blocks == nil {
		return c.fill
	}
	return c.blocks[index(x, y, z)]
}

// Set changes the block at local coordinates, allocating storage on the first differing block.
func (c *Chunk) Set(x, y, z int, b Block) {
	if c.blocks == nil {
		if b == c.fill {
			return
		}
		c.blocks = new([Volume]Block)
		for i := range c.blocks {
			c.blocks[i] = c.fill
		}
	}
	c.blocks[index(x, y, z)] = b
}

// Uniform reports whether the chunk is stored as a single fill block.
func (c *Chunk) Uniform() bool { return c.blocks == nil }

// compact frees storage when every block is the same.
func (c *Chunk) compact() {
	if c.blocks == nil {
		return
	}
	for _, b := range c.blocks {
		if b != c.blocks[0] {
			return
		}
	}
	c.fill, c.blocks = c.blocks[0], nil
}
