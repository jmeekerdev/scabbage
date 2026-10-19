package access

import (
	"database/sql"
	"errors"
)

var ErrNoPlayer = errors.New("no such player")

// Players lists every approved player, sorted by name ignoring case.
func (s *Service) Players() ([]Player, error) {
	rows, err := s.db.Query(`SELECT id, name, color, admin FROM players ORDER BY name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Player
	for rows.Next() {
		var p Player
		if err := rows.Scan(&p.ID, &p.Name, &p.Color, &p.Admin); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// SetAdmin grants or removes a player's admin rights and returns the updated player.
func (s *Service) SetAdmin(id int64, admin bool) (Player, error) {
	var p Player
	err := s.db.QueryRow(`UPDATE players SET admin = ? WHERE id = ? RETURNING id, name, color, admin`, admin, id).
		Scan(&p.ID, &p.Name, &p.Color, &p.Admin)
	if errors.Is(err, sql.ErrNoRows) {
		return Player{}, ErrNoPlayer
	}
	return p, err
}
