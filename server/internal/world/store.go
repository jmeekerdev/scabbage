package world

import (
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"errors"
	"time"
)

// Border radii in blocks.
const (
	DefaultRadius int32 = 512 // a 1024×1024 world
	Unbounded     int32 = -1
)

// Meta is a world's stored identity. Generator and Seed never change after creation.
type Meta struct {
	ID        int64
	Name      string
	Seed      int64
	Generator uint32
	Radius    int32 // half the border width in blocks, or Unbounded
}

// Defaults apply only when a world is first created. Zero values pick a random seed and DefaultRadius.
type Defaults struct {
	Seed    int64
	Radius  int32
	Upgrade bool // move a version 1 world to version 2: the same land, with dungeons where no one built
}

// Store reads and writes worlds and chunks in SQLite.
type Store struct{ db *sql.DB }

// NewStore wraps a database opened by store.Open.
func NewStore(db *sql.DB) *Store { return &Store{db: db} }

// world loads the named world, creating it from d if it doesn't exist.
func (s *Store) world(name string, d Defaults) (Meta, error) {
	seed, radius := d.Seed, d.Radius
	if seed == 0 {
		seed = randomSeed()
	}
	switch {
	case radius == 0:
		radius = DefaultRadius
	case radius > 0:
		radius = (radius + Size - 1) / Size * Size // whole chunks
	}
	_, err := s.db.Exec(`INSERT OR IGNORE INTO worlds (name, seed, generator, radius, created_ms)
		VALUES (?, ?, ?, ?, ?)`, name, seed, LatestGenerator, radius, time.Now().UnixMilli())
	if err != nil {
		return Meta{}, err
	}
	if d.Upgrade {
		if _, err := s.db.Exec(`UPDATE worlds SET generator = 2 WHERE name = ? AND generator = 1`, name); err != nil {
			return Meta{}, err
		}
	}
	m := Meta{Name: name}
	err = s.db.QueryRow(`SELECT id, seed, generator, radius FROM worlds WHERE name = ?`, name).
		Scan(&m.ID, &m.Seed, &m.Generator, &m.Radius)
	return m, err
}

// loadChunk returns a stored chunk's encoded data, or nil if it was never saved.
func (s *Store) loadChunk(world int64, p ChunkPos) ([]byte, error) {
	var data []byte
	err := s.db.QueryRow(`SELECT data FROM chunks WHERE world_id = ? AND x = ? AND y = ? AND z = ?`,
		world, p.X, p.Y, p.Z).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return data, err
}

// saveChunks writes encoded chunks in one transaction.
func (s *Store) saveChunks(world int64, chunks map[ChunkPos][]byte) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	stmt, err := tx.Prepare(`INSERT OR REPLACE INTO chunks (world_id, x, y, z, data) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	for p, data := range chunks {
		if _, err := stmt.Exec(world, p.X, p.Y, p.Z, data); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// deleteChunks forgets stored chunks, so they generate afresh.
func (s *Store) deleteChunks(world int64, ps []ChunkPos) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, p := range ps {
		if _, err := tx.Exec(`DELETE FROM chunks WHERE world_id = ? AND x = ? AND y = ? AND z = ?`, world, p.X, p.Y, p.Z); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func randomSeed() int64 {
	var b [8]byte
	_, _ = rand.Read(b[:])
	if s := int64(binary.LittleEndian.Uint64(b[:]) >> 1); s != 0 {
		return s
	}
	return 1
}
