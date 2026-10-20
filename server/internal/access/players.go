package access

import (
	"database/sql"
	"errors"
)

var (
	ErrNoPlayer = errors.New("no such player")
	ErrBadMode  = errors.New("unknown game mode")
)

// Mode is a player's game mode; the default follows the server's setting.
type Mode string

const (
	ModeDefault  Mode = ""
	ModeCreative Mode = "creative"
	ModeSurvival Mode = "survival"
)

func (m Mode) Valid() bool { return m == ModeDefault || m == ModeCreative || m == ModeSurvival }

// Players lists every approved player, sorted by name ignoring case.
func (s *Service) Players() ([]Player, error) {
	rows, err := s.db.Query(`SELECT id, name, color, admin, mode FROM players ORDER BY name COLLATE NOCASE`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Player
	for rows.Next() {
		var p Player
		if err := rows.Scan(&p.ID, &p.Name, &p.Color, &p.Admin, &p.Mode); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// SetAdmin grants or removes a player's admin rights and returns the updated player.
func (s *Service) SetAdmin(id int64, admin bool) (Player, error) {
	var p Player
	err := s.db.QueryRow(`UPDATE players SET admin = ? WHERE id = ? RETURNING id, name, color, admin, mode`, admin, id).
		Scan(&p.ID, &p.Name, &p.Color, &p.Admin, &p.Mode)
	if errors.Is(err, sql.ErrNoRows) {
		return Player{}, ErrNoPlayer
	}
	return p, err
}

// SetMode changes a player's game mode and returns the updated player.
func (s *Service) SetMode(id int64, m Mode) (Player, error) {
	if !m.Valid() {
		return Player{}, ErrBadMode
	}
	var p Player
	err := s.db.QueryRow(`UPDATE players SET mode = ? WHERE id = ? RETURNING id, name, color, admin, mode`, m, id).
		Scan(&p.ID, &p.Name, &p.Color, &p.Admin, &p.Mode)
	if errors.Is(err, sql.ErrNoRows) {
		return Player{}, ErrNoPlayer
	}
	return p, err
}

// Save is where a player was and what they had when last saved; Placed is false until then.
type Save struct {
	Placed     bool
	X, Y, Z    float64
	Yaw, Pitch float32
	Flying     bool
	Health     int
	Inventory  []byte // as inventory.Encode stored it
}

func (s *Service) Saved(id int64) (Save, error) {
	var v Save
	err := s.db.QueryRow(`SELECT placed, x, y, z, yaw, pitch, flying, health, inventory FROM players WHERE id = ?`, id).
		Scan(&v.Placed, &v.X, &v.Y, &v.Z, &v.Yaw, &v.Pitch, &v.Flying, &v.Health, &v.Inventory)
	if errors.Is(err, sql.ErrNoRows) {
		return Save{}, ErrNoPlayer
	}
	if len(v.Inventory) == 0 {
		v.Inventory = nil
	}
	return v, err
}

func (s *Service) Store(id int64, v Save) error {
	if v.Inventory == nil {
		v.Inventory = []byte{} // an empty blob, not NULL
	}
	_, err := s.db.Exec(`UPDATE players SET placed = ?, x = ?, y = ?, z = ?, yaw = ?, pitch = ?, flying = ?, health = ?, inventory = ?
		WHERE id = ?`, v.Placed, v.X, v.Y, v.Z, v.Yaw, v.Pitch, v.Flying, v.Health, v.Inventory, id)
	return err
}
