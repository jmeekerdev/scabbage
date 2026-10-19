package edit

import (
	"testing"
	"time"

	"scabbage/internal/world"
)

func TestDiggerWaitsForEachBlocksMiningTime(t *testing.T) {
	var d Digger
	t0 := time.Unix(100, 0)
	stone := time.Duration(world.Info(world.Stone).MineMs) * time.Millisecond
	if !d.Dig(world.Stone, t0) {
		t.Fatal("the first break should not wait")
	}
	if d.Dig(world.Stone, t0.Add(stone/4)) {
		t.Fatal("broke stone far too fast")
	}
	if !d.Dig(world.Stone, t0.Add(stone)) {
		t.Fatal("refused a fair dig")
	}
	if !d.Dig(world.Torch, t0.Add(stone)) {
		t.Fatal("torches pop off at once")
	}
}

func TestDiggerAllowsNetworkJitter(t *testing.T) {
	var d Digger
	t0 := time.Unix(100, 0)
	d.Dig(world.Dirt, t0)
	dirt := time.Duration(world.Info(world.Dirt).MineMs) * time.Millisecond
	if !d.Dig(world.Dirt, t0.Add(dirt*8/10)) {
		t.Fatal("a slightly early packet was refused")
	}
}
