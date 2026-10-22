package world

import (
	"bytes"
	"encoding/hex"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

func TestChunkOfHandlesNegativeCoordinates(t *testing.T) {
	cases := []struct {
		x, y, z int32
		want    ChunkPos
		local   [3]int
	}{
		{0, 0, 0, ChunkPos{0, 0, 0}, [3]int{0, 0, 0}},
		{Size - 1, Size, Size + 1, ChunkPos{0, 1, 1}, [3]int{Size - 1, 0, 1}},
		{-1, -Size, -Size - 1, ChunkPos{-1, -1, -2}, [3]int{Size - 1, 0, Size - 1}},
	}
	for _, c := range cases {
		pos, lx, ly, lz := ChunkOf(c.x, c.y, c.z)
		if pos != c.want || [3]int{lx, ly, lz} != c.local {
			t.Errorf("ChunkOf(%d,%d,%d) = %v %v, want %v %v", c.x, c.y, c.z, pos, [3]int{lx, ly, lz}, c.want, c.local)
		}
	}
}

func TestChunkStartsUniformAndMaterializesOnSet(t *testing.T) {
	c := NewChunk(Stone)
	if !c.Uniform() || c.Get(3, 4, 5) != Stone {
		t.Fatal("new chunk should be uniform stone")
	}
	c.Set(3, 4, 5, Air)
	if c.Uniform() || c.Get(3, 4, 5) != Air || c.Get(0, 0, 0) != Stone {
		t.Fatal("set should change only one block")
	}
}

func TestChunkSetSameBlockStaysUniform(t *testing.T) {
	c := NewChunk(Air)
	c.Set(1, 1, 1, Air)
	if !c.Uniform() {
		t.Fatal("setting the fill block should not allocate")
	}
}

func TestEncodeRoundTrip(t *testing.T) {
	c := NewChunk(Air)
	for x := range Size {
		for z := range Size {
			c.Set(x, 0, z, Stone)
			c.Set(x, 1, z, Dirt)
		}
	}
	c.Set(7, 9, 3, Glass)
	got, err := Decode(Encode(c))
	if err != nil {
		t.Fatal(err)
	}
	for i := range Volume {
		x, y, z := i%Size, i/(Size*Size), (i/Size)%Size
		if got.Get(x, y, z) != c.Get(x, y, z) {
			t.Fatalf("block %d,%d,%d differs", x, y, z)
		}
	}
}

func TestEncodeIsCompact(t *testing.T) {
	if n := len(Encode(NewChunk(Air))); n > 4 {
		t.Errorf("uniform chunk encodes to %d bytes", n)
	}
	layered := NewChunk(Stone)
	for x := range Size {
		for z := range Size {
			layered.Set(x, Size-1, z, Grass)
		}
	}
	if n := len(Encode(layered)); n > 16 {
		t.Errorf("two-layer chunk encodes to %d bytes", n)
	}
}

func TestDecodeUniformStaysUniform(t *testing.T) {
	c, err := Decode(Encode(NewChunk(Water)))
	if err != nil || !c.Uniform() || c.Get(0, 0, 0) != Water {
		t.Fatalf("got %v, %v", c, err)
	}
}

func TestDecodeRejectsCorruptData(t *testing.T) {
	good := Encode(NewChunk(Stone))
	cases := map[string][]byte{
		"empty":         {},
		"bad version":   append([]byte{99}, good[1:]...),
		"no palette":    {encodingVersion, 0},
		"short runs":    {encodingVersion, 2, byte(Air), byte(Stone), 10, 0},
		"bad index":     {encodingVersion, 2, byte(Air), byte(Stone), 0x80, 0x20, 5},
		"trailing data": append(bytes.Clone(good), 0),
	}
	for name, data := range cases {
		if _, err := Decode(data); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
}

var updateGolden = flag.Bool("update", false, "rewrite testdata golden files")

// goldenChunk is mirrored by client/src/world/chunk-codec.test.ts, which decodes the same file.
func goldenChunk() *Chunk {
	c := NewChunk(Stone)
	for i := range 100 {
		c.Set(i%Size, 0, i/Size, Block(i*7%13))
	}
	c.Set(Size-1, Size-1, Size-1, Glass)
	return c
}

func TestEncodeGoldenFile(t *testing.T) {
	path := filepath.Join("testdata", "golden-chunk.hex")
	got := []byte(hex.EncodeToString(Encode(goldenChunk())) + "\n")
	if *updateGolden {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("encoding changed (run with -update and bump encodingVersion): %v", err)
	}
}

func TestEncodeKeepsSecondRangeIDs(t *testing.T) {
	c := NewChunk(SecondBuiltin + 300)
	c.Set(1, 2, 3, Stone)
	got, err := Decode(Encode(c))
	if err != nil || got.Get(0, 0, 0) != SecondBuiltin+300 || got.Get(1, 2, 3) != Stone {
		t.Fatalf("decoded %v, err %v", got.Get(0, 0, 0), err)
	}
}
