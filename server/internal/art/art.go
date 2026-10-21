// Package art stores pixel art players make (skins, new blocks, repainted textures) and its approval.
package art

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"scabbage/internal/world"
)

// Kind is what a piece of art is for.
type Kind int

const (
	Skin    Kind = iota + 1 // a player's look
	Block                   // a new block
	Texture                 // a repainted built-in texture (admins only)
)

const (
	BlockSize      = 16
	SkinWidth      = 64
	SkinHeight     = 32
	MaxBlocks      = 64 // custom blocks the world can hold
	maxPendingEach = 3
	maxPending     = 20
)

var (
	ErrBadKind   = errors.New("unknown kind of art")
	ErrBadSize   = errors.New("the picture is the wrong size")
	ErrBadName   = errors.New("block names are 1-20 letters, digits, spaces, ' _ or -")
	ErrNameTaken = errors.New("there's already a block with that name")
	ErrTooMany   = errors.New("wait for an admin to look at what you've sent first")
	ErrFull      = errors.New("the world has all the new blocks it can hold")
	ErrNotAdmin  = errors.New("only admins can repaint blocks")
	ErrNotFound  = errors.New("art not found or already decided")
)

var kindNames = map[Kind]string{Skin: "skin", Block: "block", Texture: "texture"}

// Piece is one submitted picture. Pixels are RGBA, rows top-down.
type Piece struct {
	ID       int64
	Kind     Kind
	Name     string // the block's name, or the texture repainted; empty for skins
	Pixels   []byte // nil when a texture is reset to its original
	PlayerID int64
	Author   string
	Color    uint32
	Approved bool
	BlockID  world.Block // set once a block is approved
	Created  time.Time
}

// Transparent reports whether any pixel is see-through.
func (p Piece) Transparent() bool {
	for i := 3; i < len(p.Pixels); i += 4 {
		if p.Pixels[i] == 0 {
			return true
		}
	}
	return false
}

// Options configures a Service.
type Options struct {
	Now             func() time.Time
	OnPendingChange func()                 // called after the waiting list changes; must not block
	Reserved        func(name string) bool // built-in block names, matched in lower case
}

// Service keeps art in the database.
type Service struct {
	db   *sql.DB
	opts Options
}

// New returns a Service using db, which must already be migrated.
func New(db *sql.DB, opts Options) *Service {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.OnPendingChange == nil {
		opts.OnPendingChange = func() {}
	}
	if opts.Reserved == nil {
		opts.Reserved = func(string) bool { return false }
	}
	return &Service{db: db, opts: opts}
}

// Submit stores art from a player. An admin's is approved at once; anyone else's waits.
func (s *Service) Submit(playerID int64, admin bool, k Kind, name string, px []byte) (Piece, error) {
	p, err := s.validate(admin, k, name, px)
	if err != nil {
		return Piece{}, err
	}
	p.PlayerID, p.Created = playerID, s.opts.Now()
	tx, err := s.db.Begin()
	if err != nil {
		return Piece{}, err
	}
	defer func() { _ = tx.Rollback() }()
	if p, err = s.store(tx, p, admin); err != nil {
		return Piece{}, err
	}
	if err := tx.Commit(); err != nil {
		return Piece{}, err
	}
	if !p.Approved {
		s.opts.OnPendingChange()
	}
	return p, nil
}

// reset drops a player's skin or a repainted texture, which needs no approval.
func reset(tx *sql.Tx, p Piece) (Piece, error) {
	var err error
	if p.Kind == Skin {
		_, err = tx.Exec(`DELETE FROM art WHERE kind = 'skin' AND approved = 1 AND player_id = ?`, p.PlayerID)
	} else {
		_, err = tx.Exec(`DELETE FROM art WHERE kind = 'texture' AND name = ?`, p.Name)
	}
	p.Approved = true
	return p, err
}

func (s *Service) validate(admin bool, k Kind, name string, px []byte) (Piece, error) {
	p := Piece{Kind: k}
	var err error
	switch k {
	case Skin:
		if len(px) == 0 {
			return p, nil // back to the default look
		}
	case Block:
		if p.Name, err = cleanBlockName(name); err != nil {
			return Piece{}, err
		}
	case Texture:
		if !admin {
			return Piece{}, ErrNotAdmin
		}
		if !validTextureName(name) {
			return Piece{}, ErrBadName
		}
		p.Name = name
		if len(px) == 0 {
			return p, nil // a reset
		}
	default:
		return Piece{}, ErrBadKind
	}
	p.Pixels, err = cleanPixels(k, px)
	return p, err
}

// store inserts a validated piece, approving it straight away for admins.
func (s *Service) store(tx *sql.Tx, p Piece, admin bool) (Piece, error) {
	if p.Pixels == nil {
		return reset(tx, p)
	}
	if err := s.checkRoom(tx, p, admin); err != nil {
		return Piece{}, err
	}
	res, err := tx.Exec(`INSERT INTO art (kind, player_id, name, pixels, created_ms) VALUES (?, ?, ?, ?, ?)`,
		kindNames[p.Kind], p.PlayerID, p.Name, p.Pixels, p.Created.UnixMilli())
	if err != nil {
		return Piece{}, err
	}
	if p.ID, err = res.LastInsertId(); err != nil {
		return Piece{}, err
	}
	if err := tx.QueryRow(`SELECT name, color FROM players WHERE id = ?`, p.PlayerID).Scan(&p.Author, &p.Color); err != nil {
		return Piece{}, err
	}
	if admin {
		return approve(tx, p)
	}
	return p, nil
}

// checkRoom refuses names in use and players with too much already waiting.
func (s *Service) checkRoom(tx *sql.Tx, p Piece, admin bool) error {
	if p.Kind == Block {
		if s.opts.Reserved(strings.ToLower(p.Name)) {
			return ErrNameTaken
		}
		var n int
		if err := tx.QueryRow(`SELECT count(*) FROM art WHERE kind = 'block' AND name = ? COLLATE NOCASE`, p.Name).Scan(&n); err != nil {
			return err
		}
		if n > 0 {
			return ErrNameTaken
		}
		if _, err := nextBlockID(tx); err != nil {
			return err // full already, so don't keep anyone waiting
		}
	}
	if admin {
		return nil
	}
	var mine, all int
	err := tx.QueryRow(`SELECT count(*) FILTER (WHERE player_id = ?), count(*) FROM art WHERE approved = 0`, p.PlayerID).
		Scan(&mine, &all)
	if err != nil {
		return err
	}
	if mine >= maxPendingEach || all >= maxPending {
		return ErrTooMany
	}
	return nil
}

// Wipe deletes every skin, custom block and repainted texture, approved or waiting.
func (s *Service) Wipe() error {
	if _, err := s.db.Exec(`DELETE FROM art`); err != nil {
		return err
	}
	s.opts.OnPendingChange()
	return nil
}

// Decide approves or drops waiting art, returning it as it now stands.
func (s *Service) Decide(id int64, ok bool) (Piece, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return Piece{}, err
	}
	defer func() { _ = tx.Rollback() }()
	p, err := scanPiece(tx.QueryRow(selectPieces+` WHERE a.id = ? AND a.approved = 0`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Piece{}, ErrNotFound
	}
	if err != nil {
		return Piece{}, err
	}
	if ok {
		p, err = approve(tx, p)
	} else {
		_, err = tx.Exec(`DELETE FROM art WHERE id = ?`, id)
	}
	if err != nil {
		return Piece{}, err
	}
	if err := tx.Commit(); err != nil {
		return Piece{}, err
	}
	s.opts.OnPendingChange()
	return p, nil
}

// approve makes a piece live: skins and textures replace older ones, blocks get the next ID.
func approve(tx *sql.Tx, p Piece) (Piece, error) {
	var err error
	switch p.Kind {
	case Skin:
		_, err = tx.Exec(`DELETE FROM art WHERE kind = 'skin' AND approved = 1 AND player_id = ?`, p.PlayerID)
	case Texture:
		_, err = tx.Exec(`DELETE FROM art WHERE kind = 'texture' AND approved = 1 AND name = ?`, p.Name)
	case Block:
		p.BlockID, err = nextBlockID(tx)
	}
	if err != nil {
		return Piece{}, err
	}
	p.Approved = true
	_, err = tx.Exec(`UPDATE art SET approved = 1, block_id = ? WHERE id = ?`, blockIDValue(p), p.ID)
	return p, err
}

func nextBlockID(tx *sql.Tx) (world.Block, error) {
	var n int
	if err := tx.QueryRow(`SELECT count(*) FROM art WHERE block_id IS NOT NULL`).Scan(&n); err != nil {
		return 0, err
	}
	if n >= MaxBlocks {
		return 0, ErrFull
	}
	return world.FirstCustom + world.Block(n), nil
}

func blockIDValue(p Piece) any {
	if p.Kind != Block {
		return nil
	}
	return int64(p.BlockID)
}
