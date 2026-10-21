// Package loot picks what chests and beaten bosses give.
package loot

import (
	"fmt"
	"math/rand/v2"
	"strings"

	"scabbage/internal/world"
)

// Item is some number of one block.
type Item struct {
	Block world.Block
	Count int
}

type roll struct {
	block    world.Block
	min, max int
}

// extras are what a chest may hold besides gold; two different ones are picked.
var extras = []roll{
	{world.Torch, 4, 8}, {world.Lamp, 1, 3}, {world.Glass, 4, 8}, {world.Brick, 4, 8},
	{world.Teleporter, 2, 2}, {world.NoteBlock, 1, 3}, {world.WhiteWool + 4, 4, 8}, {world.WhiteWool + 10, 4, 8},
}

const gemOdds = 4 // one chest in this many holds a gem

// Chest is a chest's loot: gold, two other things and now and then a gem.
func Chest(rng *rand.Rand) []Item {
	items := []Item{pick(rng, roll{world.GoldBlock, 1, 3})}
	for _, i := range rng.Perm(len(extras))[:2] {
		items = append(items, pick(rng, extras[i]))
	}
	if rng.IntN(gemOdds) == 0 {
		items = append(items, Item{world.GemBlock, 1})
	}
	return items
}

// Boss is a beaten boss's reward.
func Boss(rng *rand.Rand) []Item {
	return []Item{pick(rng, roll{world.GemBlock, 2, 3}), pick(rng, roll{world.GoldBlock, 3, 5})}
}

func pick(rng *rand.Rand, r roll) Item {
	return Item{r.block, r.min + rng.IntN(r.max-r.min+1)}
}

// Describe lists items for a chat message, like "torch ×6, gold block ×1".
func Describe(items []Item) string {
	parts := make([]string, len(items))
	for i, it := range items {
		parts[i] = fmt.Sprintf("%s ×%d", strings.ReplaceAll(it.Block.String(), "_", " "), it.Count)
	}
	return strings.Join(parts, ", ")
}
