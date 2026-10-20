package gateway

import (
	"time"

	"scabbage/internal/drops"
	"scabbage/internal/edit"
	"scabbage/internal/inventory"
	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/world"
)

func inventoryProto(inv *inventory.Inventory) *pb.Inventory {
	out := &pb.Inventory{Slots: make([]*pb.ItemStack, len(inv))}
	for i, st := range inv {
		out.Slots[i] = stackProto(st)
	}
	return out
}

func inventoryMessage(inv *inventory.Inventory) *pb.ServerMessage {
	return &pb.ServerMessage{Body: &pb.ServerMessage_Inventory{Inventory: inventoryProto(inv)}}
}

// itemsChanged lists drops that appeared or moved and ids that went; nil when there are none.
func itemsChanged(moved []drops.Drop, gone []uint64) *pb.ServerMessage {
	if len(moved) == 0 && len(gone) == 0 {
		return nil
	}
	out := &pb.ItemsChanged{Gone: gone}
	for _, d := range moved {
		out.Drops = append(out.Drops, &pb.ItemDrop{
			Id: d.ID, Block: uint32(d.Block), Pos: &pb.Vec3{X: float32(d.Pos.X), Y: float32(d.Pos.Y), Z: float32(d.Pos.Z)},
		})
	}
	return &pb.ServerMessage{Body: &pb.ServerMessage_ItemsChanged{ItemsChanged: out}}
}

// changedLocked tells everyone about drops that changed; call with g.mu held.
func (g *Gateway) changedLocked(moved []drops.Drop, gone []uint64) {
	if msg := itemsChanged(moved, gone); msg != nil {
		g.broadcastLocked(msg)
	}
}

// spend applies survival rules to an allowed edit: breaks take digging time, placements use up items.
func (g *Gateway) spend(s *session, r edit.Request, current world.Block, slot uint32, now time.Time) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	a := g.players[s]
	if a == nil || a.mode != survival {
		return true
	}
	if r.Block == world.Air {
		return a.digger.Dig(current, now)
	}
	if !a.inv.Take(int(slot), r.Block.Item()) {
		return false
	}
	s.push(inventoryMessage(&a.inv))
	return true
}

// broke drops what a survival player dug up and lets items above the hole fall.
func (g *Gateway) broke(s *session, r edit.Request, was world.Block, now time.Time) {
	g.mu.Lock()
	defer g.mu.Unlock()
	moved := g.drops.Resettle(r.X, r.Z, g.cfg.World.Boxes)
	var gone []uint64
	if a := g.players[s]; a != nil && a.mode == survival {
		var d drops.Drop
		d, gone = g.drops.Spawn(world.Drops(was), drops.Rest(r.X, r.Y, r.Z, g.cfg.World.Boxes), now)
		moved = append(moved, d)
	}
	g.changedLocked(moved, gone)
}

// collectLocked hands survival players the items they walk over and clears out old ones.
func (g *Gateway) collectLocked(now time.Time) {
	if g.drops.Len() == 0 {
		return
	}
	gone := g.drops.Expire(now)
	for s, a := range g.players {
		if a.mode != survival {
			continue
		}
		took := g.drops.Collect(a.pos, now, func(d drops.Drop) bool { return a.inv.Add(d.Block, 1) == 0 })
		if len(took) > 0 {
			s.push(inventoryMessage(&a.inv))
			gone = append(gone, took...)
		}
	}
	g.changedLocked(nil, gone)
}

// itemsLyingAroundLocked is every drop, for a player who just arrived; nil when there are none.
func (g *Gateway) itemsLyingAroundLocked() *pb.ServerMessage {
	return itemsChanged(g.drops.All(), nil)
}

// moveItem rearranges the player's inventory.
func (s *session) moveItem(m *pb.MoveItem) error {
	if s.console {
		return reject(pb.DisconnectCode_DISCONNECT_CODE_PROTOCOL, "the console has no pockets")
	}
	s.gw.mu.Lock()
	defer s.gw.mu.Unlock()
	a := s.gw.players[s]
	if a == nil || m.From >= inventory.Slots || m.To >= inventory.Slots || !a.inv.Move(int(m.From), int(m.To)) {
		return nil
	}
	s.push(inventoryMessage(&a.inv))
	return nil
}
