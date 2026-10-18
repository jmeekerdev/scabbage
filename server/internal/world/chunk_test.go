package world

import (
	"bytes"
	"testing"
)

func TestChunkOfHandlesNegativeCoordinates(t *testing.T) {
	cases := []struct {
		x, y, z int32
		want    ChunkPos
		local   [3]int
	}{
		{0, 0, 0, ChunkPos{0, 0, 0}, [3]int{0, 0, 0}},
		{15, 16, 17, ChunkPos{0, 1, 1}, [3]int{15, 0, 1}},
		{-1, -16, -17, ChunkPos{-1, -1, -2}, [3]int{15, 0, 15}},
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
			layered.Set(x, 15, z, Grass)
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
