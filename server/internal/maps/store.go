package maps

import (
	"database/sql"
	"errors"
	"time"

	"scabbage/internal/zones"
)

// ErrNoMap means the zone has no saved map.
var ErrNoMap = errors.New("no map is saved for that zone")

// Service saves a map beside each zone; removing the zone removes its map.
type Service struct{ db *sql.DB }

func New(db *sql.DB) *Service { return &Service{db: db} }

// Put saves m as the map of a saved zone.
func (s *Service) Put(zone string, m Map) error {
	data, err := Encode(m)
	if err != nil {
		return err
	}
	res, err := s.db.Exec(`INSERT INTO maps (zone_id, data, saved_ms) SELECT id, ?, ? FROM zones WHERE name = ?
		ON CONFLICT (zone_id) DO UPDATE SET data = excluded.data, saved_ms = excluded.saved_ms`,
		data, time.Now().UnixMilli(), zone)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return zones.ErrNoZone
	}
	return nil
}

// Get finds a zone's map.
func (s *Service) Get(zone string) (Map, bool, error) {
	data, ok, err := s.File(zone)
	if !ok || err != nil {
		return Map{}, false, err
	}
	m, _, err := Decode(data)
	return m, err == nil, err
}

// File is a zone's map as a .scabmap file.
func (s *Service) File(zone string) ([]byte, bool, error) {
	var data []byte
	err := s.db.QueryRow(`SELECT m.data FROM maps m JOIN zones z ON z.id = m.zone_id WHERE z.name = ?`, zone).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	return data, err == nil, err
}

// PutUnder keeps the land a map was laid over, so removing the map can put it back.
// The zone must already have a map.
func (s *Service) PutUnder(zone string, land Map) error {
	data, err := Encode(land)
	if err != nil {
		return err
	}
	res, err := s.db.Exec(`UPDATE maps SET under = ? WHERE zone_id = (SELECT id FROM zones WHERE name = ?)`, data, zone)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return zones.ErrNoZone
	}
	return nil
}

// Under is the land a zone's map was laid over, if it was kept.
func (s *Service) Under(zone string) (Map, bool, error) {
	var data []byte
	err := s.db.QueryRow(`SELECT m.under FROM maps m JOIN zones z ON z.id = m.zone_id WHERE z.name = ?`, zone).Scan(&data)
	if errors.Is(err, sql.ErrNoRows) || err == nil && data == nil {
		return Map{}, false, nil
	}
	if err != nil {
		return Map{}, false, err
	}
	m, _, err := Decode(data)
	return m, err == nil, err
}

// Names lists the zones with maps.
func (s *Service) Names() ([]string, error) {
	rows, err := s.db.Query(`SELECT z.name FROM maps m JOIN zones z ON z.id = m.zone_id ORDER BY z.name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

// Remove forgets a zone's map but keeps the zone.
func (s *Service) Remove(zone string) error {
	res, err := s.db.Exec(`DELETE FROM maps WHERE zone_id = (SELECT id FROM zones WHERE name = ?)`, zone)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNoMap
	}
	return nil
}
