package arenas

import (
	"database/sql"
	"encoding/json"
	"errors"

	"scabbage/internal/zones"
)

// Service saves arenas beside their zones; removing a zone removes its arena.
type Service struct{ db *sql.DB }

func New(db *sql.DB) *Service { return &Service{db: db} }

// Get finds the arena set up in a zone.
func (s *Service) Get(zone string) (Arena, bool, error) {
	var a Arena
	var setup string
	err := s.db.QueryRow(`SELECT z.name, a.setup FROM arenas a JOIN zones z ON z.id = a.zone_id
		WHERE z.name = ?`, zone).Scan(&a.Zone, &setup)
	if errors.Is(err, sql.ErrNoRows) {
		return a, false, nil
	}
	if err != nil {
		return a, false, err
	}
	return a, true, json.Unmarshal([]byte(setup), &a)
}

// All lists every arena by zone.
func (s *Service) All() ([]Arena, error) {
	rows, err := s.db.Query(`SELECT z.name, a.setup FROM arenas a JOIN zones z ON z.id = a.zone_id ORDER BY z.id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Arena
	for rows.Next() {
		var a Arena
		var setup string
		if err := rows.Scan(&a.Zone, &setup); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(setup), &a); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// Put saves an arena in a saved zone.
func (s *Service) Put(zone string, a Arena) error {
	setup, err := json.Marshal(a)
	if err != nil {
		return err
	}
	res, err := s.db.Exec(`INSERT INTO arenas (zone_id, setup) SELECT id, ? FROM zones WHERE name = ?
		ON CONFLICT (zone_id) DO UPDATE SET setup = excluded.setup`, setup, zone)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return zones.ErrNoZone
	}
	return nil
}

func (s *Service) Remove(zone string) error {
	res, err := s.db.Exec(`DELETE FROM arenas WHERE zone_id = (SELECT id FROM zones WHERE name = ?)`, zone)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNoArena
	}
	return nil
}
