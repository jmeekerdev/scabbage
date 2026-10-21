package gateway

import (
	"testing"

	"scabbage/internal/motion"
	"scabbage/internal/world"
)

func TestNearAnyKeepsColumnsInViewOfSomeone(t *testing.T) {
	keep := nearAny([]motion.Vec{{X: 10, Y: 200, Z: 10}, {X: -1000, Y: 50, Z: 5}}, 2)
	cases := []struct {
		p    world.ChunkPos
		want bool
	}{
		{world.ChunkPos{X: 0, Y: 0, Z: 0}, true},
		{world.ChunkPos{X: 2, Y: 11, Z: -2}, true},
		{world.ChunkPos{X: 3, Y: 6, Z: 0}, false},
		{world.ChunkPos{X: -32, Y: 1, Z: 1}, true}, // the second player's column
		{world.ChunkPos{X: -16, Y: 1, Z: 0}, false},
	}
	for _, c := range cases {
		if got := keep(c.p); got != c.want {
			t.Errorf("keep(%v) = %v, want %v", c.p, got, c.want)
		}
	}
}

func TestNearNoOneKeepsNothing(t *testing.T) {
	if nearAny(nil, 12)(world.ChunkPos{}) {
		t.Fatal("with no players every chunk can go")
	}
}
