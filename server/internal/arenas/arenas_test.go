package arenas

import (
	"errors"
	"path/filepath"
	"testing"

	"scabbage/internal/store"
	"scabbage/internal/zones"
)

var field = zones.Zone{Name: "Field", X0: 0, Z0: 0, X1: 9, Z1: 9}

func newService(t *testing.T) *Service {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	z, err := zones.New(db, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := z.Put(field.Name, field.X0, field.Z0, field.X1, field.Z1); err != nil {
		t.Fatal(err)
	}
	return New(db)
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestMarksStayInsideTheZone(t *testing.T) {
	var a Arena
	must(t, a.Mark(Spawns, field, Mark{Team: Red, X: 1, Y: 5, Z: 1}))
	must(t, a.Mark(Goals, field, Mark{X: 9, Y: 0, Z: 9}))
	if err := a.Mark(Spawns, field, Mark{X: 10, Y: 5, Z: 1}); !errors.Is(err, ErrOutside) {
		t.Fatalf("got %v", err)
	}
	if err := a.Mark(Spawns, field, Mark{Team: "pink"}); !errors.Is(err, ErrBadTeam) {
		t.Fatalf("got %v", err)
	}
	if len(a.Spawns) != 1 || len(a.Goals) != 1 {
		t.Fatalf("arena %+v", a)
	}
}

func TestMarksAreLimited(t *testing.T) {
	var a Arena
	for range MaxMarks {
		must(t, a.Mark(Spawns, field, Mark{}))
	}
	if err := a.Mark(Spawns, field, Mark{}); !errors.Is(err, ErrTooMany) {
		t.Fatalf("got %v", err)
	}
}

func TestTeamsComeFromSpawnsInOrder(t *testing.T) {
	var a Arena
	for _, team := range []Team{Blue, NoTeam, Red, Blue} {
		must(t, a.Mark(Spawns, field, Mark{Team: team}))
	}
	if got := a.Teams(); len(got) != 2 || got[0] != Blue || got[1] != Red {
		t.Fatalf("teams %v", got)
	}
}

func TestWinConditions(t *testing.T) {
	var a Arena
	if a.Win != WinLast {
		t.Fatalf("new arenas are last-one-standing, got %v", a.Win)
	}
	must(t, a.SetWin(WinPoints, 5))
	if a.Win != WinPoints || a.Points != 5 {
		t.Fatalf("arena %+v", a)
	}
	for _, bad := range []struct {
		w Win
		n int
	}{{WinPoints, 0}, {WinPoints, MaxPoints + 1}, {Win(9), 0}, {WinGoal, 3}} {
		if err := a.SetWin(bad.w, bad.n); !errors.Is(err, ErrBadWin) {
			t.Errorf("SetWin(%v, %d) = %v", bad.w, bad.n, err)
		}
	}
}

func TestTimer(t *testing.T) {
	var a Arena
	must(t, a.SetTimer(90))
	must(t, a.SetTimer(0))
	if err := a.SetTimer(MaxSeconds + 1); !errors.Is(err, ErrBadTimer) {
		t.Fatalf("got %v", err)
	}
	if a.Seconds != 0 {
		t.Fatalf("arena %+v", a)
	}
}

func TestArenasAreSavedWithTheirZone(t *testing.T) {
	s := newService(t)
	var a Arena
	must(t, a.Mark(Spawns, field, Mark{Team: Red, X: 1, Y: 2, Z: 3}))
	must(t, a.SetTimer(60))
	must(t, s.Put("field", a))
	got, ok, err := s.Get("FIELD")
	if err != nil || !ok || got.Seconds != 60 || len(got.Spawns) != 1 || got.Spawns[0].Y != 2 {
		t.Fatalf("Get = %+v, %v, %v", got, ok, err)
	}
	if all, err := s.All(); err != nil || len(all) != 1 || all[0].Zone != "Field" {
		t.Fatalf("All = %+v, %v", all, err)
	}
}

func TestArenasNeedASavedZone(t *testing.T) {
	s := newService(t)
	if err := s.Put("Nowhere", Arena{}); !errors.Is(err, zones.ErrNoZone) {
		t.Fatalf("got %v", err)
	}
	if _, ok, err := s.Get("Nowhere"); ok || err != nil {
		t.Fatalf("got %v, %v", ok, err)
	}
}

func TestRemovingTheZoneRemovesTheArena(t *testing.T) {
	s := newService(t)
	must(t, s.Put("Field", Arena{}))
	if _, err := s.db.Exec(`DELETE FROM zones WHERE name = 'Field'`); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := s.Get("Field"); ok {
		t.Fatal("the arena outlived its zone")
	}
}

func TestRemove(t *testing.T) {
	s := newService(t)
	must(t, s.Put("Field", Arena{}))
	must(t, s.Remove("field"))
	if err := s.Remove("Field"); !errors.Is(err, ErrNoArena) {
		t.Fatalf("got %v", err)
	}
}

func TestWinNames(t *testing.T) {
	for _, w := range []Win{WinLast, WinGoal, WinPoints} {
		if got, ok := ParseWin(w.String()); !ok || got != w {
			t.Errorf("%v round-trips to %v", w, got)
		}
	}
	if _, ok := ParseWin("fastest"); ok {
		t.Error("fastest is not a win condition")
	}
}

func TestPortalsMayStandOutsideTheZone(t *testing.T) {
	var a Arena
	must(t, a.Mark(Portals, field, Mark{X: 999, Z: 999}))
	if len(a.Portals) != 1 || len(a.Spawns)+len(a.Goals) != 0 {
		t.Fatalf("arena %+v", a)
	}
	p := a.Portals[0]
	if !p.At(999, 0, 999) || !p.At(1000, 0, 998) || p.At(999, 1, 999) || p.At(1001, 0, 999) {
		t.Fatal("a mark covers the meter around its block, on its own level")
	}
}
