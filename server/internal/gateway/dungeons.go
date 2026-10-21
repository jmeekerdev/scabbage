package gateway

import (
	"fmt"
	"time"

	"scabbage/internal/drops"
	"scabbage/internal/logic"
	"scabbage/internal/loot"
	"scabbage/internal/mobs"
	"scabbage/internal/motion"
	"scabbage/internal/world"
)

// openChest gives s the loot in the chest at p and leaves it open; call with editMu held.
func (g *Gateway) openChest(s *session, p logic.Pos) error {
	if err := g.cfg.World.SetBlock(p.X, p.Y, p.Z, world.Chest+1); err != nil {
		return err
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.broadcastLocked(blockChanged(p.X, p.Y, p.Z, world.Chest+1))
	items := loot.Chest(g.rng)
	g.giveLocked(s, items, motion.Vec{X: float64(p.X) + 0.5, Y: float64(p.Y) + 1, Z: float64(p.Z) + 0.5})
	s.push(noticeMessage("🎁 You found " + loot.Describe(items) + "!"))
	return nil
}

// giveLocked puts items in s's inventory, dropping at `at` whatever doesn't fit.
func (g *Gateway) giveLocked(s *session, items []loot.Item, at motion.Vec) {
	a := g.players[s]
	if a == nil {
		return
	}
	now := time.Now()
	var spilled []drops.Drop
	var gone []uint64
	for _, it := range items {
		if left := a.inv.Add(it.Block, it.Count); left > 0 {
			d, old := g.drops.SpawnMany(it.Block, left, at, now)
			spilled, gone = append(spilled, d), append(gone, old...)
		}
	}
	s.push(inventoryMessage(&a.inv))
	g.changedLocked(spilled, gone)
}

// wakeBoss summons the altar at p's boss onto it, unless one is already up nearby.
func (g *Gateway) wakeBoss(s *session, p logic.Pos) {
	g.mu.Lock()
	defer g.mu.Unlock()
	kind := altarBoss(p)
	m, ok := g.cfg.Mobs.Summon(kind, motion.Vec{X: float64(p.X) + 0.5, Y: float64(p.Y) + 1, Z: float64(p.Z) + 0.5}, time.Now())
	if !ok {
		s.push(noticeMessage("A boss is already awake nearby."))
		return
	}
	g.broadcastLocked(mobsChanged([]mobs.Mob{m}, nil))
	g.broadcastLocked(noticeMessage(fmt.Sprintf("⚔️ The %s wakes up!", mobs.BossName(kind))))
}

// altarBoss is which boss an altar wakes; each altar always wakes the same one.
func altarBoss(p logic.Pos) mobs.Kind {
	if (p.X^p.Z)&1 == 0 {
		return mobs.KingSlime
	}
	return mobs.Golem
}

// bossBeatenLocked rewards whoever landed the last hit and tells everyone.
func (g *Gateway) bossBeatenLocked(s *session, boss mobs.Mob) {
	g.giveLocked(s, loot.Boss(g.rng), boss.Pos)
	g.broadcastLocked(noticeMessage(fmt.Sprintf("🏆 %s beat the %s!", s.player.Name, mobs.BossName(boss.Kind))))
}
