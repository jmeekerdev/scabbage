package plan

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"scabbage/internal/store"
)

type fixture struct {
	svc *Service
	db  *sql.DB
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "plans.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return &fixture{db: db, svc: New(db, func() time.Time { return time.UnixMilli(1000) })}
}

func (f *fixture) player(t *testing.T, name string) int64 {
	t.Helper()
	res, err := f.db.Exec(`INSERT INTO players (name, color, admin, created_ms) VALUES (?, 255, 0, 0)`, name)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	return id
}

func named(name string) Plan {
	p := hut()
	p.Name = name
	if err := p.Check(isBlock); err != nil {
		panic(err)
	}
	return p
}

func (f *fixture) save(t *testing.T, owner int64, name string) {
	t.Helper()
	if err := f.svc.Save(owner, named(name)); err != nil {
		t.Fatal(err)
	}
}

func TestSaveAndGetOwnPlan(t *testing.T) {
	f := newFixture(t)
	ann := f.player(t, "ann")
	f.save(t, ann, "hut")
	got, err := f.svc.Get(ann, "HUT")
	if err != nil {
		t.Fatal(err)
	}
	if got.Owner != "ann" || got.Shared || got.Layers[1][1] != "pgp" || got.Palette["g"] != "glass" {
		t.Errorf("got %+v", got)
	}
	if got.W != 3 || got.D != 2 || got.H != 3 || got.Count != 17 {
		t.Errorf("size %d %d %d, %d cells", got.W, got.D, got.H, got.Count)
	}
}

func TestSaveAgainReplacesThePlanAndKeepsItShared(t *testing.T) {
	f := newFixture(t)
	ann := f.player(t, "ann")
	f.save(t, ann, "hut")
	mustShare(t, f, ann, "hut")
	p := named("hut")
	p.About = "bigger"
	if err := f.svc.Save(ann, p); err != nil {
		t.Fatal(err)
	}
	got, _ := f.svc.Get(ann, "hut")
	if got.About != "bigger" || !got.Shared {
		t.Errorf("got %+v", got)
	}
}

func TestOthersSeeOnlySharedPlansAndOwnPlansComeFirst(t *testing.T) {
	f := newFixture(t)
	ann, bob := f.player(t, "ann"), f.player(t, "bob")
	f.save(t, ann, "hut")
	if _, err := f.svc.Get(bob, "hut"); !errors.Is(err, ErrNotFound) {
		t.Errorf("bob saw ann's private plan: %v", err)
	}
	mustShare(t, f, ann, "hut")
	if got, err := f.svc.Get(bob, "hut"); err != nil || got.Owner != "ann" {
		t.Errorf("bob can't see the shared plan: %+v %v", got, err)
	}
	f.save(t, bob, "hut")
	if got, _ := f.svc.Get(bob, "hut"); got.Owner != "bob" {
		t.Errorf("bob's own hut should win, got %s's", got.Owner)
	}
	list, err := f.svc.List(bob)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].Owner != "bob" || list[1].Owner != "ann" || list[1].Layers != nil {
		t.Errorf("list %+v", list)
	}
}

func TestSharedNamesAreUnique(t *testing.T) {
	f := newFixture(t)
	ann, bob := f.player(t, "ann"), f.player(t, "bob")
	f.save(t, ann, "hut")
	f.save(t, bob, "hut")
	mustShare(t, f, ann, "hut")
	if err := f.svc.Share(bob, "hut", true); !errors.Is(err, ErrSharedName) {
		t.Errorf("got %v", err)
	}
	if err := f.svc.Rename(bob, "hut", "shack"); err != nil {
		t.Fatal(err)
	}
	mustShare(t, f, bob, "shack")
	if err := f.svc.Rename(ann, "hut", "shack"); !errors.Is(err, ErrSharedName) {
		t.Errorf("renaming a shared plan onto another shared name: %v", err)
	}
	if err := f.svc.Share(ann, "hut", false); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.Rename(ann, "hut", "shack"); err != nil {
		t.Errorf("a private plan may share a name with someone's shared one: %v", err)
	}
}

func TestRenameRefusesTakenAndBadNames(t *testing.T) {
	f := newFixture(t)
	ann := f.player(t, "ann")
	f.save(t, ann, "hut")
	f.save(t, ann, "shed")
	if err := f.svc.Rename(ann, "hut", "shed"); !errors.Is(err, ErrTaken) {
		t.Errorf("got %v", err)
	}
	if err := f.svc.Rename(ann, "hut", "cabin"); err == nil {
		t.Error("renamed onto a template")
	}
	if err := f.svc.Rename(ann, "barn", "shack"); !errors.Is(err, ErrNotFound) {
		t.Errorf("got %v", err)
	}
}

func TestDelete(t *testing.T) {
	f := newFixture(t)
	ann, bob := f.player(t, "ann"), f.player(t, "bob")
	f.save(t, ann, "hut")
	f.save(t, ann, "shed")
	mustShare(t, f, ann, "shed")
	if err := f.svc.Delete(bob, "hut"); !errors.Is(err, ErrNotFound) {
		t.Errorf("bob deleted ann's plan: %v", err)
	}
	if err := f.svc.Delete(ann, "hut"); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.DeleteShared("hut"); !errors.Is(err, ErrNotFound) {
		t.Errorf("deleted a plan that isn't shared: %v", err)
	}
	if err := f.svc.DeleteShared("shed"); err != nil {
		t.Fatal(err)
	}
	if list, _ := f.svc.List(ann); len(list) != 0 {
		t.Errorf("left %+v", list)
	}
}

func TestSaveRefusesTooManyPlans(t *testing.T) {
	f := newFixture(t)
	ann := f.player(t, "ann")
	for i := range MaxPlans {
		f.save(t, ann, fmt.Sprintf("hut%d", i))
	}
	if err := f.svc.Save(ann, named("onemore")); !errors.Is(err, ErrFull) {
		t.Errorf("got %v", err)
	}
	if err := f.svc.Save(ann, named("hut0")); err != nil {
		t.Errorf("replacing one should still work: %v", err)
	}
}

func mustShare(t *testing.T, f *fixture, owner int64, name string) {
	t.Helper()
	if err := f.svc.Share(owner, name, true); err != nil {
		t.Fatal(err)
	}
}
