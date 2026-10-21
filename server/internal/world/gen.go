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
	seaLevel    = 30 * Metre
	baseHeight  = 36 * Metre
	heightRange = 30 * Metre
	heightScale = 160.0 * Metre
	hillRange   = 48 * Metre
	hillScale   = 480.0 * Metre
	detailRange = Metre // small bumps, a metre up or down
	detailScale = 12.0 * Metre
	beachRise   = Metre     // land this little above the sea is sand
	soilDepth   = 3 * Metre // dirt or sand over the stone
	bankBand    = 2 * Metre // land this near the sea level rises in steps...
	bankStep    = 2         // ...this many blocks high
	cragScale   = 16.0 * Metre
	cragCover   = 0.4       // crag noise above this lifts the land into bare rock...
	cragRise    = 3 * Metre // ...up to this high
	pebbleScale = 6.0 * Metre
	pebbleCover = 0.25 // pebble noise above this makes a beach gravel
)

// GenV3 makes rolling hills with stepped banks, beaches, lakes, meadows, woods and dungeons, in
// blocks a third of a metre across.
type GenV3 struct {
	height, hills, detail, crags, pebbles, meadow, blooms noise
	features, plants, dungeons                            uint64
}

// NewGenV3 returns a version 3 generator for seed.
func NewGenV3(seed int64) *GenV3 {
	return &GenV3{
		height: newNoise(seed), hills: newNoise(seed ^ 0x68696c6c), detail: newNoise(seed ^ 0x64657461696c),
		crags:   newNoise(seed ^ 0x63726167),
		pebbles: newNoise(seed ^ 0x706562626c65), meadow: newNoise(seed ^ 0x6d6561646f77), blooms: newNoise(seed ^ 0x626c6f6f6d),
		features: mix(uint64(seed) ^ 0x7265657473), plants: mix(uint64(seed) ^ 0x706c616e7473),
		dungeons: mix(uint64(seed) ^ 0x64756e67656f6e),
	}
}

// Version implements Generator.
func (g *GenV3) Version() uint32 { return 3 }

// SurfaceY returns the y of the topmost terrain block in column x,z.
func (g *GenV3) SurfaceY(x, z int32) int32 {
	h, _ := g.surface(x, z)
	return h
}

// surface returns column x,z's surface height and whether it is a rocky crag.
func (g *GenV3) surface(x, z int32) (int32, bool) {
	fx, fz := float64(x), float64(z)
	n := g.height.fbm(fx/heightScale, fz/heightScale, 4)
	hill := max(0, g.hills.fbm(fx/hillScale, fz/hillScale, 2))
	bump := g.detail.fbm(fx/detailScale, fz/detailScale, 2)
	h := float64(heightRange*n) + float64(hillRange*float64(hill*hill)) + float64(detailRange*bump)
	crag := g.crag(x, z)
	return stepBanks(baseHeight + int32(math.Floor(h)) + int32(cragRise*crag)), crag > 0
}

// crag is how far into a rocky outcrop x,z is, from 0 outside to 1 at its heart.
func (g *GenV3) crag(x, z int32) float64 {
	n := g.crags.fbm(float64(x)/cragScale, float64(z)/cragScale, 2)
	return max(0, n-cragCover) / (1 - cragCover)
}

// stepBanks levels land near the sea into wide steps.
func stepBanks(h int32) int32 {
	if abs(h-seaLevel) >= bankBand {
		return h
	}
	return seaLevel + floorDiv(h-seaLevel, bankStep)*bankStep
}

// Generate implements Generator.
func (g *GenV3) Generate(p ChunkPos) *Chunk {
	c := NewChunk(Air)
	ox, oy, oz := p.Origin()
	if oy >= MaxY || oy+Size <= MinY {
		return c
	}
	cv := canvasFor(c, p)
	for x := range Size {
		for z := range Size {
			wx, wz := ox+int32(x), oz+int32(z)
			col := g.column(wx, wz)
			for y := range Size {
				c.Set(x, y, z, col.at(oy+int32(y)))
			}
			if plant := g.plantAt(wx, wz, col); plant != Air {
				cv.put(wx, col.h+1, wz, plant, true)
			}
		}
	}
	g.placeFeatures(p, cv)
	g.placeDungeon(p, cv.put)
	c.compact()
	return c
}

// putFunc writes a block at a world position; soft writes only replace air and plants.
type putFunc func(x, y, z int32, b Block, soft bool)

// canvas is where generation draws: put writes a block, and lo..hi bounds the blocks worth drawing.
type canvas struct {
	put    putFunc
	lo, hi [3]int32
}

// clamp narrows a range along axis to the canvas.
func (cv canvas) clamp(axis int, from, to int32) (int32, int32) {
	return max(from, cv.lo[axis]), min(to, cv.hi[axis])
}

// canvasFor returns a canvas that writes world positions inside chunk p and ignores the rest.
func canvasFor(c *Chunk, p ChunkPos) canvas {
	ox, oy, oz := p.Origin()
	cv := canvas{lo: [3]int32{ox, oy, oz}, hi: [3]int32{ox + Size - 1, oy + Size - 1, oz + Size - 1}}
	cv.put = func(x, y, z int32, b Block, soft bool) {
		lx, ly, lz := x-ox, y-oy, z-oz
		if lx < 0 || ly < 0 || lz < 0 || lx >= Size || ly >= Size || lz >= Size {
			return
		}
		if was := c.Get(int(lx), int(ly), int(lz)); !soft || was == Air || Info(was).Plant() {
			c.Set(int(lx), int(ly), int(lz), b)
		}
	}
	return cv
}

// column is one x,z's ground: its surface height and the blocks on and under it.
type column struct {
	h         int32
	top, soil Block
}

// column picks the ground at x,z: sand or pebbles by the water, bare rock on crags, else grass
// over dirt.
func (g *GenV3) column(x, z int32) column {
	h, crag := g.surface(x, z)
	switch {
	case h <= seaLevel+beachRise && g.pebbly(x, z):
		return column{h, Gravel, Gravel}
	case h <= seaLevel+beachRise:
		return column{h, Sand, Sand}
	case crag:
		return column{h, Stone, Stone}
	}
	return column{h, Grass, Dirt}
}

func (g *GenV3) pebbly(x, z int32) bool {
	return g.pebbles.fbm(float64(x)/pebbleScale, float64(z)/pebbleScale, 2) > pebbleCover
}

// at picks the column's block at height y.
func (c column) at(y int32) Block {
	switch {
	case y < Metre:
		return Bedrock
	case y < c.h-soilDepth:
		return Stone
	case y < c.h:
		return c.soil
	case y == c.h:
		return c.top
	case y <= seaLevel:
		return Water
	}
	return Air
}

// placeDungeon digs in the part of a dungeon that reaches into chunk p.
func (g *GenV3) placeDungeon(p ChunkPos, put putFunc) {
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
