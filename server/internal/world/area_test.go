package world

import (
	"errors"
	"slices"
	"testing"
)

func TestAreaIndexIsLayerByLayer(t *testing.T) {
	a := Area{X0: -3, Z0: 5, X1: 2, Z1: 8}
	if a.Width() != 6 || a.Depth() != 4 || a.Volume() != 6*4*MaxY {
		t.Fatalf("size %d×%d, volume %d", a.Width(), a.Depth(), a.Volume())
	}
	if a.Index(-3, 0, 5) != 0 || a.Index(-2, 0, 5) != 1 || a.Index(-3, 0, 6) != 6 || a.Index(-3, 1, 5) != 24 {
		t.Fatal("index isn't x, then z, then y")
	}
}

func TestCopyThenPasteRestoresAnArea(t *testing.T) {
	w := openWorld(t, openDB(t))
	a := Area{X0: -20, Z0: 10, X1: -5, Z1: 30} // across chunk borders
	before, err := w.Copy(a)
	if err != nil || len(before) != a.Volume() {
		t.Fatalf("copy: %d blocks, %v", len(before), err)
	}
	mustSet(t, w, -10, 50, 20, Brick)
	mustSet(t, w, -6, 3, 29, Air)
	mustSet(t, w, -4, 50, 20, Brick) // outside the area, so it stays
	changes, err := w.Paste(a, before)
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 2 || changes[0].Pos != (Pos{-10, 50, 20}) || changes[0].Block != before[a.Index(-10, 50, 20)] {
		t.Fatalf("changes = %+v", changes)
	}
	after, _ := w.Copy(a)
	if !slices.Equal(after, before) {
		t.Fatal("paste didn't restore the area")
	}
	if b, _ := w.Block(-4, 50, 20); b != Brick {
		t.Fatal("paste reached outside its area")
	}
}

func TestPasteIndexesTickingBlocksAndMarksDirty(t *testing.T) {
	w := openWorld(t, openDB(t))
	a := Area{X0: 0, Z0: 0, X1: 1, Z1: 1}
	blocks, _ := w.Copy(a)
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	blocks[a.Index(1, 60, 1)] = Timer
	if _, err := w.Paste(a, blocks); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(w.Ticking(), Pos{1, 60, 1}) || w.Dirty() == 0 {
		t.Fatalf("ticking %v, dirty %d", w.Ticking(), w.Dirty())
	}
}

func TestPasteRefusesTheWrongSizeOrOutsideTheBorder(t *testing.T) {
	w := openWorld(t, openDB(t)) // radius 64
	a := Area{X0: 0, Z0: 0, X1: 1, Z1: 1}
	if _, err := w.Paste(a, make([]Block, 3)); err == nil {
		t.Fatal("pasted the wrong number of blocks")
	}
	out := Area{X0: 60, Z0: 0, X1: 70, Z1: 1}
	if _, err := w.Copy(out); !errors.Is(err, ErrOutside) {
		t.Fatalf("copy outside: %v", err)
	}
	if _, err := w.Paste(out, make([]Block, out.Volume())); !errors.Is(err, ErrOutside) {
		t.Fatalf("paste outside: %v", err)
	}
}

func TestEncodeBlocksRoundTrip(t *testing.T) {
	blocks := make([]Block, 5000)
	for i := range blocks {
		blocks[i] = Block(i / 700 % 3)
	}
	data := EncodeBlocks(blocks)
	got, err := DecodeBlocks(data, len(blocks))
	if err != nil || !slices.Equal(got, blocks) {
		t.Fatalf("round trip failed: %v", err)
	}
	if len(data) > 64 {
		t.Fatalf("runs of the same block took %d bytes", len(data))
	}
	if _, err := DecodeBlocks(data, len(blocks)+1); err == nil {
		t.Fatal("decoded the wrong number of blocks")
	}
	same, err := DecodeBlocks(EncodeBlocks(make([]Block, 9)), 9)
	if err != nil || len(same) != 9 {
		t.Fatalf("uniform: %d, %v", len(same), err)
	}
}

func mustSet(t *testing.T, w *World, x, y, z int32, b Block) {
	t.Helper()
	if err := w.SetBlock(x, y, z, b); err != nil {
		t.Fatal(err)
	}
}
