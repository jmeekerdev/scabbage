package access

import (
	"errors"
	"reflect"
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

func TestNewPlayersFollowTheServerMode(t *testing.T) {
	f := newFixture(t)
	if p := f.approved(t, secret(1), "Amy"); p.Mode != ModeDefault {
		t.Fatalf("mode = %q, want the server default", p.Mode)
	}
}

func TestSetModeIsRemembered(t *testing.T) {
	f := newFixture(t)
	p := f.approved(t, secret(1), "Amy")
	for _, m := range []Mode{ModeSurvival, ModeCreative, ModeDefault} {
		got, err := f.svc.SetMode(p.ID, m)
		if err != nil {
			t.Fatal(err)
		}
		if got.Mode != m || got.Name != "Amy" {
			t.Fatalf("SetMode(%q) = %+v", m, got)
		}
		if r, _ := f.svc.Check(secret(1)); r.Player.Mode != m {
			t.Fatalf("device sees mode %q, want %q", r.Player.Mode, m)
		}
		if ps, _ := f.svc.Players(); ps[0].Mode != m {
			t.Fatalf("list shows mode %q, want %q", ps[0].Mode, m)
		}
	}
}

func TestSetModeRejectsUnknownModes(t *testing.T) {
	f := newFixture(t)
	p := f.approved(t, secret(1), "Amy")
	if _, err := f.svc.SetMode(p.ID, "flying"); err == nil {
		t.Fatal("unknown mode was stored")
	}
	if _, err := f.svc.SetMode(99, ModeSurvival); !errors.Is(err, ErrNoPlayer) {
		t.Fatalf("err = %v, want ErrNoPlayer", err)
	}
}

func TestSavesAreRemembered(t *testing.T) {
	f := newFixture(t)
	p := f.approved(t, secret(1), "Amy")
	if got, err := f.svc.Saved(p.ID); err != nil || got.Placed || got.Health != 20 || len(got.Inventory) != 0 {
		t.Fatalf("new player has %+v, %v", got, err)
	}
	want := Save{Placed: true, X: 1.5, Y: 70, Z: -3.25, Yaw: 0.5, Pitch: -0.25, Flying: true, Health: 7, Inventory: []byte{1, 2, 3},
		Checkpoint: &[3]int32{4, -5, 6}}
	if err := f.svc.Store(p.ID, want); err != nil {
		t.Fatal(err)
	}
	if got, _ := f.svc.Saved(p.ID); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v", got)
	}
	want.Checkpoint = nil
	_ = f.svc.Store(p.ID, want)
	if got, _ := f.svc.Saved(p.ID); got.Checkpoint != nil {
		t.Fatalf("checkpoint stayed: %v", got.Checkpoint)
	}
	if _, err := f.svc.Saved(99); !errors.Is(err, ErrNoPlayer) {
		t.Fatalf("err = %v, want ErrNoPlayer", err)
	}
}
