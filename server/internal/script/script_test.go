package script

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
)

type fakeHost struct {
	said     []string
	blocks   []string
	given    []string
	moved    []string
	refusing error
}

func (h *fakeHost) Say(text string) { h.said = append(h.said, text) }

func (h *fakeHost) SetBlock(x, y, z int32, name string) error {
	h.blocks = append(h.blocks, fmt.Sprintf("%d %d %d %s", x, y, z, name))
	return h.refusing
}

func (h *fakeHost) GiveItem(player, item string, count int) error {
	h.given = append(h.given, fmt.Sprintf("%s %s %d", player, item, count))
	return h.refusing
}

func (h *fakeHost) Teleport(player string, x, y, z float64) error {
	h.moved = append(h.moved, player)
	return h.refusing
}

var t0 = time.Unix(1000, 0)

func load(t *testing.T, src string) (*Script, *fakeHost) {
	t.Helper()
	h := &fakeHost{}
	s, err := Load(src, h, t0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return s, h
}

func TestTouchRunsTheHandlerWithThePlayersName(t *testing.T) {
	s, h := load(t, `function on_touch(player) say("hi " .. player) end`)
	if err := s.Touch("ada", t0); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(h.said, []string{"hi ada"}) {
		t.Fatalf("said %q", h.said)
	}
}

func TestPowerTellsTheScriptOnOrOff(t *testing.T) {
	s, h := load(t, `function on_power(on) if on then set_block(1, 2, 3, "lamp") else say("off") end end`)
	if err := s.Power(true, t0); err != nil {
		t.Fatal(err)
	}
	if err := s.Power(false, t0); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(h.blocks, []string{"1 2 3 lamp"}) || !slices.Equal(h.said, []string{"off"}) {
		t.Fatalf("blocks %q, said %q", h.blocks, h.said)
	}
}

func TestMissingHandlersDoNothing(t *testing.T) {
	s, _ := load(t, `x = 1`)
	if err := s.Touch("ada", t0); err != nil {
		t.Fatal(err)
	}
	if err := s.Power(true, t0); err != nil {
		t.Fatal(err)
	}
}

func TestGivesAndTeleportsPlayers(t *testing.T) {
	s, h := load(t, `function on_touch(p) give_item(p, "torch", 3) teleport(p, 0.5, 20, 0.5) end`)
	if err := s.Touch("ada", t0); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(h.given, []string{"ada torch 3"}) || !slices.Equal(h.moved, []string{"ada"}) {
		t.Fatalf("given %q, moved %q", h.given, h.moved)
	}
}

func TestTimersRunWhenDue(t *testing.T) {
	s, h := load(t, `function on_touch(p) after(2, function() say("later " .. p) end) end`)
	if _, ok := s.NextTimer(); ok {
		t.Fatal("no timer yet")
	}
	if err := s.Touch("ada", t0); err != nil {
		t.Fatal(err)
	}
	at, ok := s.NextTimer()
	if !ok || !at.Equal(t0.Add(2*time.Second)) {
		t.Fatalf("next timer %v %v", at, ok)
	}
	if err := s.RunTimers(t0.Add(time.Second)); err != nil || len(h.said) != 0 {
		t.Fatalf("ran early: %q %v", h.said, err)
	}
	if err := s.RunTimers(t0.Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(h.said, []string{"later ada"}) {
		t.Fatalf("said %q", h.said)
	}
	if _, ok := s.NextTimer(); ok {
		t.Fatal("timers run once")
	}
}

func TestEndlessLoopsAreCutOff(t *testing.T) {
	for name, src := range map[string]string{
		"top level": `while true do end`,
		"caught":    `while true do pcall(function() while true do end end) end`,
		"recursion": `local function f() return 1 + f() end f()`,
	} {
		start := time.Now()
		_, err := Load(src, &fakeHost{}, t0)
		if err == nil {
			t.Errorf("%s: ran forever without an error", name)
		}
		if time.Since(start) > time.Second {
			t.Errorf("%s: took %v to stop", name, time.Since(start))
		}
	}
	s, _ := load(t, `function on_touch() while true do end end`)
	if err := s.Touch("ada", t0); err == nil {
		t.Fatal("a stuck handler should fail")
	}
	if err := s.Touch("ada", t0); !errors.Is(err, ErrStopped) {
		t.Fatalf("a script that overran should stay stopped, got %v", err)
	}
}

func TestDangerousLibrariesAreGone(t *testing.T) {
	for _, name := range []string{
		"os", "io", "debug", "package", "require", "dofile", "loadfile", "load", "loadstring",
		"print", "collectgarbage", "getfenv", "setfenv", "module", "newproxy", "coroutine",
	} {
		s, h := load(t, `function on_touch() say(type(`+name+`)) end`)
		if err := s.Touch("ada", t0); err != nil || h.said[0] != "nil" {
			t.Errorf("%s is %q (%v)", name, h.said, err)
		}
	}
}

func TestSafeLibrariesStay(t *testing.T) {
	s, h := load(t, `function on_touch() say(string.upper("a") .. math.floor(2.5) .. table.concat({"b", "c"})) end`)
	if err := s.Touch("ada", t0); err != nil || h.said[0] != "A2bc" {
		t.Fatalf("said %q (%v)", h.said, err)
	}
}

func TestHugeStringsAreRefused(t *testing.T) {
	for _, src := range []string{
		`x = string.rep("x", 1e9)`,
		`x = string.format("%999999999d", 1)`,
	} {
		if _, err := Load(src, &fakeHost{}, t0); err == nil {
			t.Errorf("%s: no error", src)
		}
	}
	if _, err := Load(strings.Repeat("-", MaxSource+1), &fakeHost{}, t0); !errors.Is(err, ErrTooLong) {
		t.Fatalf("long source: %v", err)
	}
}

func TestActionsPerRunAreCapped(t *testing.T) {
	s, h := load(t, `function on_touch() for i = 1, 1000 do say("x") end end`)
	if err := s.Touch("ada", t0); err == nil {
		t.Fatal("spamming should fail")
	}
	if len(h.said) != MaxActions {
		t.Fatalf("said %d times", len(h.said))
	}
}

func TestTimersAreCapped(t *testing.T) {
	s, _ := load(t, `function on_touch() for i = 1, 100 do after(1, function() end) end end`)
	if err := s.Touch("ada", t0); err == nil {
		t.Fatal("too many timers should fail")
	}
}

func TestBadArgumentsAndHostRefusalsBecomeErrors(t *testing.T) {
	s, h := load(t, `function on_touch(p) give_item(p, "torch", 1000) end`)
	if err := s.Touch("ada", t0); err == nil || len(h.given) != 0 {
		t.Fatalf("given %q (%v)", h.given, err)
	}
	s, h = load(t, `function on_touch() set_block(0, 0, 0, "stone") end`)
	h.refusing = errors.New("that spot is protected")
	if err := s.Touch("ada", t0); err == nil || !strings.Contains(err.Error(), "protected") {
		t.Fatalf("refusal lost: %v", err)
	}
}

func TestSyntaxErrorsSayWhere(t *testing.T) {
	_, err := Load("x = = 1", &fakeHost{}, t0)
	if err == nil || err.Error() != "line 1: near '=': syntax error" {
		t.Fatalf("got %v", err)
	}
}
