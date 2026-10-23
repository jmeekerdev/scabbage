// Package agent lets programs drive a player's own avatar: tokens, the tool table, and the MCP
// and JSON endpoints that serve them.
package agent

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"regexp"
	"strings"
	"time"
)

const (
	// TokenPrefix starts every token, so a leaked one is easy to recognize.
	TokenPrefix = "scab_"
	// MaxTokens is how many agents one player may link at once.
	MaxTokens = 5
	// DefaultLabel names a token linked without a label.
	DefaultLabel = "agent"
	tokenBytes   = 32
)

var (
	ErrBadToken = errors.New("unknown or revoked agent token")
	ErrBadLabel = errors.New("labels are 1-16 letters, digits, _ or -, and not \"all\"")
	ErrTooMany  = errors.New("too many agents linked; revoke one first")
	ErrNoPlayer = errors.New("no such player")
)

var labelPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,16}$`)

// Owner is the player a token acts for.
type Owner struct {
	PlayerID    int64
	Name        string
	Admin       bool
	Allowed     bool // an admin turned agents on for them
	SkipConfirm bool // their agents' jobs needn't wait for /yes, if they're an admin
	Label       string
}

// MayUseAgents reports whether the owner may drive their avatar with an agent.
func (o Owner) MayUseAgents() bool { return o.Admin || o.Allowed }

// MustConfirm reports whether the owner says /yes before their agents' jobs change the world.
func (o Owner) MustConfirm() bool { return !o.Admin || !o.SkipConfirm }

// Token describes a linked agent, without its secret.
type Token struct {
	Label             string
	Created, LastUsed time.Time // LastUsed is zero until first used
}

// Tokens keeps agent tokens as hashes, so a copied database can't drive anyone.
type Tokens struct {
	db  *sql.DB
	now func() time.Time
}

// NewTokens uses db, which must already be migrated; now is time.Now when nil.
func NewTokens(db *sql.DB, now func() time.Time) *Tokens {
	if now == nil {
		now = time.Now
	}
	return &Tokens{db: db, now: now}
}

// Create makes a token for a player's agent, replacing any token with the same label.
func (t *Tokens) Create(playerID int64, label string) (string, error) {
	if !labelPattern.MatchString(label) || strings.EqualFold(label, "all") {
		return "", ErrBadLabel
	}
	secret := make([]byte, tokenBytes)
	_, _ = rand.Read(secret) // never fails
	token := TokenPrefix + base64.RawURLEncoding.EncodeToString(secret)
	tx, err := t.db.Begin()
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`DELETE FROM agent_tokens WHERE player_id = ? AND label = ?`, playerID, label); err != nil {
		return "", err
	}
	var n int
	if err := tx.QueryRow(`SELECT count(*) FROM agent_tokens WHERE player_id = ?`, playerID).Scan(&n); err != nil {
		return "", err
	}
	if n >= MaxTokens {
		return "", ErrTooMany
	}
	if _, err := tx.Exec(`INSERT INTO agent_tokens (hash, player_id, label, created_ms) VALUES (?, ?, ?, ?)`,
		hashToken(token), playerID, label, t.now().UnixMilli()); err != nil {
		return "", err
	}
	return token, tx.Commit()
}

// Owner finds who a token acts for, noting that it was used.
func (t *Tokens) Owner(token string) (Owner, error) {
	o := Owner{}
	err := t.db.QueryRow(`UPDATE agent_tokens SET last_used_ms = ? WHERE hash = ?
		RETURNING player_id, label, (SELECT name FROM players WHERE id = player_id),
		(SELECT admin FROM players WHERE id = player_id), (SELECT agent_allowed FROM players WHERE id = player_id),
		(SELECT agent_skip_confirm FROM players WHERE id = player_id)`,
		t.now().UnixMilli(), hashToken(token)).Scan(&o.PlayerID, &o.Label, &o.Name, &o.Admin, &o.Allowed, &o.SkipConfirm)
	if errors.Is(err, sql.ErrNoRows) {
		return Owner{}, ErrBadToken
	}
	return o, err
}

// List shows a player's linked agents by label.
func (t *Tokens) List(playerID int64) ([]Token, error) {
	rows, err := t.db.Query(`SELECT label, created_ms, last_used_ms FROM agent_tokens WHERE player_id = ? ORDER BY label`, playerID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Token
	for rows.Next() {
		var label string
		var created, used int64
		if err := rows.Scan(&label, &created, &used); err != nil {
			return nil, err
		}
		tok := Token{Label: label, Created: time.UnixMilli(created)}
		if used > 0 {
			tok.LastUsed = time.UnixMilli(used)
		}
		out = append(out, tok)
	}
	return out, rows.Err()
}

// Revoke drops the token with this label, or every one of the player's for "all".
func (t *Tokens) Revoke(playerID int64, label string) (int, error) {
	q, args := `DELETE FROM agent_tokens WHERE player_id = ? AND label = ?`, []any{playerID, label}
	if strings.EqualFold(label, "all") {
		q, args = `DELETE FROM agent_tokens WHERE player_id = ?`, args[:1]
	}
	res, err := t.db.Exec(q, args...)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	return int(n), err
}

// Allow turns agents on or off for a player who isn't an admin; off drops their tokens too.
func (t *Tokens) Allow(playerID int64, on bool) error {
	res, err := t.db.Exec(`UPDATE players SET agent_allowed = ? WHERE id = ?`, on, playerID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNoPlayer
	}
	if !on {
		_, err = t.Revoke(playerID, "all")
	}
	return err
}

// SetConfirm sets whether a player's agents' jobs wait for their /yes; it only counts for admins.
func (t *Tokens) SetConfirm(playerID int64, on bool) error {
	res, err := t.db.Exec(`UPDATE players SET agent_skip_confirm = ? WHERE id = ?`, !on, playerID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNoPlayer
	}
	return nil
}

// Allowed reports whether an admin turned agents on for a player.
func (t *Tokens) Allowed(playerID int64) (bool, error) {
	var on bool
	err := t.db.QueryRow(`SELECT agent_allowed FROM players WHERE id = ?`, playerID).Scan(&on)
	if errors.Is(err, sql.ErrNoRows) {
		return false, ErrNoPlayer
	}
	return on, err
}

func hashToken(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}
