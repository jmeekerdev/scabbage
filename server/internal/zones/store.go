package zones

import "database/sql"

// save writes a zone and its builders, then reloads every zone.
func (s *Service) save(z Zone) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var id int64
	err = tx.QueryRow(`INSERT INTO zones (name, x0, z0, x1, z1, mode) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (name) DO UPDATE SET x0 = excluded.x0, z0 = excluded.z0, x1 = excluded.x1,
		z1 = excluded.z1, mode = excluded.mode RETURNING id`,
		z.Name, z.X0, z.Z0, z.X1, z.Z1, z.Mode).Scan(&id)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM zone_builders WHERE zone_id = ?`, id); err != nil {
		return err
	}
	for _, p := range z.Builders {
		if _, err := tx.Exec(`INSERT INTO zone_builders (zone_id, player_id) VALUES (?, ?)`, id, p); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return s.load()
}

func (s *Service) load() error {
	zones, err := readZones(s.db)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.zones = zones
	return nil
}

func readZones(db *sql.DB) ([]Zone, error) {
	rows, err := db.Query(`SELECT z.name, z.x0, z.z0, z.x1, z.z1, z.mode, b.player_id
		FROM zones z LEFT JOIN zone_builders b ON b.zone_id = z.id ORDER BY z.id, b.player_id`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Zone
	for rows.Next() {
		var z Zone
		var builder sql.NullInt64
		if err := rows.Scan(&z.Name, &z.X0, &z.Z0, &z.X1, &z.Z1, &z.Mode, &builder); err != nil {
			return nil, err
		}
		if n := len(out); n == 0 || out[n-1].Name != z.Name {
			out = append(out, z)
		}
		if builder.Valid {
			last := &out[len(out)-1]
			last.Builders = append(last.Builders, builder.Int64)
		}
	}
	return out, rows.Err()
}
