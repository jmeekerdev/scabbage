// Package scripts keeps the Lua that script blocks run: saved in the database, loaded on first use,
// and fed touches, power and timers one at a time.
package scripts

import (
	"database/sql"
	"errors"
	"time"

	"scabbage/internal/script"
	"scabbage/internal/world"
)

// Host is how one script block's Lua touches the world, acting for the script's owner.
type Host = script.Host

// HostFunc makes the host for the script at `at`, last saved by `owner`.
type HostFunc func(at world.Pos, owner int64) Host

const (
	// MaxChain is how many scripts one event may set off, through blocks they change.
	MaxChain = 32
	// maxLoaded bounds how many scripts stay in memory; those waiting on timers are kept.
	maxLoaded = 256
)

// ErrChain means scripts kept setting each other off, so the rest were skipped.
var ErrChain = errors.New("scripts kept setting each other off, so some were skipped")

// Saved is a script block's stored script and how it last went.
type Saved struct {
	Source string
	Owner  int64
	Err    string // the last error loading or running it, if any
}

// Failure is a script that went wrong, for telling its owner.
type Failure struct {
	Pos   world.Pos
	Owner int64
	Err   error
}

// Service runs script blocks; it is not safe for concurrent use, so callers hold one lock around it.
type Service struct {
	db     *sql.DB
	host   HostFunc
	loaded map[world.Pos]*entry
	queue  []event
	busy   bool // a script is running, so events it sets off wait in queue
	npcs   npcs
}

type entry struct {
	Saved
	run *script.Script // nil without a script, or when it failed to load
}

// event is one thing to tell a script block: a touch, a power change or due timers.
type event struct {
	at  world.Pos
	now time.Time
	run func(*script.Script, time.Time) error
}

// New makes a service whose scripts reach the world through host, reading the characters they made.
func New(db *sql.DB, host HostFunc, onNPC NPCFunc) (*Service, error) {
	s := &Service{db: db, host: host, loaded: map[world.Pos]*entry{}, npcs: npcs{
		byPos: map[world.Pos]NPC{}, unsaved: map[world.Pos]struct{}{}, notify: onNPC,
	}}
	return s, s.loadNPCs()
}

// Source is the script saved at `at`, if there is one.
func (s *Service) Source(at world.Pos) (Saved, bool, error) {
	var e *entry
	var err error
	s.drain(func() { e, err = s.entry(at, time.Now()) })
	if err != nil || e.run == nil && e.Source == "" {
		return Saved{}, false, err
	}
	return e.Saved, true, nil
}

// Save stores src as the script at `at` and starts it; a script that fails to load is kept
// so its author can fix it, and the error is returned. Its character leaves unless the new script names one.
func (s *Service) Save(at world.Pos, src string, owner int64, now time.Time) error {
	if len(src) > script.MaxSource {
		return script.ErrTooLong
	}
	if _, err := s.db.Exec(`INSERT INTO scripts (x, y, z, source, owner_id, updated_ms) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (x, y, z) DO UPDATE SET source = excluded.source, owner_id = excluded.owner_id,
			updated_ms = excluded.updated_ms, npc = ''`,
		at.X, at.Y, at.Z, src, owner, now.UnixMilli()); err != nil {
		return err
	}
	s.unload(at)
	s.dropNPC(at)
	var e *entry
	s.drain(func() { e = s.start(at, Saved{Source: src, Owner: owner}, now) })
	if e.Err != "" {
		return errors.New(e.Err)
	}
	return nil
}

// Delete forgets the script at `at` and its character, e.g. when its block is broken.
func (s *Service) Delete(at world.Pos) error {
	s.unload(at)
	s.dropNPC(at)
	_, err := s.db.Exec(`DELETE FROM scripts WHERE x = ? AND y = ? AND z = ?`, at.X, at.Y, at.Z)
	return err
}

// Touch runs on_touch(player) at `at`.
func (s *Service) Touch(at world.Pos, player string, now time.Time) []Failure {
	return s.dispatch(event{at, now, func(r *script.Script, now time.Time) error { return r.Touch(player, now) }})
}

// Power runs on_power(on) at `at`.
func (s *Service) Power(at world.Pos, on bool, now time.Time) []Failure {
	return s.dispatch(event{at, now, func(r *script.Script, now time.Time) error { return r.Power(on, now) }})
}

// Talk runs on_talk(player) at `at`, when someone talks to its character.
func (s *Service) Talk(at world.Pos, player string, now time.Time) []Failure {
	return s.dispatch(event{at, now, func(r *script.Script, now time.Time) error { return r.Talk(player, now) }})
}

// Answer hands the script at `at` the player's answer to its question.
func (s *Service) Answer(at world.Pos, player, answer string, now time.Time) []Failure {
	return s.dispatch(event{at, now, func(r *script.Script, now time.Time) error { return r.Answer(player, answer, now) }})
}

// RoundStart runs on_round_start(players) at `at`.
func (s *Service) RoundStart(at world.Pos, players []string, now time.Time) []Failure {
	return s.dispatch(event{at, now, func(r *script.Script, now time.Time) error { return r.RoundStart(players, now) }})
}

// RoundEnd runs on_round_end(winners) at `at`.
func (s *Service) RoundEnd(at world.Pos, winners []string, now time.Time) []Failure {
	return s.dispatch(event{at, now, func(r *script.Script, now time.Time) error { return r.RoundEnd(winners, now) }})
}

// Inside lists the script blocks in the columns from x0, z0 to x1, z1, e.g. an arena's zone.
func (s *Service) Inside(x0, z0, x1, z1 int32) ([]world.Pos, error) {
	rows, err := s.db.Query(`SELECT x, y, z FROM scripts WHERE x BETWEEN ? AND ? AND z BETWEEN ? AND ? ORDER BY x, y, z`,
		x0, x1, z0, z1)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []world.Pos
	for rows.Next() {
		var p world.Pos
		if err := rows.Scan(&p.X, &p.Y, &p.Z); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// RunTimers runs every loaded script's due timers.
func (s *Service) RunTimers(now time.Time) []Failure {
	var out []Failure
	for at, e := range s.loaded {
		if next, ok := s.nextTimer(e); ok && !next.After(now) {
			out = append(out, s.dispatch(event{at, now, (*script.Script).RunTimers})...)
		}
	}
	return out
}

// Forget drops every loaded script, e.g. when the world unloads; they start afresh when next used.
func (s *Service) Forget() {
	for at := range s.loaded {
		s.unload(at)
	}
}

// dispatch runs ev, then whatever it set off, unless a script is already running.
func (s *Service) dispatch(ev event) []Failure {
	s.queue = append(s.queue, ev)
	return s.drain(func() {})
}

// drain runs first, which may load scripts, then the queued events; scripts never run inside
// each other, so a running script's events wait for it. Characters' names are saved once all have run.
func (s *Service) drain(first func()) (out []Failure) {
	if s.busy {
		first()
		return nil
	}
	s.busy = true
	defer func() {
		s.busy = false
		out = append(out, s.saveNPCs()...)
	}()
	first()
	for ran := 0; len(s.queue) > 0; ran++ {
		next := s.queue[0]
		s.queue = s.queue[1:]
		if ran > MaxChain {
			s.queue = nil
			return append(out, s.failure(next.at, ErrChain))
		}
		if err := s.runEvent(next); err != nil {
			out = append(out, s.failure(next.at, err))
		}
	}
	return out
}

func (s *Service) runEvent(ev event) error {
	e, err := s.entry(ev.at, ev.now)
	if err != nil || e.run == nil {
		return err
	}
	if err := ev.run(e.run, ev.now); err != nil {
		e.Err = err.Error()
		return err
	}
	return nil
}

func (s *Service) failure(at world.Pos, err error) Failure {
	f := Failure{Pos: at, Err: err}
	if e, ok := s.loaded[at]; ok {
		f.Owner = e.Owner
	}
	return f
}

// entry is the loaded script at `at`, reading and starting it the first time.
func (s *Service) entry(at world.Pos, now time.Time) (*entry, error) {
	if e, ok := s.loaded[at]; ok {
		return e, nil
	}
	var saved Saved
	var owner sql.NullInt64
	err := s.db.QueryRow(`SELECT source, owner_id FROM scripts WHERE x = ? AND y = ? AND z = ?`, at.X, at.Y, at.Z).
		Scan(&saved.Source, &owner)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	saved.Owner = owner.Int64
	return s.start(at, saved, now), nil
}

// start loads saved into memory at `at`, making room first if too many are loaded.
func (s *Service) start(at world.Pos, saved Saved, now time.Time) *entry {
	if len(s.loaded) >= maxLoaded {
		s.evictIdle()
	}
	e := &entry{Saved: saved}
	if saved.Source != "" {
		run, err := script.Load(saved.Source, s.host(at, saved.Owner), now)
		if err != nil {
			e.Err = err.Error()
		}
		e.run = run
	}
	s.loaded[at] = e
	return e
}

// evictIdle unloads one script that waits on no timers.
func (s *Service) evictIdle() {
	for at, e := range s.loaded {
		if _, waiting := s.nextTimer(e); !waiting {
			s.unload(at)
			return
		}
	}
}

func (s *Service) nextTimer(e *entry) (time.Time, bool) {
	if e.run == nil {
		return time.Time{}, false
	}
	return e.run.NextTimer()
}

func (s *Service) unload(at world.Pos) {
	if e, ok := s.loaded[at]; ok && e.run != nil {
		e.run.Close()
	}
	delete(s.loaded, at)
}
