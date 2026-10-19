package edit

import (
	"testing"
	"time"

	"scabbage/internal/motion"
	"scabbage/internal/world"
)

// The player stands at 0.5,10,0.5, so their eye is at y=11.62.
var feet = motion.Vec{X: 0.5, Y: 10, Z: 0.5}

func place(x, y, z int32, b world.Block) Request { return Request{X: x, Y: y, Z: z, Block: b} }

func TestBreaksSolidBlocksInReach(t *testing.T) {
	if !Allowed(feet, place(2, 9, 0, world.Air), world.Stone, nil) {
		t.Fatal("breaking nearby stone should be allowed")
	}
}

func TestCannotBreakAirWaterOrBedrock(t *testing.T) {
	for _, b := range []world.Block{world.Air, world.Water, world.Bedrock} {
		if Allowed(feet, place(2, 9, 0, world.Air), b, nil) {
			t.Errorf("breaking %v allowed", b)
		}
	}
}

func TestReachIsLimited(t *testing.T) {
	if Allowed(feet, place(7, 11, 0, world.Air), world.Stone, nil) {
		t.Fatal("a block 7 blocks away is out of reach")
	}
	if !Allowed(feet, place(5, 11, 0, world.Air), world.Stone, nil) {
		t.Fatal("a block 5 blocks away is in reach")
	}
}

func TestPlacesOnlyIntoAirOrWater(t *testing.T) {
	if !Allowed(feet, place(2, 10, 0, world.Planks), world.Air, nil) {
		t.Fatal("placing into air should be allowed")
	}
	if !Allowed(feet, place(2, 10, 0, world.Planks), world.Water, nil) {
		t.Fatal("placing into water should be allowed")
	}
	if Allowed(feet, place(2, 10, 0, world.Planks), world.Stone, nil) {
		t.Fatal("placing into stone should be rejected")
	}
}

func TestCannotPlaceBedrockOrUnknownBlocks(t *testing.T) {
	for _, b := range []world.Block{world.Bedrock, world.Block(999)} {
		if Allowed(feet, place(2, 10, 0, b), world.Air, nil) {
			t.Errorf("placing %d allowed", b)
		}
	}
}

func TestSolidBlocksMayNotOverlapPlayers(t *testing.T) {
	others := []motion.Vec{feet, {X: 3.5, Y: 10, Z: 0.5}}
	if Allowed(feet, place(0, 11, 0, world.Planks), world.Air, others) {
		t.Fatal("placing inside the player's own body should be rejected")
	}
	if Allowed(feet, place(3, 10, 0, world.Planks), world.Air, others) {
		t.Fatal("placing inside another player should be rejected")
	}
	if !Allowed(feet, place(0, 9, 0, world.Planks), world.Air, others) {
		t.Fatal("placing right under the player's feet should be allowed")
	}
	if !Allowed(feet, place(0, 11, 0, world.Water), world.Air, others) {
		t.Fatal("water may overlap players")
	}
}

func TestRateRefillsOverTime(t *testing.T) {
	t0 := time.Unix(100, 0)
	r := NewRate(t0)
	n := 0
	for r.Take(t0) {
		n++
	}
	if n != burst {
		t.Fatalf("took %d at once, want %d", n, burst)
	}
	if !r.Take(t0.Add(time.Second / perSecond)) {
		t.Fatal("rate should refill")
	}
}
