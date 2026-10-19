// Package vitals tracks a player's health: hurting from falls, fainting and slowly healing.
package vitals

import (
	"math"
	"time"
)

// Max is full health, shown as ten hearts.
const Max = 20

const (
	safeFall  = 3.0 // blocks anyone can drop unhurt
	healDelay = 5 * time.Second
	healEvery = 3 * time.Second
	noFallYet = -math.MaxFloat64
)

// Footing is what holds the player up after a move.
type Footing int

const (
	Air Footing = iota
	Ground
	Water
)

// State is one player's health and the fall they may be in.
type State struct {
	Health   int
	peak     float64 // highest point of the current fall
	hurtAt   time.Time
	healedAt time.Time
}

func New(now time.Time) *State {
	return &State{Health: Max, peak: noFallYet, hurtAt: now, healedAt: now}
}

// Fall follows a move to height y and returns the damage if it ends a fall.
func (s *State) Fall(y float64, f Footing) int {
	if f == Air {
		s.peak = math.Max(s.peak, y)
		return 0
	}
	drop := s.peak - y
	s.peak = y // a later fall starts from here
	if f == Water || drop <= safeFall {
		return 0
	}
	return int(drop - safeFall)
}

// Settle forgets the current fall, e.g. after a teleport.
func (s *State) Settle() { s.peak = noFallYet }

// Hurt takes away health and reports whether the player fainted.
func (s *State) Hurt(n int, now time.Time) bool {
	s.Health = max(0, s.Health-n)
	s.hurtAt = now
	return s.Health == 0
}

// Revive restores full health after fainting.
func (s *State) Revive(now time.Time) {
	s.Health, s.hurtAt, s.peak = Max, now, noFallYet
}

// Heal adds one point when the player has been safe long enough; it reports any change.
func (s *State) Heal(now time.Time) bool {
	if s.Health >= Max || now.Sub(s.hurtAt) < healDelay || now.Sub(s.healedAt) < healEvery {
		return false
	}
	s.Health++
	s.healedAt = now
	return true
}
