package scripts

import (
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"scabbage/internal/store"
	"scabbage/internal/world"
)

const amy = 1

var (
	here  = world.Pos{X: 1, Y: 2, Z: 3}
	there = world.Pos{X: 9, Y: 2, Z: 3}
	t0    = time.Unix(1000, 0)
)

// fakeHost records what scripts said, and lets SetBlock power other script blocks.
type fakeHost struct {
	at    world.Pos
	said  *[]string
	onSet func(at world.Pos)
}

func (h fakeHost) Say(text string) { *h.said = append(*h.said, text) }
func (h fakeHost) SetBlock(x, y, z int32, name string) error {
	if h.onSet != nil {
		h.onSet(world.Pos{X: x, Y: y, Z: z})
	}
	return nil
}
func (fakeHost) GiveItem(string, string, int) error               { return nil }
func (fakeHost) Teleport(string, float64, float64, float64) error { return nil }
func (fakeHost) AddPoints(string, int) error                      { return nil }
func (h fakeHost) Where() (x, y, z int32)                         { return h.at.X, h.at.Y, h.at.Z }

type env struct {
	svc   *Service
	said  []string
	onSet func(at world.Pos)
}

func newEnv(t *testing.T) *env {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`INSERT INTO players (id, name, color, created_ms) VALUES (?, 'Amy', 0, 0)`, amy); err != nil {
		t.Fatal(err)
	}
	e := &env{}
	e.svc = New(db, func(at world.Pos, owner int64) Host {
		return fakeHost{at: at, said: &e.said, onSet: func(p world.Pos) {
			if e.onSet != nil {
				e.onSet(p)
			}
		}}
	})
	t.Cleanup(e.svc.Forget)
	return e
}

func (e *env) save(t *testing.T, at world.Pos, src string) {
	t.Helper()
	if err := e.svc.Save(at, src, amy, t0); err != nil {
		t.Fatal(err)
	}
}

func TestSavedScriptsRunWhenTouchedAndPowered(t *testing.T) {
	e := newEnv(t)
	e.save(t, here, `function on_touch(p) say("hi " .. p) end function on_power(on) say(tostring(on)) end`)
	if f := e.svc.Touch(here, "bob", t0); len(f) != 0 {
		t.Fatal(f)
	}
	if f := e.svc.Power(here, true, t0); len(f) != 0 {
		t.Fatal(f)
	}
	if !slices.Equal(e.said, []string{"hi bob", "true"}) {
		t.Fatalf("said %q", e.said)
	}
}

func TestBlocksWithoutScriptsDoNothing(t *testing.T) {
	e := newEnv(t)
	if f := e.svc.Touch(here, "bob", t0); len(f) != 0 || len(e.said) != 0 {
		t.Fatalf("%v %q", f, e.said)
	}
}

func TestScriptsSurviveBeingForgotten(t *testing.T) {
	e := newEnv(t)
	e.save(t, here, `n = 0 function on_touch() n = n + 1 say(tostring(n)) end`)
	e.svc.Touch(here, "bob", t0)
	e.svc.Forget()
	e.svc.Touch(here, "bob", t0)
	if !slices.Equal(e.said, []string{"1", "1"}) {
		t.Fatalf("said %q", e.said)
	}
	got, ok, err := e.svc.Source(here)
	if err != nil || !ok || got.Owner != amy || !strings.Contains(got.Source, "n = 0") {
		t.Fatalf("source %+v %v %v", got, ok, err)
	}
}

func TestBrokenScriptsAreSavedWithTheirError(t *testing.T) {
	e := newEnv(t)
	err := e.svc.Save(here, `function on_touch(`, amy, t0)
	if err == nil {
		t.Fatal("no syntax error")
	}
	got, ok, _ := e.svc.Source(here)
	if !ok || got.Source != `function on_touch(` || got.Err == "" {
		t.Fatalf("source %+v", got)
	}
	if f := e.svc.Touch(here, "bob", t0); len(f) != 0 {
		t.Fatalf("a broken script should just sit there: %v", f)
	}
}

func TestFailuresNameTheScriptAndItsOwner(t *testing.T) {
	e := newEnv(t)
	e.save(t, here, `function on_touch() error("oops") end`)
	f := e.svc.Touch(here, "bob", t0)
	if len(f) != 1 || f[0].Pos != here || f[0].Owner != amy || !strings.Contains(f[0].Err.Error(), "oops") {
		t.Fatalf("failures %+v", f)
	}
	got, _, _ := e.svc.Source(here)
	if !strings.Contains(got.Err, "oops") {
		t.Fatalf("the editor should see the last error, got %q", got.Err)
	}
}

func TestDeletedScriptsAreGone(t *testing.T) {
	e := newEnv(t)
	e.save(t, here, `function on_touch() say("hi") end`)
	if err := e.svc.Delete(here); err != nil {
		t.Fatal(err)
	}
	e.svc.Touch(here, "bob", t0)
	if _, ok, _ := e.svc.Source(here); ok || len(e.said) != 0 {
		t.Fatalf("still there: %q", e.said)
	}
}

func TestTimersRunOnTheirScripts(t *testing.T) {
	e := newEnv(t)
	e.save(t, here, `function on_touch() after(1, function() say("ding") end) end`)
	e.svc.Touch(here, "bob", t0)
	e.svc.RunTimers(t0)
	if len(e.said) != 0 {
		t.Fatal("rang early")
	}
	if f := e.svc.RunTimers(t0.Add(time.Second)); len(f) != 0 || !slices.Equal(e.said, []string{"ding"}) {
		t.Fatalf("%v %q", f, e.said)
	}
}

func TestScriptsSetOffByScriptsRunAfterward(t *testing.T) {
	e := newEnv(t)
	e.save(t, here, `function on_touch() set_block(9, 2, 3, "wire") say("first") end`)
	e.save(t, there, `function on_power(on) say("second") end`)
	e.onSet = func(at world.Pos) {
		if f := e.svc.Power(at, true, t0); len(f) != 0 {
			t.Errorf("queued power failed: %v", f)
		}
	}
	e.svc.Touch(here, "bob", t0)
	if !slices.Equal(e.said, []string{"first", "second"}) {
		t.Fatalf("said %q", e.said)
	}
}

func TestScriptsPoweringEachOtherStopAfterAWhile(t *testing.T) {
	e := newEnv(t)
	e.save(t, here, `function on_power(on) set_block(9, 2, 3, "wire") end`)
	e.save(t, there, `function on_power(on) set_block(1, 2, 3, "wire") end`)
	e.onSet = func(at world.Pos) { e.svc.Power(at, true, t0) }
	f := e.svc.Power(here, true, t0)
	if len(f) != 1 || !errors.Is(f[0].Err, ErrChain) {
		t.Fatalf("failures %v", f)
	}
}

func TestSourceIsCapped(t *testing.T) {
	e := newEnv(t)
	if err := e.svc.Save(here, strings.Repeat("-", 20_000), amy, t0); err == nil {
		t.Fatal("saved a huge script")
	}
	if _, ok, _ := e.svc.Source(here); ok {
		t.Fatal("a refused script should not be stored")
	}
}

func TestRoundsReachScriptsInTheirArena(t *testing.T) {
	e := newEnv(t)
	e.save(t, here, `function on_round_start(p) say("go " .. p[1]) end function on_round_end(w) say("won " .. w[1]) end`)
	e.save(t, world.Pos{X: 100, Y: 2, Z: 3}, `function on_round_start(p) say("far away") end`)
	inside, err := e.svc.Inside(0, 0, 10, 10)
	if err != nil || !slices.Equal(inside, []world.Pos{here}) {
		t.Fatalf("inside %v %v", inside, err)
	}
	for _, at := range inside {
		e.svc.RoundStart(at, []string{"amy"}, t0)
		e.svc.RoundEnd(at, []string{"amy"}, t0)
	}
	if !slices.Equal(e.said, []string{"go amy", "won amy"}) {
		t.Fatalf("said %q", e.said)
	}
}
