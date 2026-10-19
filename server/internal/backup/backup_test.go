package backup

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"scabbage/internal/store"
)

func TestNextRun(t *testing.T) {
	loc := time.FixedZone("test", 0)
	cases := []struct{ now, want time.Time }{
		{time.Date(2026, 5, 1, 1, 0, 0, 0, loc), time.Date(2026, 5, 1, 4, 0, 0, 0, loc)},
		{time.Date(2026, 5, 1, 4, 0, 0, 0, loc), time.Date(2026, 5, 2, 4, 0, 0, 0, loc)},
		{time.Date(2026, 5, 31, 23, 0, 0, 0, loc), time.Date(2026, 6, 1, 4, 0, 0, 0, loc)},
	}
	for _, c := range cases {
		if got := NextRun(c.now, 4); !got.Equal(c.want) {
			t.Errorf("NextRun(%v) = %v, want %v", c.now, got, c.want)
		}
	}
}

func TestSnapshotCopiesDatabase(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "scabbage.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(`INSERT INTO worlds (name, seed, generator, radius, created_ms) VALUES ('main', 1, 1, 16, 0)`); err != nil {
		t.Fatal(err)
	}
	b := Backups{DB: db, Dir: filepath.Join(dir, "backups"), Keep: 2}
	path, err := b.Snapshot(time.Date(2026, 5, 1, 4, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "scabbage-20260501-040000.db" {
		t.Fatalf("path = %s", path)
	}
	copied, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = copied.Close() }()
	var name string
	if err := copied.QueryRow(`SELECT name FROM worlds`).Scan(&name); err != nil || name != "main" {
		t.Fatalf("name = %q, err = %v", name, err)
	}
}

func TestSnapshotKeepsNewest(t *testing.T) {
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "scabbage.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	b := Backups{DB: db, Dir: filepath.Join(dir, "backups"), Keep: 2}
	start := time.Date(2026, 5, 1, 4, 0, 0, 0, time.UTC)
	for day := range 4 {
		if _, err := b.Snapshot(start.AddDate(0, 0, day)); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(b.Dir, "notes.txt"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Snapshot(start.AddDate(0, 0, 4)); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(b.Dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	want := []string{"notes.txt", "scabbage-20260504-040000.db", "scabbage-20260505-040000.db"}
	if !slices.Equal(names, want) {
		t.Fatalf("files = %v, want %v", names, want)
	}
}
