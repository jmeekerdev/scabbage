package loot

import (
	"math/rand/v2"
	"testing"

	"scabbage/internal/world"
)

func TestChestsHoldGoldAndTwoOtherThings(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	gems := 0
	for range 200 {
		items := Chest(rng)
		seen := map[world.Block]bool{}
		for _, it := range items {
			if seen[it.Block] || it.Count < 1 || it.Count > 8 || !world.Placeable(it.Block) {
				t.Fatalf("bad loot %v", items)
			}
			seen[it.Block] = true
		}
		if !seen[world.GoldBlock] || len(items) < 3 {
			t.Fatalf("loot %v has no gold or too little", items)
		}
		if seen[world.GemBlock] {
			gems++
		}
	}
	if gems == 0 || gems > 100 {
		t.Errorf("%d of 200 chests had a gem", gems)
	}
}

func TestBossesDropGems(t *testing.T) {
	items := Boss(rand.New(rand.NewPCG(3, 4)))
	if len(items) != 2 || items[0].Block != world.GemBlock || items[0].Count < 2 {
		t.Fatalf("boss loot %v", items)
	}
}

func TestDescribe(t *testing.T) {
	got := Describe([]Item{{world.Torch, 6}, {world.GoldBlock, 1}})
	if got != "torch ×6, gold block ×1" {
		t.Fatalf("got %q", got)
	}
}
