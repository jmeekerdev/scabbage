package access

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"strings"
	"time"
)

const (
	codeAlphabet   = "23456789ABCDEFGHJKMNPQRSTUVWXYZ" // no 0/O, 1/I/L
	codeLength     = 8
	adminInviteTTL = 7 * 24 * time.Hour
)

// Code is a join code that admits devices until it runs out of uses or expires.
type Code struct {
	Code    string
	Uses    int
	Expires time.Time
}

func newCode() string {
	b := make([]byte, codeLength)
	for i := range b {
		b[i] = codeAlphabet[randIndex(len(codeAlphabet))]
	}
	return string(b)
}

// randIndex returns a uniform random index in [0, n) using rejection sampling.
func randIndex(n int) int {
	limit := 256 - 256%n
	var b [1]byte
	for {
		_, _ = rand.Read(b[:]) // crypto/rand.Read never fails
		if int(b[0]) < limit {
			return int(b[0]) % n
		}
	}
}

// normalizeCode uppercases and drops separators so "abcd-efgh" matches "ABCDEFGH".
func normalizeCode(s string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(s) {
		if strings.ContainsRune(codeAlphabet, r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// CreateCode makes a regular join code; requests made with it still need admin approval.
func (s *Service) CreateCode(uses int, ttl time.Duration) (Code, error) {
	return s.insertCode(uses, ttl, false)
}

func (s *Service) insertCode(uses int, ttl time.Duration, admin bool) (Code, error) {
	c := Code{Code: newCode(), Uses: uses, Expires: s.opts.Now().Add(ttl)}
	_, err := s.db.Exec(`INSERT INTO join_codes (code, uses_left, expires_ms, admin, created_ms) VALUES (?, ?, ?, ?, ?)`,
		c.Code, c.Uses, c.Expires.UnixMilli(), admin, s.nowMs())
	return c, err
}

// AdminInvite returns a single-use admin code while no admin exists, so the server owner can get in.
func (s *Service) AdminInvite() (code string, ok bool, err error) {
	var admins int
	if err := s.db.QueryRow(`SELECT count(*) FROM players WHERE admin = 1`).Scan(&admins); err != nil || admins > 0 {
		return "", false, err
	}
	err = s.db.QueryRow(`SELECT code FROM join_codes WHERE admin = 1 AND uses_left > 0 AND expires_ms > ?`,
		s.nowMs()).Scan(&code)
	if err == nil {
		return code, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", false, err
	}
	c, err := s.insertCode(1, adminInviteTTL, true)
	return c.Code, err == nil, err
}

// useCode consumes one use of code, reporting whether it is an admin invite.
func (s *Service) useCode(tx *sql.Tx, code string) (admin bool, err error) {
	res, err := tx.Exec(`UPDATE join_codes SET uses_left = uses_left - 1
		WHERE code = ? AND uses_left > 0 AND expires_ms > ?`, code, s.nowMs())
	if err != nil {
		return false, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return false, ErrBadCode
	}
	err = tx.QueryRow(`SELECT admin FROM join_codes WHERE code = ?`, code).Scan(&admin)
	return admin, err
}
