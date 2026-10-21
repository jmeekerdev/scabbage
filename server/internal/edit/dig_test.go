package edit

import (
	"testing"
	"time"
)

func TestDiggerWaitsForEachDigsTime(t *testing.T) {
	var d Digger
	t0 := time.Unix(100, 0)
	const stone = 500 * time.Millisecond
	if !d.Dig(stone, t0) {
		t.Fatal("the first break should not wait")
	}
	if d.Dig(stone, t0.Add(stone/4)) {
		t.Fatal("broke stone far too fast")
	}
	if !d.Dig(stone, t0.Add(stone)) {
		t.Fatal("refused a fair dig")
	}
	if !d.Dig(0, t0.Add(stone)) {
		t.Fatal("torches pop off at once")
	}
}

func TestDiggerAllowsNetworkJitter(t *testing.T) {
	var d Digger
	t0 := time.Unix(100, 0)
	const dirt = 200 * time.Millisecond
	d.Dig(dirt, t0)
	if !d.Dig(dirt, t0.Add(dirt*8/10)) {
		t.Fatal("a slightly early packet was refused")
	}
}
