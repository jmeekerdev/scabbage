package access

import (
	"errors"
	"testing"
)

func (f *fixture) approved(t *testing.T, s []byte, name string) Player {
	t.Helper()
	r := f.pending(t, s, name)
	if err := f.svc.Decide(r.RequestID, true); err != nil {
		t.Fatal(err)
	}
	got, err := f.svc.Check(s)
	if err != nil {
		t.Fatal(err)
	}
	return got.Player
}

func TestPlayersAreListedByName(t *testing.T) {
	f := newFixture(t)
	f.approved(t, secret(1), "zed")
	f.approved(t, secret(2), "Amy")
	ps, err := f.svc.Players()
	if err != nil {
		t.Fatal(err)
	}
	if len(ps) != 2 || ps[0].Name != "Amy" || ps[1].Name != "zed" || ps[0].Admin {
		t.Fatalf("players = %+v", ps)
	}
}

func TestSetAdminGrantsAndRemovesRights(t *testing.T) {
	f := newFixture(t)
	p := f.approved(t, secret(1), "Amy")
	for _, admin := range []bool{true, false} {
		got, err := f.svc.SetAdmin(p.ID, admin)
		if err != nil {
			t.Fatal(err)
		}
		if got.Admin != admin || got.Name != "Amy" {
			t.Fatalf("SetAdmin(%v) = %+v", admin, got)
		}
		if r, _ := f.svc.Check(secret(1)); r.Player.Admin != admin {
			t.Fatalf("device sees admin = %v, want %v", r.Player.Admin, admin)
		}
	}
}

func TestSetAdminOnUnknownPlayer(t *testing.T) {
	f := newFixture(t)
	if _, err := f.svc.SetAdmin(99, true); !errors.Is(err, ErrNoPlayer) {
		t.Fatalf("err = %v, want ErrNoPlayer", err)
	}
}
