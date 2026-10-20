package zones

import (
	"errors"
	"path/filepath"
	"testing"

	"scabbage/internal/access"
	"scabbage/internal/store"
)

const (
	amy = 1
	bob = 2
)

func newService(t *testing.T, spawn int32) *Service {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, name := range []string{"Amy", "Bob"} {
		if _, err := db.Exec(`INSERT INTO players (name, color, created_ms) VALUES (?, 0, 0)`, name); err != nil {
			t.Fatal(err)
		}
	}
	s, err := New(db, spawn)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestOnlyAdminsBuildAtSpawn(t *testing.T) {
	s := newService(t, 8)
	if s.CanBuild(amy, false, 3, -8) || !s.CanBuild(amy, true, 3, -8) {
		t.Fatal("spawn should be for admins only")
	}
	if !s.CanBuild(amy, false, 9, 0) {
		t.Fatal("outside spawn anyone can build")
	}
	if z, ok := s.At(0, 0); !ok || z.Name != SpawnName {
		t.Fatalf("At(0, 0) = %+v, %v", z, ok)
	}
}

func TestNoSpawnZoneWhenSizeIsZero(t *testing.T) {
	s := newService(t, 0)
	if _, ok := s.At(0, 0); ok || len(s.All()) != 0 {
		t.Fatal("expected no zones")
	}
}

func TestBuildersOfAHouse(t *testing.T) {
	s := newService(t, 0)
	must(t, s.Put("house", 10, 10, 14, 14))
	must(t, s.Allow("HOUSE", amy, true))
	if !s.CanBuild(amy, false, 12, 14) || s.CanBuild(bob, false, 12, 14) {
		t.Fatal("only Amy should build in her house")
	}
	must(t, s.Allow("house", amy, false))
	if s.CanBuild(amy, false, 12, 12) {
		t.Fatal("Amy was disallowed")
	}
}

func TestSmallestZoneWins(t *testing.T) {
	s := newService(t, 32)
	must(t, s.Put("house", 4, 4, 6, 6))
	must(t, s.Allow("house", amy, true))
	if z, _ := s.At(5, 5); z.Name != "house" || !s.CanBuild(amy, false, 5, 5) {
		t.Fatalf("At(5, 5) = %+v", z)
	}
}

func TestZonesAreSaved(t *testing.T) {
	s := newService(t, 0)
	must(t, s.Put("farm", 0, 0, 9, 9))
	must(t, s.Allow("farm", bob, true))
	must(t, s.SetMode("farm", access.ModeSurvival))
	again, err := New(s.db, 0)
	must(t, err)
	z, ok := again.At(3, 3)
	if !ok || z.Mode != access.ModeSurvival || !z.Lets(bob) || z.Lets(amy) {
		t.Fatalf("reloaded %+v, %v", z, ok)
	}
}

func TestPutMovesAZoneButKeepsItsBuilders(t *testing.T) {
	s := newService(t, 0)
	must(t, s.Put("farm", 0, 0, 9, 9))
	must(t, s.Allow("farm", bob, true))
	must(t, s.Put("Farm", 20, 20, 29, 29))
	if _, ok := s.At(3, 3); ok {
		t.Fatal("the farm moved away")
	}
	if z, _ := s.At(25, 25); !z.Lets(bob) || len(s.All()) != 1 {
		t.Fatalf("moved farm = %+v", z)
	}
}

func TestSpawnCanBeChangedAndRestored(t *testing.T) {
	s := newService(t, 8)
	must(t, s.SetMode(SpawnName, access.ModeCreative))
	if z, _ := s.At(0, 0); z.Mode != access.ModeCreative || z.X1 != 8 {
		t.Fatalf("spawn = %+v", z)
	}
	must(t, s.Remove("spawn"))
	if z, _ := s.At(0, 0); z.Mode != access.ModeDefault || len(s.All()) != 1 {
		t.Fatalf("spawn after remove = %+v", z)
	}
}

func TestBadRequests(t *testing.T) {
	s := newService(t, 0)
	if err := s.Remove("nowhere"); !errors.Is(err, ErrNoZone) {
		t.Fatalf("Remove: %v", err)
	}
	if err := s.Allow("nowhere", amy, true); !errors.Is(err, ErrNoZone) {
		t.Fatalf("Allow: %v", err)
	}
	if err := s.Put("a-name-that-is-far-too-long", 0, 0, 1, 1); !errors.Is(err, ErrBadName) {
		t.Fatalf("Put: %v", err)
	}
	if err := s.SetMode("nowhere", "flying"); !errors.Is(err, access.ErrBadMode) {
		t.Fatalf("SetMode: %v", err)
	}
}

func TestCornersInAnyOrder(t *testing.T) {
	s := newService(t, 0)
	must(t, s.Put("pond", 5, 5, -5, -5))
	if z, ok := s.At(0, 0); !ok || z.X0 != -5 || z.X1 != 5 {
		t.Fatalf("pond = %+v, %v", z, ok)
	}
}
