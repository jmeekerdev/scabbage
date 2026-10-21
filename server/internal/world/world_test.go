package world

import (
	"bytes"
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
	if w.Meta().Seed != 42 || w.Meta().Generator != LatestGenerator || w.Meta().Radius != 64 {
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
	if _, err := w.Chunk(ChunkPos{1, 0, -2}); err != nil {
		t.Fatalf("edge chunk: %v", err)
	}
	for _, p := range []ChunkPos{{2, 0, 0}, {0, 0, -3}, {0, -1, 0}, {0, MaxY / Size, 0}} {
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
		if b.ID >= FirstCustom {
			break // other tests register custom blocks
		}
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
	data, err := w.EncodedChunk(ChunkPos{0, 1, 0})
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
	if err := w.SetBlock(1, 300, 1, Stone); err != nil {
		t.Fatal(err)
	}
	if err := w.SetBlock(2, 300, 1, Water); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		x, y, z int32
		want    bool
	}{
		{1, 300, 1, true},   // stone
		{2, 300, 1, false},  // water
		{3, 300, 1, false},  // air
		{64, 300, 0, true},  // past the border
		{0, -1, 0, true},    // below the world
		{0, MaxY, 0, false}, // open sky
	}
	for _, c := range cases {
		if got := len(w.Boxes(c.x, c.y, c.z)) > 0; got != c.want {
			t.Errorf("Boxes(%d,%d,%d) = %v, want %v", c.x, c.y, c.z, got, c.want)
		}
	}
}

func TestTickingBlocksAreIndexedWhileLoaded(t *testing.T) {
	db := openDB(t)
	w := openWorld(t, db)
	for _, p := range []Pos{{1, 50, 1}, {2, 50, 1}} {
		if err := w.SetBlock(p.X, p.Y, p.Z, Timer); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.SetBlock(2, 50, 1, Stone); err != nil {
		t.Fatal(err)
	}
	if got := w.Ticking(); len(got) != 1 || got[0] != (Pos{1, 50, 1}) {
		t.Fatalf("ticking = %v", got)
	}
	if err := w.Unload(); err != nil || len(w.Ticking()) != 0 {
		t.Fatalf("unloaded chunks still tick: %v", err)
	}
	if _, err := w.Block(1, 50, 1); err != nil || len(openWorld(t, db).Ticking()) != 0 {
		t.Fatal(err)
	}
	if got := w.Ticking(); len(got) != 1 {
		t.Fatalf("loading the chunk again should find its timer: %v", got)
	}
}

func TestOldScaleWorldsRefuseToOpen(t *testing.T) {
	db := openDB(t)
	if _, err := db.Exec(`INSERT INTO worlds (name, seed, generator, radius, created_ms) VALUES ('main', 42, 2, 64, 0)`); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(NewStore(db), "main", Defaults{}); !errors.Is(err, ErrOldScale) {
		t.Fatalf("err = %v", err)
	}
}

func TestTopIsJustAboveAColumnsHighestBlock(t *testing.T) {
	w := openWorld(t, openDB(t))
	y, err := w.Top(5, 5)
	if err != nil || y <= NewGenV3(42).SurfaceY(5, 5) {
		t.Fatalf("top = %d, %v", y, err)
	}
	if err := w.SetBlock(5, y+3, 5, Stone); err != nil {
		t.Fatal(err)
	}
	if again, _ := w.Top(5, 5); again != y+4 {
		t.Fatalf("top under a floating block = %d, want %d", again, y+4)
	}
	if _, err := w.Top(10000, 0); err == nil {
		t.Fatal("no error beyond the border")
	}
}

func TestWipeForgetsEditsAndCanChangeTheSeed(t *testing.T) {
	db := openDB(t)
	w := openWorld(t, db)
	if err := w.SetBlock(1, 300, 1, Brick); err != nil {
		t.Fatal(err)
	}
	if err := w.Flush(); err != nil {
		t.Fatal(err)
	}
	if err := w.Wipe(42); err != nil {
		t.Fatal(err)
	}
	if b, _ := w.Block(1, 300, 1); b != Air || w.Meta().Seed != 42 {
		t.Fatalf("after reset: block %v, seed %d", b, w.Meta().Seed)
	}
	fresh, _ := Open(NewStore(openDB(t)), "main", Defaults{Seed: 7, Radius: 64})
	want, _ := fresh.EncodedChunk(ChunkPos{0, 2, 0})
	if err := w.Wipe(7); err != nil {
		t.Fatal(err)
	}
	got, _ := w.EncodedChunk(ChunkPos{0, 2, 0})
	if w.Meta().Seed != 7 || !bytes.Equal(got, want) {
		t.Fatal("the new world isn't seed 7's land")
	}
	again, _ := Open(NewStore(db), "main", Defaults{})
	if again.Meta().Seed != 7 {
		t.Fatalf("stored seed %d", again.Meta().Seed)
	}
}
