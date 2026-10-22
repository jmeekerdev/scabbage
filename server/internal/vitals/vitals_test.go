package vitals

import (
	"testing"
	"time"

	"scabbage/internal/world"
)

var t0 = time.Unix(1000, 0)

// fall drops from y=20 by blocks through the air and lands with the given footing.
const m = world.Meter

// fall drops a player the given meters from y = 20 m onto land.
func fall(s *State, meters float64, land Footing) int {
	s.Fall(20*m, Ground)
	for y := 20.0; y > 20-meters; y-- {
		s.Fall(y*m, Air)
	}
	return s.Fall((20-meters)*m, land)
}

func TestShortFallsAreSafe(t *testing.T) {
	s := New(t0)
	for _, h := range []float64{1, 2, 3} {
		if d := fall(s, h, Ground); d != 0 {
			t.Errorf("falling %v m hurt %d", h, d)
		}
	}
}

func TestLongFallsHurtMoreTheHigherTheyAre(t *testing.T) {
	s := New(t0)
	if d := fall(s, 4, Ground); d != 1 {
		t.Errorf("4 m hurt %d, want 1", d)
	}
	if d := fall(s, 10, Ground); d != 7 {
		t.Errorf("10 m hurt %d, want 7", d)
	}
}

func TestJumpsCountFromTheirHighestPoint(t *testing.T) {
	s := New(t0)
	s.Fall(10*m, Ground)
	s.Fall(11.2*m, Air)
	if d := s.Fall(6*m, Ground); d != 2 {
		t.Errorf("hurt %d, want 2", d)
	}
}

func TestWaterBreaksAFall(t *testing.T) {
	s := New(t0)
	if d := fall(s, 30, Water); d != 0 {
		t.Errorf("landing in water hurt %d", d)
	}
	if d := s.Fall(9, Ground); d != 0 {
		t.Errorf("climbing out of water hurt %d", d)
	}
}

func TestFallsStartOverAfterATeleport(t *testing.T) {
	s := New(t0)
	s.Fall(100, Air)
	s.Settle()
	if d := s.Fall(5, Ground); d != 0 {
		t.Errorf("hurt %d after a teleport", d)
	}
}

func TestHurtFaintsAtZero(t *testing.T) {
	s := New(t0)
	if s.Hurt(Max-1, t0) {
		t.Fatal("fainted with health left")
	}
	if !s.Hurt(5, t0) || s.Health != 0 {
		t.Fatalf("health = %d, want to faint at 0", s.Health)
	}
	s.Revive(t0)
	if s.Health != Max {
		t.Fatalf("revived with %d", s.Health)
	}
}

func TestHealsSlowlyOnceSafe(t *testing.T) {
	s := New(t0)
	s.Hurt(4, t0)
	if s.Heal(t0.Add(healDelay - time.Millisecond)) {
		t.Fatal("healed too soon after getting hurt")
	}
	now := t0.Add(healDelay)
	if !s.Heal(now) || s.Health != Max-3 {
		t.Fatalf("health = %d after waiting", s.Health)
	}
	if s.Heal(now.Add(healEvery - time.Millisecond)) {
		t.Fatal("healed again too soon")
	}
	if !s.Heal(now.Add(healEvery)) || s.Health != Max-2 {
		t.Fatalf("health = %d", s.Health)
	}
}

func TestFullHealthNeedsNoHealing(t *testing.T) {
	if New(t0).Heal(t0.Add(time.Hour)) {
		t.Fatal("healed past full")
	}
}
