package world

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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
	p := ChunkPos{3, 1, -2}
	a := Encode(NewGenV1(42).Generate(p))
	if !bytes.Equal(a, Encode(NewGenV1(42).Generate(p))) {
		t.Fatal("same seed and position gave different chunks")
	}
	if bytes.Equal(a, Encode(NewGenV1(43).Generate(p))) {
		t.Fatal("different seeds gave identical chunks")
	}
}

func TestGenV1Layers(t *testing.T) {
	g := NewGenV1(42)
	bottom := g.Generate(ChunkPos{0, 0, 0})
	if bottom.Get(5, 0, 5) != Bedrock {
		t.Error("y=0 should be bedrock")
	}
	if top := g.Generate(ChunkPos{0, MaxY/Size - 1, 0}); !top.Uniform() || top.Get(0, 0, 0) != Air {
		t.Error("the top chunk should be empty air")
	}
	h := g.SurfaceY(5, 5)
	if h < 1 || h >= MaxY-16 {
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

// Trees near chunk edges must be complete in every chunk they touch.
func TestTreesAreSeamless(t *testing.T) {
	g := NewGenV1(42)
	at := cachedBlocks(g)
	tops := 0
	for cx := int32(-4); cx < 4; cx++ {
		for cz := int32(-4); cz < 4; cz++ {
			for x := cx * Size; x < (cx+1)*Size; x++ {
				for z := cz * Size; z < (cz+1)*Size; z++ {
					top, ok := treeTop(g, at, x, z)
					if !ok {
						continue
					}
					tops++
					for _, d := range [][2]int32{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
						if b := at(x+d[0], top, z+d[1]); b != Leaves && b != Log {
							t.Fatalf("tree at %d,%d missing leaves at %d,%d: %v", x, z, x+d[0], z+d[1], b)
						}
					}
				}
			}
		}
	}
	if tops == 0 {
		t.Fatal("no trees found in test area")
	}
}

// Golden hashes pin generator v1: if this fails, the change needs a new generator version.
func TestGenV1Golden(t *testing.T) {
	g := NewGenV1(42)
	h := sha256.New()
	for _, p := range []ChunkPos{{0, 1, 0}, {0, 2, 0}, {-3, 1, 5}, {7, 2, -7}} {
		h.Write(Encode(g.Generate(p)))
	}
	const want = "1c3797855e1b776cd9f43896adf538755bb01ff77cc9933e226a07a39236f283"
	if got := hex.EncodeToString(h.Sum(nil)); got != want {
		t.Fatalf("generator v1 output changed: %s", got)
	}
}

// Golden hashes pin generator v2, through the dungeon in region 2,0.
func TestGenV2Golden(t *testing.T) {
	g := NewGenV2(42)
	h := sha256.New()
	for _, p := range []ChunkPos{{0, 1, 0}, {11, 1, 1}, {10, 1, 1}, {10, 2, 1}} {
		h.Write(Encode(g.Generate(p)))
	}
	const want = "b1f3c36bf3f993663892173683d44fdc670c445645976292db78c56f00464c96"
	if got := hex.EncodeToString(h.Sum(nil)); got != want {
		t.Fatalf("generator v2 output changed: %s", got)
	}
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

// treeTop returns the y of the topmost log if column x,z holds a trunk.
func treeTop(g *GenV1, at func(x, y, z int32) Block, x, z int32) (int32, bool) {
	h := g.SurfaceY(x, z)
	if at(x, h+1, z) != Log {
		return 0, false
	}
	y := h + 1
	for at(x, y+1, z) == Log {
		y++
	}
	return y, true
}

func BenchmarkGenerateSurfaceChunk(b *testing.B) {
	g := NewGenV1(42)
	for i := range b.N {
		g.Generate(ChunkPos{int32(i % 64), 2, int32(i / 64 % 64)})
	}
}

func BenchmarkEncodeSurfaceChunk(b *testing.B) {
	c := NewGenV1(42).Generate(ChunkPos{0, 2, 0})
	b.ReportAllocs()
	for range b.N {
		Encode(c)
	}
	b.ReportMetric(float64(len(Encode(c))), "bytes")
}
