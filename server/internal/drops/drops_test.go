package drops

import (
	"testing"
	"time"

	"scabbage/internal/motion"
	"scabbage/internal/world"
)

var t0 = time.Unix(1000, 0)

// floor is solid at y < 10, with a slab at 5,10,5.
func floor(x, y, z int32) []world.Box {
	if y < 10 {
		return []world.Box{world.FullBox}
	}
	if x == 5 && y == 10 && z == 5 {
		return world.Info(world.StoneSlab).Boxes()
	}
	return nil
}

func TestRestFallsToTheGround(t *testing.T) {
	if got := Rest(2, 14, 3, floor); got != (motion.Vec{X: 2.5, Y: 10, Z: 3.5}) {
		t.Fatalf("Rest = %+v", got)
	}
	if got := Rest(5, 12, 5, floor); got.Y != 10.5 {
		t.Fatalf("on a slab Y = %v, want 10.5", got.Y)
	}
	if got := Rest(0, 0, 0, func(int32, int32, int32) []world.Box { return nil }); got.Y != 0 {
		t.Fatalf("stopped at %v instead of the bottom of the world", got.Y)
	}
}

func TestCollectAfterAShortDelay(t *testing.T) {
	var f Field
	d, _ := f.Spawn(world.Dirt, motion.Vec{X: 1.5, Y: 10, Z: 1.5}, t0)
	feet := motion.Vec{X: 2, Y: 10, Z: 2}
	took := func(Drop) bool { return true }
	if gone := f.Collect(feet, t0, took); len(gone) != 0 {
		t.Fatal("picked up the instant it dropped")
	}
	if gone := f.Collect(motion.Vec{X: 9, Y: 10, Z: 9}, t0.Add(time.Second), took); len(gone) != 0 {
		t.Fatal("picked up from across the room")
	}
	gone := f.Collect(feet, t0.Add(time.Second), took)
	if len(gone) != 1 || gone[0] != d.ID || f.Len() != 0 {
		t.Fatalf("gone %v, %d left", gone, f.Len())
	}
}

func TestCollectLeavesWhatDoesNotFit(t *testing.T) {
	var f Field
	f.Spawn(world.Dirt, motion.Vec{X: 0.5, Y: 10, Z: 0.5}, t0)
	if gone := f.Collect(motion.Vec{Y: 10}, t0.Add(time.Second), func(Drop) bool { return false }); len(gone) != 0 || f.Len() != 1 {
		t.Fatal("a full inventory picked something up")
	}
}

func TestExpire(t *testing.T) {
	var f Field
	old, _ := f.Spawn(world.Dirt, motion.Vec{}, t0)
	f.Spawn(world.Log, motion.Vec{}, t0.Add(time.Minute))
	gone := f.Expire(t0.Add(Lifetime))
	if len(gone) != 1 || gone[0] != old.ID || f.Len() != 1 {
		t.Fatalf("gone %v, %d left", gone, f.Len())
	}
}

func TestSpawnPushesOutTheOldestWhenFull(t *testing.T) {
	var f Field
	first, _ := f.Spawn(world.Dirt, motion.Vec{}, t0)
	for range Max - 1 {
		f.Spawn(world.Dirt, motion.Vec{}, t0)
	}
	_, gone := f.Spawn(world.Log, motion.Vec{}, t0)
	if len(gone) != 1 || gone[0] != first.ID || f.Len() != Max {
		t.Fatalf("gone %v, %d left", gone, f.Len())
	}
}

func TestResettleDropsTheColumn(t *testing.T) {
	var f Field
	f.Spawn(world.Dirt, motion.Vec{X: 2.5, Y: 12, Z: 3.5}, t0)
	f.Spawn(world.Dirt, motion.Vec{X: 7.5, Y: 12, Z: 3.5}, t0)
	moved := f.Resettle(2, 3, floor)
	if len(moved) != 1 || moved[0].Pos.Y != 10 || f.All()[0].Pos.Y != 10 {
		t.Fatalf("moved %+v", moved)
	}
}
