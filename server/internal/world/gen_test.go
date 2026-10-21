package world

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"testing"
)

func TestNoiseIsDeterministicAndBounded(t *testing.T) {
	a, b := newNoise(7), newNoise(7)
	for i := range 2000 {
		x, z := float64(i)*0.37-300, float64(i)*-0.53+90
		va := a.fbm(x, z, 4)
		if va != b.fbm(x, z, 4) {
			t.Fatal("same seed gave different noise")
		}
		if va < -1 || va > 1 {
			t.Fatalf("fbm(%v,%v) = %v out of [-1,1]", x, z, va)
		}
	}
	if newNoise(7).fbm(10.5, 3.25, 4) == newNoise(8).fbm(10.5, 3.25, 4) {
		t.Error("different seeds gave identical noise")
	}
}

func TestGenerateIsDeterministic(t *testing.T) {
	p := ChunkPos{3, 3, -2}
	a := Encode(NewGenV3(42).Generate(p))
	if !bytes.Equal(a, Encode(NewGenV3(42).Generate(p))) {
		t.Fatal("same seed and position gave different chunks")
	}
	if bytes.Equal(a, Encode(NewGenV3(43).Generate(p))) {
		t.Fatal("different seeds gave identical chunks")
	}
}

func TestGenV3Layers(t *testing.T) {
	g := NewGenV3(42)
	bottom := g.Generate(ChunkPos{0, 0, 0})
	if bottom.Get(5, 0, 5) != Bedrock || bottom.Get(5, Metre-1, 5) != Bedrock {
		t.Error("the bottom metre should be bedrock")
	}
	if top := g.Generate(ChunkPos{0, MaxY/Size - 1, 0}); !top.Uniform() || top.Get(0, 0, 0) != Air {
		t.Error("the top chunk should be empty air")
	}
	h := g.SurfaceY(5, 5)
	if h < Metre || h >= MaxY-Size {
		t.Fatalf("surface %d out of range", h)
	}
	surface := blockAt(g, 5, h, 5)
	if surface != Grass && surface != Sand {
		t.Errorf("surface block = %v, want grass or sand", surface)
	}
	if above := blockAt(g, 5, h+1, 5); above != Air && above != Water && above != Log {
		t.Errorf("block above surface = %v", above)
	}
}

// Features near chunk edges must be complete in every chunk they touch.
func TestFeaturesAreSeamless(t *testing.T) {
	g := NewGenV3(42)
	at := cachedBlocks(g)
	seen := map[featureKind]int{}
	for mx := int32(-30); mx < 30; mx++ {
		for mz := int32(-30); mz < 30; mz++ {
			f, ok := g.featureAt(mx, mz)
			if !ok {
				continue
			}
			seen[f.kind]++
			f.solid(anywhere(func(x, y, z int32, b Block, _ bool) {
				if got := at(x, y, z); !Info(got).Solid {
					t.Fatalf("%v at %d,%d: %v at %d,%d,%d, want %v", f.kind, f.x, f.z, got, x, y, z, b)
				}
			}))
			f.foliage(anywhere(func(x, y, z int32, b Block, _ bool) {
				if got := at(x, y, z); got == Air || Info(got).Plant() {
					t.Fatalf("%v at %d,%d: nothing at %d,%d,%d, want %v", f.kind, f.x, f.z, x, y, z, b)
				}
			}))
		}
	}
	if len(seen) < 3 {
		t.Fatalf("features found: %v", seen)
	}
}

func TestEveryKindOfFeatureGrows(t *testing.T) {
	g := NewGenV3(42)
	seen := map[featureKind]int{}
	for mx := int32(-200); mx < 200; mx++ {
		for mz := int32(-200); mz < 200; mz++ {
			if f, ok := g.featureAt(mx, mz); ok {
				seen[f.kind]++
			}
		}
	}
	for k := oakTree; k <= rock; k++ {
		if seen[k] == 0 {
			t.Errorf("no %v in 400 m square: %v", k, seen)
		}
	}
}

func TestTreesHaveLeavesBesideTheirTop(t *testing.T) {
	for _, k := range []featureKind{oakTree, pineTree, slimTree} {
		f := feature{kind: k, x: 30, z: 30, ground: 100}
		tr := f.tree()
		blocks := map[[3]int32]Block{}
		f.foliage(anywhere(func(x, y, z int32, b Block, _ bool) { blocks[[3]int32{x, y, z}] = b }))
		for _, d := range [][2]int32{{-1, 0}, {tr.width, 0}, {0, -1}, {0, tr.width}} {
			if blocks[[3]int32{tr.x + d[0], tr.top, tr.z + d[1]}] == Air {
				t.Errorf("%v has no leaves beside its top log at %v", k, d)
			}
		}
		if tr.top < f.ground+3*Metre {
			t.Errorf("%v is only %d blocks tall", k, tr.top-f.ground)
		}
	}
}

func TestPlantsGrowOnGrass(t *testing.T) {
	g := NewGenV3(42)
	at := cachedBlocks(g)
	plants := map[Block]int{}
	for x := int32(-64); x < 64; x++ {
		for z := int32(-64); z < 64; z++ {
			h := g.SurfaceY(x, z)
			b := at(x, h+1, z)
			if !Info(b).Plant() {
				continue
			}
			plants[b]++
			if under := at(x, h, z); under != Grass {
				t.Fatalf("%v at %d,%d,%d grows on %v", b, x, h+1, z, under)
			}
		}
	}
	if plants[TallGrass] == 0 {
		t.Fatalf("plants = %v", plants)
	}
	flowers := map[Block]bool{}
	for x := int32(-3000); x < 3000; x += 5 {
		for z := int32(-3000); z < 3000; z += 5 {
			if c := g.column(x, z); Info(g.plantAt(x, z, c)).Shape == Flower {
				flowers[g.plantAt(x, z, c)] = true
			}
		}
	}
	if len(flowers) != len(flowerColors) {
		t.Fatalf("flowers = %v", flowers)
	}
}

func TestBeachesAreSandOrPebblesAndCragsBareRock(t *testing.T) {
	g := NewGenV3(42)
	tops := map[Block]int{}
	for x := int32(-3000); x < 3000; x += 7 {
		for z := int32(-3000); z < 3000; z += 7 {
			c := g.column(x, z)
			tops[c.top]++
			beach := c.h <= seaLevel+beachRise
			if (c.top == Sand || c.top == Gravel) != beach {
				t.Fatalf("column %d,%d at %d is %v", x, z, c.h, c.top)
			}
		}
	}
	for _, b := range []Block{Grass, Sand, Gravel, Stone} {
		if tops[b] == 0 {
			t.Fatalf("surfaces = %v", tops)
		}
	}
}

func TestBanksRiseInSteps(t *testing.T) {
	g := NewGenV3(42)
	near := 0
	for x := int32(-3000); x < 3000; x += 7 {
		for z := int32(-3000); z < 3000; z += 7 {
			h := g.SurfaceY(x, z)
			if abs(h-seaLevel) < bankBand {
				near++
				if (h-seaLevel)%bankStep != 0 {
					t.Fatalf("bank at %d,%d is %d above the sea", x, z, h-seaLevel)
				}
			}
		}
	}
	if near == 0 {
		t.Fatal("no banks found")
	}
}

// Golden hashes pin generator v3, through the room and arch of the dungeon in region -4,-4.
func TestGenV3Golden(t *testing.T) {
	g := NewGenV3(42)
	h := sha256.New()
	for _, p := range []ChunkPos{{0, 3, 0}, {0, 4, 0}, {-19, 2, -20}, {-22, 3, -20}} {
		h.Write(Encode(g.Generate(p)))
	}
	const want = "d62d2972fc2bac4d30e67a8f1bbd84efa6256730052b03a87132e1d7040829d9"
	if got := hex.EncodeToString(h.Sum(nil)); got != want {
		t.Fatalf("generator v3 output changed: %s", got)
	}
}

// anywhere is a canvas over the whole world.
func anywhere(put putFunc) canvas {
	return canvas{put: put, lo: [3]int32{math.MinInt32, MinY, math.MinInt32}, hi: [3]int32{math.MaxInt32, MaxY, math.MaxInt32}}
}

func blockAt(g Generator, x, y, z int32) Block {
	p, lx, ly, lz := ChunkOf(x, y, z)
	return g.Generate(p).Get(lx, ly, lz)
}

// cachedBlocks returns a block lookup that generates each chunk once.
func cachedBlocks(g Generator) func(x, y, z int32) Block {
	chunks := map[ChunkPos]*Chunk{}
	return func(x, y, z int32) Block {
		p, lx, ly, lz := ChunkOf(x, y, z)
		if chunks[p] == nil {
			chunks[p] = g.Generate(p)
		}
		return chunks[p].Get(lx, ly, lz)
	}
}

func BenchmarkGenerateSurfaceChunk(b *testing.B) {
	g := NewGenV3(42)
	for i := range b.N {
		g.Generate(ChunkPos{int32(i % 64), 3, int32(i / 64 % 64)})
	}
}

func BenchmarkEncodeSurfaceChunk(b *testing.B) {
	c := NewGenV3(42).Generate(ChunkPos{0, 3, 0})
	b.ReportAllocs()
	for range b.N {
		Encode(c)
	}
	b.ReportMetric(float64(len(Encode(c))), "bytes")
}
