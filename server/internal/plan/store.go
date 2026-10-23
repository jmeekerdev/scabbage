package plan

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// MaxPlans is how many plans each player may keep.
const MaxPlans = 50

var (
	ErrNotFound   = errors.New("there's no plan by that name")
	ErrTaken      = errors.New("you already have a plan by that name")
	ErrSharedName = errors.New("another shared plan has that name; rename yours first")
	ErrFull       = fmt.Errorf("you have %d plans; delete one first", MaxPlans)
)

// Stored is a saved plan with its owner's name and its sizes. Listings leave out the palette and layers.
type Stored struct {
	Plan
	Owner          string
	Shared         bool
	W, D, H, Count int
	Created        time.Time
}

// body is the part of a plan kept as JSON.
type body struct {
	Palette map[string]string `json:"palette"`
	Layers  [][]string        `json:"layers"`
}

// Service keeps plans in the database.
type Service struct {
	db  *sql.DB
	now func() time.Time
}

// New keeps plans in db.
func New(db *sql.DB, now func() time.Time) *Service { return &Service{db: db, now: now} }

// Save stores a checked plan as the owner's, replacing their plan of the same name but keeping it shared.
func (s *Service) Save(owner int64, p Plan) error {
	var n int
	err := s.db.QueryRow(`SELECT count(*) FROM plans WHERE player_id = ? AND name != ?`, owner, p.Name).Scan(&n)
	if err != nil {
		return err
	}
	if n >= MaxPlans {
		return ErrFull
	}
	js, err := json.Marshal(body{p.Palette, p.Layers})
	if err != nil {
		return err
	}
	w, d, h := p.Size()
	_, err = s.db.Exec(`INSERT INTO plans (player_id, name, about, w, d, h, cells, body, created_ms)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (player_id, name) DO UPDATE SET about = excluded.about, w = excluded.w, d = excluded.d,
		h = excluded.h, cells = excluded.cells, body = excluded.body, created_ms = excluded.created_ms`,
		owner, p.Name, p.About, w, d, h, p.Cells(), string(js), s.now().UnixMilli())
	return err
}

const selectPlans = `SELECT p.name, p.about, p.w, p.d, p.h, p.cells, p.shared, p.created_ms, o.name, %s
	FROM plans p JOIN players o ON o.id = p.player_id `

// Get finds the owner's plan by name, or else a shared one.
func (s *Service) Get(owner int64, name string) (Stored, error) {
	row := s.db.QueryRow(fmt.Sprintf(selectPlans, "p.body")+
		`WHERE p.name = ? AND (p.player_id = ? OR p.shared = 1) ORDER BY p.player_id != ? LIMIT 1`,
		strings.ToLower(name), owner, owner)
	var js string
	st, err := scan(row, &js)
	if errors.Is(err, sql.ErrNoRows) {
		return st, ErrNotFound
	}
	if err != nil {
		return st, err
	}
	var b body
	err = json.Unmarshal([]byte(js), &b)
	st.Palette, st.Layers = b.Palette, b.Layers
	return st, err
}

// List gives the owner's plans, then everyone else's shared ones, each by name, without their layers.
func (s *Service) List(owner int64) ([]Stored, error) {
	rows, err := s.db.Query(fmt.Sprintf(selectPlans, "''")+
		`WHERE p.player_id = ? OR p.shared = 1 ORDER BY p.player_id != ?, p.name`, owner, owner)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Stored
	for rows.Next() {
		var ignored string
		st, err := scan(rows, &ignored)
		if err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

type scanner interface{ Scan(dest ...any) error }

func scan(row scanner, js *string) (Stored, error) {
	var st Stored
	var created int64
	err := row.Scan(&st.Name, &st.About, &st.W, &st.D, &st.H, &st.Count, &st.Shared, &created, &st.Owner, js)
	st.Created = time.UnixMilli(created)
	return st, err
}

// Share shares or unshares the owner's plan. A shared name must be unique among shared plans.
func (s *Service) Share(owner int64, name string, shared bool) error {
	name = strings.ToLower(name)
	if shared && s.sharedByOther(owner, name) {
		return ErrSharedName
	}
	return s.changeOwn(`UPDATE plans SET shared = ? WHERE player_id = ? AND name = ?`, shared, owner, name)
}

// Rename renames the owner's plan, keeping a shared one's name unique among shared plans.
func (s *Service) Rename(owner int64, from, to string) error {
	from, to = strings.ToLower(from), strings.ToLower(to)
	if err := CheckName(to); err != nil {
		return err
	}
	var shared bool
	err := s.db.QueryRow(`SELECT shared FROM plans WHERE player_id = ? AND name = ?`, owner, from).Scan(&shared)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if s.owns(owner, to) {
		return ErrTaken
	}
	if shared && s.sharedByOther(owner, to) {
		return ErrSharedName
	}
	return s.changeOwn(`UPDATE plans SET name = ? WHERE player_id = ? AND name = ?`, to, owner, from)
}

// Delete removes the owner's plan.
func (s *Service) Delete(owner int64, name string) error {
	return s.changeOwn(`DELETE FROM plans WHERE player_id = ? AND name = ?`, owner, strings.ToLower(name))
}

// DeleteShared removes a shared plan, whoever owns it; for admins.
func (s *Service) DeleteShared(name string) error {
	return s.changeOwn(`DELETE FROM plans WHERE shared = 1 AND name = ?`, strings.ToLower(name))
}

func (s *Service) owns(owner int64, name string) bool {
	var one int
	return s.db.QueryRow(`SELECT 1 FROM plans WHERE player_id = ? AND name = ?`, owner, name).Scan(&one) == nil
}

func (s *Service) sharedByOther(owner int64, name string) bool {
	var one int
	return s.db.QueryRow(`SELECT 1 FROM plans WHERE shared = 1 AND name = ? AND player_id != ?`, name, owner).Scan(&one) == nil
}

// changeOwn runs an update or delete that should touch one plan, or says there's none.
func (s *Service) changeOwn(query string, args ...any) error {
	res, err := s.db.Exec(query, args...)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
