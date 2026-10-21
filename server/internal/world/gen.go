package world

import (
	"errors"
	"fmt"
	"math"
)

// Generator deterministically fills chunks from a seed. A changed algorithm needs a new version.
type Generator interface {
	Version() uint32
	Generate(p ChunkPos) *Chunk
}

// LatestGenerator is the version new worlds use.
const LatestGenerator = 3

// ErrOldScale means a world was made when blocks were a metre across, which this server can't load.
var ErrOldScale = errors.New("this world was made for bigger blocks; start a new one in an empty data directory")

// NewGenerator returns the generator for a stored version.
func NewGenerator(version uint32, seed int64) (Generator, error) {
	switch version {
	case 1, 2:
		return nil, ErrOldScale
	case 3:
		return NewGenV3(seed), nil
	}
	return nil, fmt.Errorf("unknown world generator version %d", version)
}

const (
	seaLevel     = 30 * Metre
	baseHeight   = 36 * Metre
	heightRange  = 30 * Metre
	heightScale  = 160.0 * Metre
	hillRange    = 48 * Metre
	hillScale    = 480.0 * Metre
	beachRise    = Metre      // land this little above the sea is sand
	soilDepth    = 3 * Metre  // dirt or sand over the stone
	treePerMille = 12         // trees per thousand square metres
	treeReach    = 2          // leaf radius in metres beyond the trunk
	treeRoom     = 10 * Metre // headroom a tree needs under the sky
)

// GenV3 makes rolling hills with beaches, lakes, scattered trees and dungeons, in blocks a third
// of a metre across.
type GenV3 struct {
	height, hills   noise
	trees, dungeons uint64
}

// NewGenV3 returns a version 3 generator for seed.
func NewGenV3(seed int64) *GenV3 {
	return &GenV3{height: newNoise(seed), hills: newNoise(seed ^ 0x68696c6c),
		trees: mix(uint64(seed) ^ 0x7265657473), dungeons: mix(uint64(seed) ^ 0x64756e67656f6e)}
}

// Version implements Generator.
func (g *GenV3) Version() uint32 { return 3 }

// SurfaceY returns the y of the topmost terrain block in column x,z.
func (g *GenV3) SurfaceY(x, z int32) int32 {
	n := g.height.fbm(float64(x)/heightScale, float64(z)/heightScale, 4)
	hill := max(0, g.hills.fbm(float64(x)/hillScale, float64(z)/hillScale, 2))
	return baseHeight + int32(math.Floor(float64(heightRange*n)+float64(hillRange*float64(hill*hill))))
}

// Generate implements Generator.
func (g *GenV3) Generate(p ChunkPos) *Chunk {
	c := NewChunk(Air)
	ox, oy, oz := p.Origin()
	if oy >= MaxY || oy+Size <= MinY {
		return c
	}
	for x := range Size {
		for z := range Size {
			h := g.SurfaceY(ox+int32(x), oz+int32(z))
			for y := range Size {
				c.Set(x, y, z, terrain(oy+int32(y), h))
			}
		}
	}
	put := clipTo(c, p)
	g.placeTrees(p, put)
	g.placeDungeon(p, put)
	c.compact()
	return c
}

// clipTo returns a put that writes world positions inside chunk p and ignores the rest.
func clipTo(c *Chunk, p ChunkPos) func(x, y, z int32, b Block, onlyAir bool) {
	ox, oy, oz := p.Origin()
	return func(x, y, z int32, b Block, onlyAir bool) {
		lx, ly, lz := x-ox, y-oy, z-oz
		if lx < 0 || ly < 0 || lz < 0 || lx >= Size || ly >= Size || lz >= Size {
			return
		}
		if !onlyAir || c.Get(int(lx), int(ly), int(lz)) == Air {
			c.Set(int(lx), int(ly), int(lz), b)
		}
	}
}

// terrain picks the block at height y in a column whose surface is h.
func terrain(y, h int32) Block {
	beach := h <= seaLevel+beachRise
	switch {
	case y < Metre:
		return Bedrock
	case y < h-soilDepth:
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

// tree is a trunk a metre square whose corner is at x,z, standing on ground h.
type tree struct {
	x, z, ground, top int32 // top is the y of the highest log
}

// treeAt returns the tree rooted in the square metre mx,mz, if any.
func (g *GenV3) treeAt(mx, mz int32) (tree, bool) {
	r := hash(g.trees, mx, mz)
	if r%1000 >= treePerMille {
		return tree{}, false
	}
	t := tree{x: mx * Metre, z: mz * Metre}
	t.ground = g.SurfaceY(t.x+Metre/2, t.z+Metre/2)
	if t.ground <= seaLevel+beachRise || t.ground+treeRoom >= MaxY {
		return tree{}, false
	}
	t.top = t.ground + (4+int32(r>>32%3))*Metre
	return t, true
}

// placeTrees draws every tree that touches chunk p: all trunks first, then leaves into air.
func (g *GenV3) placeTrees(p ChunkPos, put func(x, y, z int32, b Block, onlyAir bool)) {
	ox, _, oz := p.Origin()
	lo, hi := floorDiv(ox, Metre)-treeReach, floorDiv(ox+Size-1, Metre)+treeReach
	loZ, hiZ := floorDiv(oz, Metre)-treeReach, floorDiv(oz+Size-1, Metre)+treeReach
	var trees []tree
	for mx := lo; mx <= hi; mx++ {
		for mz := loZ; mz <= hiZ; mz++ {
			if t, ok := g.treeAt(mx, mz); ok {
				trees = append(trees, t)
			}
		}
	}
	for _, t := range trees {
		fillBox(t.x, t.ground-Metre+1, t.z, t.x+Metre-1, t.top, t.z+Metre-1, func(x, y, z int32) { put(x, y, z, Log, false) })
	}
	for _, t := range trees {
		t.leaves(func(x, y, z int32) { put(x, y, z, Leaves, true) })
	}
}

// leaves calls put for each leaf block: a canopy of metre cubes around the trunk's top metre.
func (t tree) leaves(put func(x, y, z int32)) {
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
				x, y, z := t.x+dx*Metre, t.top+dy*Metre-Metre+1, t.z+dz*Metre
				fillBox(x, y, z, x+Metre-1, y+Metre-1, z+Metre-1, put)
			}
		}
	}
}

// placeDungeon digs in the part of a dungeon that reaches into chunk p.
func (g *GenV3) placeDungeon(p ChunkPos, put func(x, y, z int32, b Block, onlyAir bool)) {
	ox, oy, oz := p.Origin()
	d, ok := g.dungeonIn(regionOf(ox), regionOf(oz))
	if !ok || oy > d.top()+d.steps+archHeight || oy+Size <= d.floor-Metre {
		return
	}
	d.build(func(x, y, z int32, b Block) { put(x, y, z, b, false) })
}

// fillBox calls put for every block from x0,y0,z0 to x1,y1,z1 inclusive.
func fillBox(x0, y0, z0, x1, y1, z1 int32, put func(x, y, z int32)) {
	for x := x0; x <= x1; x++ {
		for y := y0; y <= y1; y++ {
			for z := z0; z <= z1; z++ {
				put(x, y, z)
			}
		}
	}
}

func floorDiv(v, d int32) int32 {
	if v < 0 {
		return (v+1)/d - 1
	}
	return v / d
}

func abs(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}
