// Package access decides who may play: device secrets, join codes, and admin approval of join requests.
package access

import (
	"crypto/sha256"
	"database/sql"
	"errors"
	"time"
)

const (
	// SecretBytes is the exact length of a device secret.
	SecretBytes = 32
	maxPending  = 10
	pendingTTL  = 24 * time.Hour
)

var (
	ErrBadSecret     = errors.New("device secret must be 32 bytes")
	ErrUnknownDevice = errors.New("device is not registered")
	ErrDenied        = errors.New("join request was denied")
	ErrBadCode       = errors.New("join code is invalid or expired")
	ErrBadName       = errors.New("name must be 1-16 letters, digits, spaces, _ or -")
	ErrRateLimited   = errors.New("too many attempts, try again later")
	ErrNotFound      = errors.New("join request not found or already decided")
)

// Player is an approved account.
type Player struct {
	ID    int64
	Name  string
	Color uint32
	Admin bool
	Mode  Mode
}

// Request is a pending join request shown to admins.
type Request struct {
	ID        int64
	Name      string
	Color     uint32
	Returning bool // the name already belongs to a player; approving adds this device to them
	Created   time.Time
}

// Join is what a new device submits with a join code.
type Join struct {
	Code  string
	Name  string
	Color uint32
}

// Status says whether a device may play now or is waiting for an admin.
type Status int

const (
	StatusApproved Status = iota + 1
	StatusPending
)

// Result is the outcome of checking or submitting a device.
type Result struct {
	Status    Status
	Player    Player // set when approved
	RequestID int64  // set when pending
}

// Options configures a Service.
type Options struct {
	Now             func() time.Time
	OnPendingChange func() // called after the pending list changes; must not block
}

// Service implements the join and approval flow on top of the store.
type Service struct {
	db      *sql.DB
	opts    Options
	limiter *limiter
	waiters *waiters
}

// New returns a Service using db, which must already be migrated.
func New(db *sql.DB, opts Options) *Service {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.OnPendingChange == nil {
		opts.OnPendingChange = func() {}
	}
	return &Service{db: db, opts: opts, limiter: newLimiter(), waiters: newWaiters()}
}

func hashSecret(secret []byte) ([]byte, error) {
	if len(secret) != SecretBytes {
		return nil, ErrBadSecret
	}
	h := sha256.Sum256(secret)
	return h[:], nil
}

func (s *Service) nowMs() int64 { return s.opts.Now().UnixMilli() }

func (s *Service) pendingCutoffMs() int64 { return s.opts.Now().Add(-pendingTTL).UnixMilli() }

// Check reports whether the device holding secret is approved or still pending.
func (s *Service) Check(secret []byte) (Result, error) {
	hash, err := hashSecret(secret)
	if err != nil {
		return Result{}, err
	}
	p, err := s.deviceOwner(hash)
	if err == nil {
		_, err = s.db.Exec(`UPDATE devices SET last_seen_ms = ? WHERE secret_hash = ?`, s.nowMs(), hash)
		return Result{Status: StatusApproved, Player: p}, err
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return Result{}, err
	}
	return s.requestStatus(hash)
}

func (s *Service) deviceOwner(hash []byte) (Player, error) {
	var p Player
	err := s.db.QueryRow(`SELECT p.id, p.name, p.color, p.admin, p.mode FROM devices d
		JOIN players p ON p.id = d.player_id WHERE d.secret_hash = ?`, hash).
		Scan(&p.ID, &p.Name, &p.Color, &p.Admin, &p.Mode)
	return p, err
}

func (s *Service) requestStatus(hash []byte) (Result, error) {
	var id, created int64
	var status string
	err := s.db.QueryRow(`SELECT id, status, created_ms FROM join_requests WHERE secret_hash = ?`, hash).
		Scan(&id, &status, &created)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return Result{}, ErrUnknownDevice
	case err != nil:
		return Result{}, err
	case status == "denied":
		return Result{}, ErrDenied
	case status == "pending" && created > s.pendingCutoffMs():
		return Result{Status: StatusPending, RequestID: id}, nil
	default: // expired, or approved but the device was since removed
		return Result{}, ErrUnknownDevice
	}
}

// RequestJoin submits a join request for a new device. Admin invite codes are approved immediately.
func (s *Service) RequestJoin(secret []byte, j Join, ip string) (Result, error) {
	hash, err := hashSecret(secret)
	if err != nil {
		return Result{}, err
	}
	name, err := cleanName(j.Name)
	if err != nil {
		return Result{}, err
	}
	if s.limiter.blocked(ip, s.opts.Now()) {
		return Result{}, ErrRateLimited
	}
	if r, err := s.Check(secret); !errors.Is(err, ErrUnknownDevice) {
		return r, err
	}

	r, err := s.submit(hash, name, j, ip)
	if errors.Is(err, ErrBadCode) {
		s.limiter.fail(ip, s.opts.Now())
	}
	if err == nil && r.Status == StatusPending {
		s.opts.OnPendingChange()
	}
	return r, err
}

func (s *Service) submit(hash []byte, name string, j Join, ip string) (Result, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = tx.Rollback() }()

	admin, err := s.useCode(tx, normalizeCode(j.Code))
	if err != nil {
		return Result{}, err
	}
	if _, err := tx.Exec(`DELETE FROM join_requests WHERE secret_hash = ?`, hash); err != nil {
		return Result{}, err
	}
	var r Result
	if admin {
		r, err = s.approveAdmin(tx, hash, name, j.Color)
	} else {
		r, err = s.insertPending(tx, hash, name, j.Color, ip)
	}
	if err != nil {
		return Result{}, err
	}
	return r, tx.Commit()
}

func (s *Service) approveAdmin(tx *sql.Tx, hash []byte, name string, color uint32) (Result, error) {
	p, err := s.playerForName(tx, name, color)
	if err != nil {
		return Result{}, err
	}
	if _, err := tx.Exec(`UPDATE players SET admin = 1 WHERE id = ?`, p.ID); err != nil {
		return Result{}, err
	}
	p.Admin = true
	return Result{Status: StatusApproved, Player: p}, s.addDevice(tx, hash, p.ID)
}

func (s *Service) insertPending(tx *sql.Tx, hash []byte, name string, color uint32, ip string) (Result, error) {
	var n int
	if err := tx.QueryRow(`SELECT count(*) FROM join_requests WHERE status = 'pending' AND created_ms > ?`,
		s.pendingCutoffMs()).Scan(&n); err != nil {
		return Result{}, err
	}
	if n >= maxPending {
		return Result{}, ErrRateLimited
	}
	res, err := tx.Exec(`INSERT INTO join_requests (secret_hash, name, color, status, ip, created_ms)
		VALUES (?, ?, ?, 'pending', ?, ?)`, hash, name, color, ip, s.nowMs())
	if err != nil {
		return Result{}, err
	}
	id, err := res.LastInsertId()
	return Result{Status: StatusPending, RequestID: id}, err
}

// Decide approves or denies a pending request and wakes anyone awaiting it.
func (s *Service) Decide(id int64, approve bool) error {
	if err := s.decide(id, approve); err != nil {
		return err
	}
	s.waiters.deliver(id, approve)
	s.opts.OnPendingChange()
	return nil
}

func (s *Service) decide(id int64, approve bool) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var hash []byte
	var name string
	var color uint32
	err = tx.QueryRow(`SELECT secret_hash, name, color FROM join_requests
		WHERE id = ? AND status = 'pending' AND created_ms > ?`, id, s.pendingCutoffMs()).Scan(&hash, &name, &color)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	status := "denied"
	if approve {
		status = "approved"
		p, err := s.playerForName(tx, name, color)
		if err != nil {
			return err
		}
		if err := s.addDevice(tx, hash, p.ID); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`UPDATE join_requests SET status = ? WHERE id = ?`, status, id); err != nil {
		return err
	}
	return tx.Commit()
}

// playerForName returns the player with name (any case), creating them if new.
func (s *Service) playerForName(tx *sql.Tx, name string, color uint32) (Player, error) {
	var p Player
	err := tx.QueryRow(`SELECT id, name, color, admin, mode FROM players WHERE name = ?`, name).
		Scan(&p.ID, &p.Name, &p.Color, &p.Admin, &p.Mode)
	if !errors.Is(err, sql.ErrNoRows) {
		return p, err
	}
	res, err := tx.Exec(`INSERT INTO players (name, color, admin, created_ms) VALUES (?, ?, 0, ?)`,
		name, color, s.nowMs())
	if err != nil {
		return Player{}, err
	}
	p = Player{Name: name, Color: color}
	p.ID, err = res.LastInsertId()
	return p, err
}

func (s *Service) addDevice(tx *sql.Tx, hash []byte, playerID int64) error {
	now := s.nowMs()
	_, err := tx.Exec(`INSERT INTO devices (secret_hash, player_id, created_ms, last_seen_ms) VALUES (?, ?, ?, ?)`,
		hash, playerID, now, now)
	return err
}

// Pending lists unexpired pending requests, oldest first.
func (s *Service) Pending() ([]Request, error) {
	rows, err := s.db.Query(`SELECT r.id, r.name, r.color, r.created_ms, p.id IS NOT NULL
		FROM join_requests r LEFT JOIN players p ON p.name = r.name COLLATE NOCASE
		WHERE r.status = 'pending' AND r.created_ms > ? ORDER BY r.id`, s.pendingCutoffMs())
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Request
	for rows.Next() {
		var r Request
		var created int64
		if err := rows.Scan(&r.ID, &r.Name, &r.Color, &created, &r.Returning); err != nil {
			return nil, err
		}
		r.Created = time.UnixMilli(created)
		out = append(out, r)
	}
	return out, rows.Err()
}

// Await returns a channel that receives the decision for request id; call cancel when done waiting.
func (s *Service) Await(id int64) (<-chan bool, func()) { return s.waiters.add(id) }
