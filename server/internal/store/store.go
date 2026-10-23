// Package store opens the server's SQLite database and applies schema migrations.
package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // pure-Go SQLite driver
)

// migrations are applied in order; PRAGMA user_version records how many have run. Append only.
var migrations = []string{
	`CREATE TABLE players (
		id         INTEGER PRIMARY KEY,
		name       TEXT NOT NULL UNIQUE COLLATE NOCASE,
		color      INTEGER NOT NULL,
		admin      INTEGER NOT NULL DEFAULT 0,
		created_ms INTEGER NOT NULL
	);
	CREATE TABLE devices (
		secret_hash  BLOB PRIMARY KEY,
		player_id    INTEGER NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		created_ms   INTEGER NOT NULL,
		last_seen_ms INTEGER NOT NULL
	);
	CREATE TABLE join_codes (
		code       TEXT PRIMARY KEY,
		uses_left  INTEGER NOT NULL,
		expires_ms INTEGER NOT NULL,
		admin      INTEGER NOT NULL DEFAULT 0,
		created_ms INTEGER NOT NULL
	);
	CREATE TABLE join_requests (
		id          INTEGER PRIMARY KEY,
		secret_hash BLOB NOT NULL UNIQUE,
		name        TEXT NOT NULL,
		color       INTEGER NOT NULL,
		status      TEXT NOT NULL CHECK (status IN ('pending', 'approved', 'denied')),
		ip          TEXT NOT NULL,
		created_ms  INTEGER NOT NULL
	);`,
	`CREATE TABLE worlds (
		id         INTEGER PRIMARY KEY,
		name       TEXT NOT NULL UNIQUE,
		seed       INTEGER NOT NULL,
		generator  INTEGER NOT NULL,
		radius     INTEGER NOT NULL,
		created_ms INTEGER NOT NULL
	);
	CREATE TABLE chunks (
		world_id INTEGER NOT NULL REFERENCES worlds(id) ON DELETE CASCADE,
		x        INTEGER NOT NULL,
		y        INTEGER NOT NULL,
		z        INTEGER NOT NULL,
		data     BLOB NOT NULL,
		PRIMARY KEY (world_id, x, y, z)
	) WITHOUT ROWID;`,
	`CREATE TABLE art (
		id         INTEGER PRIMARY KEY,
		kind       TEXT NOT NULL CHECK (kind IN ('skin', 'block', 'texture')),
		player_id  INTEGER REFERENCES players(id) ON DELETE SET NULL,
		name       TEXT NOT NULL,
		pixels     BLOB NOT NULL,
		approved   INTEGER NOT NULL DEFAULT 0,
		block_id   INTEGER UNIQUE,
		created_ms INTEGER NOT NULL
	);`,
	`ALTER TABLE players ADD COLUMN mode TEXT NOT NULL DEFAULT '' CHECK (mode IN ('', 'creative', 'survival'));`,
	`ALTER TABLE players ADD COLUMN inventory BLOB NOT NULL DEFAULT x'';`,
	`CREATE TABLE zones (
		id   INTEGER PRIMARY KEY,
		name TEXT NOT NULL UNIQUE COLLATE NOCASE,
		x0   INTEGER NOT NULL,
		z0   INTEGER NOT NULL,
		x1   INTEGER NOT NULL,
		z1   INTEGER NOT NULL,
		mode TEXT NOT NULL DEFAULT '' CHECK (mode IN ('', 'creative', 'survival'))
	);
	CREATE TABLE zone_builders (
		zone_id   INTEGER NOT NULL REFERENCES zones(id) ON DELETE CASCADE,
		player_id INTEGER NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		PRIMARY KEY (zone_id, player_id)
	) WITHOUT ROWID;`,
	`CREATE TABLE arenas (
		zone_id INTEGER PRIMARY KEY REFERENCES zones(id) ON DELETE CASCADE,
		setup   TEXT NOT NULL
	);`,
	`ALTER TABLE players ADD COLUMN placed INTEGER NOT NULL DEFAULT 0;
	ALTER TABLE players ADD COLUMN x REAL NOT NULL DEFAULT 0;
	ALTER TABLE players ADD COLUMN y REAL NOT NULL DEFAULT 0;
	ALTER TABLE players ADD COLUMN z REAL NOT NULL DEFAULT 0;
	ALTER TABLE players ADD COLUMN yaw REAL NOT NULL DEFAULT 0;
	ALTER TABLE players ADD COLUMN pitch REAL NOT NULL DEFAULT 0;
	ALTER TABLE players ADD COLUMN flying INTEGER NOT NULL DEFAULT 0;
	ALTER TABLE players ADD COLUMN health INTEGER NOT NULL DEFAULT 20;`,
	`CREATE TABLE scripts (
		x          INTEGER NOT NULL,
		y          INTEGER NOT NULL,
		z          INTEGER NOT NULL,
		source     TEXT NOT NULL,
		owner_id   INTEGER REFERENCES players(id) ON DELETE SET NULL,
		updated_ms INTEGER NOT NULL,
		PRIMARY KEY (x, y, z)
	) WITHOUT ROWID;`,
	`ALTER TABLE scripts ADD COLUMN npc TEXT NOT NULL DEFAULT '';
	CREATE TABLE memories (
		player_id INTEGER NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		key       TEXT NOT NULL,
		value     TEXT NOT NULL,
		PRIMARY KEY (player_id, key)
	) WITHOUT ROWID;
	CREATE TABLE quests (
		player_id INTEGER NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		title     TEXT NOT NULL,
		step      TEXT NOT NULL,
		done      INTEGER NOT NULL,
		place     INTEGER NOT NULL,
		PRIMARY KEY (player_id, title)
	) WITHOUT ROWID;`,
	`CREATE TABLE maps (
		zone_id  INTEGER PRIMARY KEY REFERENCES zones(id) ON DELETE CASCADE,
		data     BLOB NOT NULL,
		saved_ms INTEGER NOT NULL
	);
	ALTER TABLE players ADD COLUMN checkpoint INTEGER NOT NULL DEFAULT 0;
	ALTER TABLE players ADD COLUMN checkpoint_x INTEGER NOT NULL DEFAULT 0;
	ALTER TABLE players ADD COLUMN checkpoint_y INTEGER NOT NULL DEFAULT 0;
	ALTER TABLE players ADD COLUMN checkpoint_z INTEGER NOT NULL DEFAULT 0;`,
	`ALTER TABLE maps ADD COLUMN under BLOB;`,
	`CREATE TABLE agent_tokens (
		hash         BLOB PRIMARY KEY,
		player_id    INTEGER NOT NULL REFERENCES players(id) ON DELETE CASCADE,
		label        TEXT NOT NULL,
		created_ms   INTEGER NOT NULL,
		last_used_ms INTEGER NOT NULL DEFAULT 0,
		UNIQUE (player_id, label)
	);
	ALTER TABLE players ADD COLUMN agent_allowed INTEGER NOT NULL DEFAULT 0;`,
}

// Open opens (creating if needed) the database at path and migrates it to the latest schema.
func Open(path string) (*sql.DB, error) {
	dsn := "file:" + path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)" +
		"&_pragma=foreign_keys(1)&_pragma=synchronous(NORMAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // one writer keeps SQLite simple; load is tiny
	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	var version int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return err
	}
	for i := version; i < len(migrations); i++ {
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(migrations[i]); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration %d: %w", i+1, err)
		}
		if _, err := tx.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, i+1)); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
