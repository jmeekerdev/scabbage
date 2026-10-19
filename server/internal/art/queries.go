package art

import (
	"database/sql"
	"errors"
	"time"

	"scabbage/internal/world"
)

const selectPieces = `SELECT a.id, a.kind, a.name, a.pixels, a.approved, a.block_id, a.created_ms,
	coalesce(a.player_id, 0), coalesce(p.name, ''), coalesce(p.color, 0)
	FROM art a LEFT JOIN players p ON p.id = a.player_id`

type scanner interface{ Scan(dest ...any) error }

func scanPiece(row scanner) (Piece, error) {
	var p Piece
	var kind string
	var blockID sql.NullInt64
	var created int64
	err := row.Scan(&p.ID, &kind, &p.Name, &p.Pixels, &p.Approved, &blockID, &created, &p.PlayerID, &p.Author, &p.Color)
	for k, n := range kindNames {
		if n == kind {
			p.Kind = k
		}
	}
	p.BlockID = world.Block(blockID.Int64)
	p.Created = time.UnixMilli(created)
	return p, err
}

func (s *Service) query(where string, args ...any) ([]Piece, error) {
	rows, err := s.db.Query(selectPieces+" WHERE "+where, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Piece
	for rows.Next() {
		p, err := scanPiece(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Pending lists art waiting for an admin, oldest first.
func (s *Service) Pending() ([]Piece, error) { return s.query(`a.approved = 0 ORDER BY a.id`) }

// Blocks lists approved custom blocks in ID order.
func (s *Service) Blocks() ([]Piece, error) {
	return s.query(`a.kind = 'block' AND a.approved = 1 ORDER BY a.block_id`)
}

// Textures lists the built-in textures admins have repainted.
func (s *Service) Textures() ([]Piece, error) {
	return s.query(`a.kind = 'texture' AND a.approved = 1 ORDER BY a.name`)
}

// Skin returns a player's approved skin, or nil if they have none.
func (s *Service) Skin(playerID int64) ([]byte, error) {
	var px []byte
	err := s.db.QueryRow(`SELECT pixels FROM art WHERE kind = 'skin' AND approved = 1 AND player_id = ?`, playerID).Scan(&px)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return px, err
}
