package store

import (
	"path/filepath"
	"testing"
)

func TestOpenMigratesAndReopens(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO players (name, color, admin, created_ms) VALUES ('Ann', 1, 0, 0)`); err != nil {
		t.Fatalf("players table missing: %v", err)
	}
	_ = db.Close()

	db, err = Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() { _ = db.Close() }()
	var version, players int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != len(migrations) {
		t.Fatalf("user_version = %d, want %d", version, len(migrations))
	}
	if err := db.QueryRow(`SELECT count(*) FROM players`).Scan(&players); err != nil || players != 1 {
		t.Fatalf("players = %d (err %v), want data kept across reopen", players, err)
	}
}

func TestPlayerNamesAreCaseInsensitiveUnique(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(`INSERT INTO players (name, color, admin, created_ms) VALUES ('Ann', 1, 0, 0)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO players (name, color, admin, created_ms) VALUES ('ANN', 1, 0, 0)`); err == nil {
		t.Fatal("expected duplicate name to fail")
	}
}
