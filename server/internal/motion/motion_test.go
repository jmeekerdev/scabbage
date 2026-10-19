package motion

import (
	"math"
	"testing"
	"time"

	"scabbage/internal/world"
)

var t0 = time.Unix(1000, 0)

var (
	full = []world.Box{world.FullBox}
	slab = []world.Box{{Max: [3]float64{1, 0.5, 1}}}
)

// cubes turns a cell test into collision boxes.
func cubes(solid func(x, y, z int32) bool) BoxesFunc {
	return func(x, y, z int32) []world.Box {
		if solid(x, y, z) {
			return full
		}
		return nil
	}
}

// floor is solid below y=10; wallX adds a one-block wall at x=5; slabX a slab there instead.
var (
	floor = cubes(func(_, y, _ int32) bool { return y < 10 })
	wallX = cubes(func(x, y, _ int32) bool { return y < 10 || x == 5 })
)

func slabX(x, y, z int32) []world.Box {
	if x == 5 && y == 10 {
		return slab
	}
	return floor(x, y, z)
}

func newTracker() *Tracker { return NewTracker(DefaultLimits, Vec{0.5, 10, 0.5}, t0) }

func at(ms int) time.Time { return t0.Add(time.Duration(ms) * time.Millisecond) }

func TestAcceptsWalking(t *testing.T) {
	tr := newTracker()
	for i := 1; i <= 40; i++ {
		to := Vec{0.5 + float64(i)*0.25, 10, 0.5} // 5 blocks/s
		if !tr.Move(to, at(i*50), floor) {
			t.Fatalf("step %d rejected", i)
		}
	}
	if got := tr.Pos(); got.X != 10.5 {
		t.Fatalf("pos = %v", got)
	}
}

func TestRejectsMovingTooFast(t *testing.T) {
	tr := newTracker()
	accepted := 0
	for i := 1; i <= 40; i++ {
		if tr.Move(Vec{0.5 + float64(accepted+1)*2, 10, 0.5}, at(i*50), floor) { // 40 blocks/s
			accepted++
		}
	}
	// Two seconds at the limit plus one second of burst.
	if maxSteps := int(DefaultLimits.Horizontal * 3 / 2); accepted == 0 || accepted > maxSteps {
		t.Fatalf("accepted %d of 40 fast steps; want at most %d", accepted, maxSteps)
	}
}

func TestBudgetRefillsWithTime(t *testing.T) {
	tr := newTracker()
	if !tr.Move(Vec{14, 10, 0.5}, at(50), floor) {
		t.Fatal("a one-second burst should be allowed")
	}
	if tr.Move(Vec{20, 10, 0.5}, at(100), floor) {
		t.Fatal("the burst budget should be spent")
	}
	if !tr.Move(Vec{20, 10, 0.5}, at(1000), floor) {
		t.Fatal("budget should refill over time")
	}
}

func TestAllowsLongFallsButNotFastRises(t *testing.T) {
	tr := NewTracker(DefaultLimits, Vec{0.5, 80, 0.5}, t0)
	if !tr.Move(Vec{0.5, 30, 0.5}, at(1000), floor) {
		t.Fatal("falling 50 blocks in a second should be allowed")
	}
	if tr.Move(Vec{0.5, 60, 0.5}, at(2000), floor) {
		t.Fatal("rising 30 blocks in a second should be rejected")
	}
}

func TestRejectsEndingInsideABlock(t *testing.T) {
	tr := newTracker()
	if tr.Move(Vec{0.5, 9.5, 0.5}, at(50), floor) {
		t.Fatal("feet inside the floor should be rejected")
	}
	if tr.Move(Vec{4.75, 10, 0.5}, at(1000), wallX) {
		t.Fatal("body overlapping the wall should be rejected")
	}
	if !tr.Move(Vec{4.7, 10, 0.5}, at(1000), wallX) {
		t.Fatal("touching the wall should be allowed")
	}
}

func TestRejectsPassingThroughAWall(t *testing.T) {
	tr := NewTracker(DefaultLimits, Vec{4.5, 10, 0.5}, t0)
	if tr.Move(Vec{6.5, 10, 0.5}, at(1000), wallX) {
		t.Fatal("jumping through the wall should be rejected")
	}
}

func TestRejectedMoveKeepsPositionAndBudget(t *testing.T) {
	tr := newTracker()
	tr.Move(Vec{0.5, 9, 0.5}, at(50), floor)
	if tr.Pos() != (Vec{0.5, 10, 0.5}) {
		t.Fatalf("pos = %v", tr.Pos())
	}
	if !tr.Move(Vec{14, 10, 0.5}, at(50), floor) {
		t.Fatal("a rejected move should not spend budget")
	}
}

func TestRejectsNonFinite(t *testing.T) {
	tr := newTracker()
	for _, v := range []Vec{{math.NaN(), 10, 0}, {0, math.Inf(1), 0}} {
		if tr.Move(v, at(50), floor) {
			t.Fatalf("%v accepted", v)
		}
	}
}

func TestResetMovesWithoutChecks(t *testing.T) {
	tr := newTracker()
	tr.Reset(Vec{100, 50, 100}, at(10))
	if tr.Pos() != (Vec{100, 50, 100}) || !tr.Move(Vec{101, 50, 100}, at(60), floor) {
		t.Fatal("reset should move the player and keep tracking from there")
	}
}

func TestStandsOnSlabs(t *testing.T) {
	tr := NewTracker(DefaultLimits, Vec{4.5, 10, 0.5}, t0)
	if tr.Move(Vec{5.5, 10.2, 0.5}, at(1000), slabX) {
		t.Fatal("feet inside the slab should be rejected")
	}
	if !tr.Move(Vec{5.5, 10.5, 0.5}, at(1000), slabX) {
		t.Fatal("stepping up onto the slab should be allowed")
	}
}
