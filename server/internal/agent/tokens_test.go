package agent

import (
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"scabbage/internal/store"
)

func newTokens(t *testing.T) (*Tokens, *sql.DB) {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, p := range []struct {
		id    int
		name  string
		admin bool
	}{{1, "Ada", true}, {2, "Kid", false}} {
		if _, err := db.Exec(`INSERT INTO players (id, name, color, admin, created_ms) VALUES (?, ?, 0, ?, 0)`, p.id, p.name, p.admin); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Unix(1_700_000_000, 0)
	return NewTokens(db, func() time.Time { return now }), db
}

func TestTokenFindsItsOwner(t *testing.T) {
	tokens, _ := newTokens(t)
	tok, err := tokens.Create(1, "laptop")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(tok, TokenPrefix) || len(tok) < 40 {
		t.Fatalf("token %q looks too weak", tok)
	}
	o, err := tokens.Owner(tok)
	if err != nil {
		t.Fatal(err)
	}
	if o != (Owner{PlayerID: 1, Name: "Ada", Admin: true, Label: "laptop"}) {
		t.Fatalf("owner = %+v", o)
	}
	if !o.MayUseAgents() {
		t.Fatal("admins may use agents by default")
	}
}

func TestUnknownTokensAreRefused(t *testing.T) {
	tokens, _ := newTokens(t)
	for _, tok := range []string{"", "nope", TokenPrefix + "AAAA"} {
		if _, err := tokens.Owner(tok); !errors.Is(err, ErrBadToken) {
			t.Errorf("Owner(%q) err = %v, want ErrBadToken", tok, err)
		}
	}
}

func TestOnlyTheHashIsStored(t *testing.T) {
	tokens, db := newTokens(t)
	tok, _ := tokens.Create(1, "laptop")
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM agent_tokens WHERE hash = ? OR CAST(hash AS TEXT) = ?`, tok, tok).Scan(&n); err != nil || n != 0 {
		t.Fatalf("token stored as is (n=%d, err=%v)", n, err)
	}
}

func TestRelinkingALabelReplacesItsToken(t *testing.T) {
	tokens, _ := newTokens(t)
	old, _ := tokens.Create(1, "laptop")
	fresh, _ := tokens.Create(1, "laptop")
	if _, err := tokens.Owner(old); !errors.Is(err, ErrBadToken) {
		t.Fatal("the old token still works")
	}
	if _, err := tokens.Owner(fresh); err != nil {
		t.Fatal(err)
	}
}

func TestListAndRevoke(t *testing.T) {
	tokens, _ := newTokens(t)
	a, _ := tokens.Create(1, "laptop")
	b, _ := tokens.Create(1, "desktop")
	if _, err := tokens.Owner(a); err != nil { // marks it used
		t.Fatal(err)
	}
	list, err := tokens.List(1)
	if err != nil || len(list) != 2 || list[0].Label != "desktop" || list[1].Label != "laptop" || list[1].LastUsed.IsZero() {
		t.Fatalf("list = %+v, %v", list, err)
	}
	if n, _ := tokens.Revoke(1, "laptop"); n != 1 {
		t.Fatalf("revoked %d, want 1", n)
	}
	if _, err := tokens.Owner(a); !errors.Is(err, ErrBadToken) {
		t.Fatal("revoked token still works")
	}
	if n, _ := tokens.Revoke(1, "all"); n != 1 {
		t.Fatalf("revoked %d, want 1", n)
	}
	if _, err := tokens.Owner(b); !errors.Is(err, ErrBadToken) {
		t.Fatal("revoke all left a token")
	}
}

func TestBadLabelsAndTooManyTokensAreRefused(t *testing.T) {
	tokens, _ := newTokens(t)
	for _, label := range []string{"", "all", "has space", strings.Repeat("x", 17)} {
		if _, err := tokens.Create(1, label); !errors.Is(err, ErrBadLabel) {
			t.Errorf("Create(%q) err = %v, want ErrBadLabel", label, err)
		}
	}
	for i := range MaxTokens {
		if _, err := tokens.Create(1, string(rune('a'+i))); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := tokens.Create(1, "extra"); !errors.Is(err, ErrTooMany) {
		t.Fatalf("err = %v, want ErrTooMany", err)
	}
}

func TestPlayersNeedAnAdminToTurnAgentsOn(t *testing.T) {
	tokens, _ := newTokens(t)
	tok, _ := tokens.Create(2, "tablet")
	o, _ := tokens.Owner(tok)
	if o.MayUseAgents() {
		t.Fatal("agents are on for a player no admin allowed")
	}
	if err := tokens.Allow(2, true); err != nil {
		t.Fatal(err)
	}
	if o, _ = tokens.Owner(tok); !o.MayUseAgents() {
		t.Fatal("allowing didn't turn agents on")
	}
	if ok, _ := tokens.Allowed(2); !ok {
		t.Fatal("Allowed says no")
	}
	if err := tokens.Allow(2, false); err != nil {
		t.Fatal(err)
	}
	if _, err := tokens.Owner(tok); !errors.Is(err, ErrBadToken) {
		t.Fatal("turning agents off should drop the player's tokens")
	}
	if err := tokens.Allow(99, true); !errors.Is(err, ErrNoPlayer) {
		t.Fatalf("err = %v, want ErrNoPlayer", err)
	}
}
