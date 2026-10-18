package world

import (
	"database/sql"
	"errors"
	"path/filepath"
	"sync"
	"testing"

	"scabbage/internal/store"
)

func openDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "w.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func openWorld(t *testing.T, db *sql.DB) *World {
	t.Helper()
	w, err := Open(NewStore(db), "main", Defaults{Seed: 42, Radius: 64})
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func TestOpenCreatesWorldOnceAndKeepsSeed(t *testing.T) {
	db := openDB(t)
	w := openWorld(t, db)
	if w.Meta().Seed != 42 || w.Meta().Generator != 1 || w.Meta().Radius != 64 {
		t.Fatalf("meta = %+v", w.Meta())
	}
	again, err := Open(NewStore(db), "main", Defaults{Seed: 7, Radius: 16})
	if err != nil || again.Meta() != w.Meta() {
		t.Fatalf("reopen meta = %+v (%v), want %+v", again.Meta(), err, w.Meta())
	}
}

func TestRandomSeedWhenUnset(t *testing.T) {
	w, err := Open(NewStore(openDB(t)), "main", Defaults{})
	if err != nil || w.Meta().Seed == 0 || w.Meta().Radius != DefaultRadius {
		t.Fatalf("meta = %+v (%v)", w.Meta(), err)
	}
}

func TestChunkIsGeneratedThenCached(t *testing.T) {
	w := openWorld(t, openDB(t))
	p := ChunkPos{0, 1, 0}
	a, err := w.Chunk(p)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := w.Chunk(p)
	if a != b || w.Loaded() != 1 || w.Dirty() != 1 {
		t.Fatalf("loaded=%d dirty=%d same=%v", w.Loaded(), w.Dirty(), a == b)
	}
}

func TestEditsSurviveFlushAndReopen(t *testing.T) {
	db := openDB(t)
	w := openWorld(t, db)
	if err := w.SetBlock(3, 60, -4, Brick); err != nil {
		t.Fatal(err)
	}
	if err := w.Flush(); err != nil || w.Dirty() != 0 {
		t.Fatalf("flush: %v dirty=%d", err, w.Dirty())
	}
	w2 := openWorld(t, db)
	if b, err := w2.Block(3, 60, -4); err != nil || b != Brick {
		t.Fatalf("after reopen block = %v (%v), want brick", b, err)
	}
	if w2.Dirty() != 0 {
		t.Error("loading a stored chunk should not mark it dirty")
	}
}

func TestUnloadFlushesAndFreesChunks(t *testing.T) {
	db := openDB(t)
	w := openWorld(t, db)
	if err := w.SetBlock(1, 50, 1, Planks); err != nil {
		t.Fatal(err)
	}
	if err := w.Unload(); err != nil || w.Loaded() != 0 {
		t.Fatalf("unload: %v loaded=%d", err, w.Loaded())
	}
	if b, _ := w.Block(1, 50, 1); b != Planks {
		t.Fatalf("block = %v after unload", b)
	}
}

func TestBorderLimitsChunksAndEdits(t *testing.T) {
	w := openWorld(t, openDB(t))
	if _, err := w.Chunk(ChunkPos{3, 0, -4}); err != nil {
		t.Fatalf("edge chunk: %v", err)
	}
	for _, p := range []ChunkPos{{4, 0, 0}, {0, 0, -5}, {0, -1, 0}, {0, MaxY / Size, 0}} {
		if _, err := w.Chunk(p); !errors.Is(err, ErrOutside) {
			t.Errorf("Chunk(%v) err = %v, want ErrOutside", p, err)
		}
	}
	for _, b := range [][3]int32{{64, 40, 0}, {-65, 40, 0}, {0, -1, 0}, {0, MaxY, 0}} {
		if err := w.SetBlock(b[0], b[1], b[2], Stone); !errors.Is(err, ErrOutside) {
			t.Errorf("SetBlock(%v) err = %v, want ErrOutside", b, err)
		}
	}
	if err := w.SetBlock(-64, 40, 63, Stone); err != nil {
		t.Errorf("SetBlock inside border: %v", err)
	}
}

func TestUnboundedWorld(t *testing.T) {
	w, err := Open(NewStore(openDB(t)), "big", Defaults{Seed: 1, Radius: Unbounded})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Chunk(ChunkPos{100000, 1, -100000}); err != nil {
		t.Fatalf("far chunk: %v", err)
	}
}

func TestSetBlockRejectsUnknownBlock(t *testing.T) {
	w := openWorld(t, openDB(t))
	if err := w.SetBlock(0, 40, 0, Block(9999)); !errors.Is(err, ErrUnknownBlock) {
		t.Fatalf("err = %v", err)
	}
}

func TestStarterBlocks(t *testing.T) {
	if n := len(Blocks()); n < 10 {
		t.Fatalf("%d blocks, want at least 10", n)
	}
	if Info(Air).Solid || !Info(Stone).Solid || Info(Water).Solid || !Info(Glass).Transparent {
		t.Error("block properties wrong")
	}
	for i, b := range Blocks() {
		if b.ID != Block(i) || b.Name == "" {
			t.Errorf("block %d = %+v", i, b)
		}
	}
}

func TestSpawnStandsOnTopOfTerrain(t *testing.T) {
	w := openWorld(t, openDB(t))
	x, y, z := w.Spawn()
	below, _ := w.Block(int32(x), int32(y)-1, int32(z))
	at, _ := w.Block(int32(x), int32(y), int32(z))
	if below == Air || at != Air {
		t.Fatalf("spawn %v,%v,%v: below=%v at=%v", x, y, z, below, at)
	}
}

func TestEncodedChunkMatchesStoredBlocks(t *testing.T) {
	w := openWorld(t, openDB(t))
	if err := w.SetBlock(1, 40, 2, Brick); err != nil {
		t.Fatal(err)
	}
	data, err := w.EncodedChunk(ChunkPos{0, 2, 0})
	if err != nil {
		t.Fatal(err)
	}
	c, err := Decode(data)
	if err != nil || c.Get(1, 8, 2) != Brick {
		t.Fatalf("decoded block = %v (%v)", c.Get(1, 8, 2), err)
	}
}

func TestEncodedChunkOutsideIsAir(t *testing.T) {
	w := openWorld(t, openDB(t))
	data, err := w.EncodedChunk(ChunkPos{100, 2, 0})
	if err != nil {
		t.Fatal(err)
	}
	if c, _ := Decode(data); !c.Uniform() || c.Get(0, 0, 0) != Air || w.Loaded() != 0 {
		t.Fatalf("outside chunk should be unloaded air, loaded=%d", w.Loaded())
	}
}

// Sessions read chunks on their own goroutines while the hub unloads; run with -race.
func TestWorldIsSafeForConcurrentUse(t *testing.T) {
	w := openWorld(t, openDB(t))
	var wg sync.WaitGroup
	for i := range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range 20 {
				_, _ = w.EncodedChunk(ChunkPos{int32(j % 4), int32(i % 4), 0})
				_ = w.SetBlock(int32(j), 50, int32(i), Planks)
			}
		}()
	}
	for range 5 {
		_ = w.Unload()
	}
	wg.Wait()
}

func TestSolidWallsInTheBorderAndFloor(t *testing.T) {
	w := openWorld(t, openDB(t))
	if err := w.SetBlock(1, 100, 1, Stone); err != nil {
		t.Fatal(err)
	}
	if err := w.SetBlock(2, 100, 1, Water); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		x, y, z int32
		want    bool
	}{
		{1, 100, 1, true},   // stone
		{2, 100, 1, false},  // water
		{3, 100, 1, false},  // air
		{64, 100, 0, true},  // past the border
		{0, -1, 0, true},    // below the world
		{0, MaxY, 0, false}, // open sky
	}
	for _, c := range cases {
		if got := w.Solid(c.x, c.y, c.z); got != c.want {
			t.Errorf("Solid(%d,%d,%d) = %v, want %v", c.x, c.y, c.z, got, c.want)
		}
	}
}
