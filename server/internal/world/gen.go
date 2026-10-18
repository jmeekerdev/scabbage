package world

import (
	"fmt"
	"math"
)

// Generator deterministically fills chunks from a seed. A changed algorithm needs a new version.
type Generator interface {
	Version() uint32
	Generate(p ChunkPos) *Chunk
}

// LatestGenerator is the version new worlds use.
const LatestGenerator = 1

// NewGenerator returns the generator for a stored version.
func NewGenerator(version uint32, seed int64) (Generator, error) {
	switch version {
	case 1:
		return NewGenV1(seed), nil
	}
	return nil, fmt.Errorf("unknown world generator version %d", version)
}

const (
	seaLevel     = 30
	baseHeight   = 36
	heightRange  = 30
	heightScale  = 160.0
	hillRange    = 48
	hillScale    = 480.0
	treePerMille = 12
	treeReach    = 2 // leaf radius: trees this close to a chunk can reach into it
)

// GenV1 makes rolling hills with beaches, lakes and scattered trees.
type GenV1 struct {
	height, hills noise
	trees         uint64
}

// NewGenV1 returns a version 1 generator for seed.
func NewGenV1(seed int64) *GenV1 {
	return &GenV1{height: newNoise(seed), hills: newNoise(seed ^ 0x68696c6c), trees: mix(uint64(seed) ^ 0x7265657473)}
}

// Version implements Generator.
func (g *GenV1) Version() uint32 { return 1 }

// SurfaceY returns the y of the topmost terrain block in column x,z.
func (g *GenV1) SurfaceY(x, z int32) int32 {
	n := g.height.fbm(float64(x)/heightScale, float64(z)/heightScale, 4)
	hill := max(0, g.hills.fbm(float64(x)/hillScale, float64(z)/hillScale, 2))
	return baseHeight + int32(math.Floor(float64(heightRange*n)+float64(hillRange*float64(hill*hill))))
}

// Generate implements Generator.
func (g *GenV1) Generate(p ChunkPos) *Chunk {
	c := NewChunk(Air)
	ox, oy, oz := p.Origin()
	if oy >= MaxY || oy+Size <= MinY {
		return c
	}
	var heights [Size + 2*treeReach][Size + 2*treeReach]int32
	for i := range heights {
		for j := range heights[i] {
			heights[i][j] = g.SurfaceY(ox+int32(i)-treeReach, oz+int32(j)-treeReach)
		}
	}
	for x := range Size {
		for z := range Size {
			h := heights[x+treeReach][z+treeReach]
			for y := range Size {
				c.Set(x, y, z, terrain(oy+int32(y), h))
			}
		}
	}
	g.placeTrees(c, p, &heights)
	c.compact()
	return c
}

// terrain picks the block at height y in a column whose surface is h.
func terrain(y, h int32) Block {
	beach := h <= seaLevel+1
	switch {
	case y == 0:
		return Bedrock
	case y < h-3:
		return Stone
	case y < h && beach:
		return Sand
	case y < h:
		return Dirt
	case y == h && beach:
		return Sand
	case y == h:
		return Grass
	case y <= seaLevel:
		return Water
	}
	return Air
}

// treeAt returns the trunk height of a tree rooted in column x,z, if any.
func (g *GenV1) treeAt(x, z, h int32) (int32, bool) {
	if h <= seaLevel+1 || h+10 >= MaxY {
		return 0, false
	}
	r := hash(g.trees, x, z)
	if r%1000 >= treePerMille {
		return 0, false
	}
	return 4 + int32(r>>32%3), true
}

// placeTrees draws every tree that touches chunk p: all trunks first, then leaves into air.
func (g *GenV1) placeTrees(c *Chunk, p ChunkPos, heights *[Size + 2*treeReach][Size + 2*treeReach]int32) {
	ox, oy, oz := p.Origin()
	put := func(x, y, z int32, b Block, onlyAir bool) {
		lx, ly, lz := x-ox, y-oy, z-oz
		if lx < 0 || ly < 0 || lz < 0 || lx >= Size || ly >= Size || lz >= Size {
			return
		}
		if !onlyAir || c.Get(int(lx), int(ly), int(lz)) == Air {
			c.Set(int(lx), int(ly), int(lz), b)
		}
	}
	for pass := range 2 {
		for i := range heights {
			for j := range heights[i] {
				x, z, h := ox+int32(i)-treeReach, oz+int32(j)-treeReach, heights[i][j]
				trunk, ok := g.treeAt(x, z, h)
				if !ok {
					continue
				}
				if pass == 0 {
					for y := h + 1; y <= h+trunk; y++ {
						put(x, y, z, Log, false)
					}
				} else {
					leaves(x, h+trunk, z, func(x, y, z int32) { put(x, y, z, Leaves, true) })
				}
			}
		}
	}
}

// leaves calls put for each leaf of a tree whose top log is at x,top,z.
func leaves(x, top, z int32, put func(x, y, z int32)) {
	for dy := int32(-2); dy <= 1; dy++ {
		r := int32(treeReach)
		if dy >= 0 {
			r = 1
		}
		for dx := -r; dx <= r; dx++ {
			for dz := -r; dz <= r; dz++ {
				corner := abs(dx) == r && abs(dz) == r
				if corner && (r == treeReach || dy == 1) {
					continue
				}
				put(x+dx, top+dy, z+dz)
			}
		}
	}
}

func abs(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}
