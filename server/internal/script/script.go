// Package script runs players' Lua in a sandbox: only safe libraries, a small time budget per run,
// and a few capped calls into the world.
package script

import (
	"context"
	"errors"
	"regexp"
	"slices"
	"strings"
	"time"

	lua "github.com/yuin/gopher-lua"
)

const (
	// MaxSource is the longest script, in bytes.
	MaxSource = 16 * 1024
	// RunTime is how long one run (loading, an event or due timers) may take before the script stops.
	RunTime = 10 * time.Millisecond
	// MaxActions is how many world calls one run may make.
	MaxActions = 64
	// MaxTimers is how many after() calls may wait at once.
	MaxTimers = 16
)

var (
	ErrTooLong = errors.New("script is too long")
	ErrStopped = errors.New("script stopped: it ran too long")
)

// Script is one loaded script; it is not safe for concurrent use.
type Script struct {
	L       *lua.LState
	host    Host
	now     time.Time // when the current run started, for after()
	actions int       // world calls made in the current run
	timers  []timer
	answers map[string]*lua.LFunction // what to do with each player's answer to ask(), by lowercased name
	stopped bool
}

type timer struct {
	at time.Time
	fn *lua.LFunction
}

// Load compiles src and runs its top level, which usually just defines on_touch and on_power.
func Load(src string, host Host, now time.Time) (*Script, error) {
	if len(src) > MaxSource {
		return nil, ErrTooLong
	}
	s := &Script{L: newSandbox(), host: host}
	s.register()
	fn, err := s.L.Load(strings.NewReader(src), "script")
	err = syntax(err)
	if err == nil {
		err = s.run(now, fn)
	}
	if err != nil {
		s.Close()
		return nil, err
	}
	return s, nil
}

// Close frees the Lua state.
func (s *Script) Close() { s.L.Close() }

// Touch runs on_touch(player), if the script has one.
func (s *Script) Touch(player string, now time.Time) error {
	return s.handle("on_touch", now, lua.LString(player))
}

// Power runs on_power(on) when the power reaching the script changes.
func (s *Script) Power(on bool, now time.Time) error {
	return s.handle("on_power", now, lua.LBool(on))
}

// Talk runs on_talk(player) when someone talks to the script's character.
func (s *Script) Talk(player string, now time.Time) error {
	return s.handle("on_talk", now, lua.LString(player))
}

// Answer runs what ask() said to do with the player's answer, once; without a question waiting it does nothing.
func (s *Script) Answer(player, answer string, now time.Time) error {
	key := strings.ToLower(player)
	fn, ok := s.answers[key]
	if !ok {
		return nil
	}
	delete(s.answers, key)
	return s.run(now, fn, lua.LString(answer))
}

// RoundStart runs on_round_start(players) when a round starts in the script's arena.
func (s *Script) RoundStart(players []string, now time.Time) error {
	return s.handle("on_round_start", now, s.list(players))
}

// RoundEnd runs on_round_end(winners) when the round ends; nobody may have won.
func (s *Script) RoundEnd(winners []string, now time.Time) error {
	return s.handle("on_round_end", now, s.list(winners))
}

func (s *Script) list(names []string) *lua.LTable {
	t := s.L.CreateTable(len(names), 0)
	for _, n := range names {
		t.Append(lua.LString(n))
	}
	return t
}

// NextTimer is when the earliest after() is due, if any wait.
func (s *Script) NextTimer() (time.Time, bool) {
	if len(s.timers) == 0 {
		return time.Time{}, false
	}
	return slices.MinFunc(s.timers, func(a, b timer) int { return a.at.Compare(b.at) }).at, true
}

// RunTimers runs every after() due by now, earliest first.
func (s *Script) RunTimers(now time.Time) error {
	var due []timer
	s.timers = slices.DeleteFunc(s.timers, func(t timer) bool {
		if t.at.After(now) {
			return false
		}
		due = append(due, t)
		return true
	})
	slices.SortStableFunc(due, func(a, b timer) int { return a.at.Compare(b.at) })
	return s.run(now, s.L.NewFunction(func(L *lua.LState) int {
		for _, t := range due {
			L.Push(t.fn)
			L.Call(0, 0)
		}
		return 0
	}))
}

func (s *Script) handle(name string, now time.Time, args ...lua.LValue) error {
	fn, ok := s.L.GetGlobal(name).(*lua.LFunction)
	if !ok {
		return nil
	}
	return s.run(now, fn, args...)
}

// run calls fn within RunTime; a script that overruns stops for good.
func (s *Script) run(now time.Time, fn *lua.LFunction, args ...lua.LValue) error {
	if s.stopped {
		return ErrStopped
	}
	s.now, s.actions = now, 0
	ctx, cancel := context.WithTimeout(context.Background(), RunTime)
	defer cancel()
	s.L.SetContext(ctx)
	defer s.L.RemoveContext()
	err := s.L.CallByParam(lua.P{Fn: fn, Protect: true}, args...)
	if ctx.Err() != nil {
		s.stopped = true
		return ErrStopped
	}
	return message(err)
}

// message is a Lua error without its Go stack trace, short enough to show the script's author.
func message(err error) error {
	var apiErr *lua.ApiError
	if errors.As(err, &apiErr) && apiErr.Object != nil {
		return errors.New(apiErr.Object.String())
	}
	return err
}

var compileError = regexp.MustCompile(`line:(\d+)\(column:\d+\)\s+(.*?)\s*$`)

// syntax rewords a compile error as "line 3: near 'end': syntax error".
func syntax(err error) error {
	if err == nil {
		return nil
	}
	if m := compileError.FindStringSubmatch(err.Error()); m != nil {
		return errors.New("line " + m[1] + ": " + strings.Join(strings.Fields(m[2]), " "))
	}
	return err
}
