package gateway

import (
	"slices"
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

// spend applies survival rules to a brush's allowed cells: a dig takes its mining time, all or
// nothing, and placements use up items until the slot runs out. It returns the cells that go ahead
// and those that don't.
func (g *Gateway) spend(s *session, b edit.Brush, cells []cellEdit, slot uint32, now time.Time) ([]cellEdit, []edit.Request) {
	g.mu.Lock()
	defer g.mu.Unlock()
	a := g.players[s]
	if a == nil || a.mode != survival || len(cells) == 0 {
		return cells, nil
	}
	if b.Block == world.Air {
		was := make([]world.Block, len(cells))
		for i, c := range cells {
			was[i] = c.was
		}
		if a.digger.Dig(edit.DigTime(was, b.Size), now) {
			return cells, nil
		}
		return nil, requests(cells)
	}
	n := 0
	for n < len(cells) && a.inv.Take(int(slot), b.Block.Item()) {
		n++
	}
	if n > 0 {
		s.push(inventoryMessage(&a.inv))
	}
	return cells[:n], requests(cells[n:])
}

func requests(cells []cellEdit) []edit.Request {
	out := make([]edit.Request, len(cells))
	for i, c := range cells {
		out[i] = c.Request
	}
	return out
}

// broke drops what a survival player dug up, one pile per kind of block in the middle of the
// brush, and lets items above the holes fall.
func (g *Gateway) broke(s *session, b edit.Brush, dug []cellEdit, now time.Time) {
	g.mu.Lock()
	defer g.mu.Unlock()
	var moved []drops.Drop
	columns := map[[2]int32]bool{}
	for _, c := range dug {
		if col := [2]int32{c.X, c.Z}; !columns[col] {
			columns[col] = true
			moved = append(moved, g.drops.Resettle(c.X, c.Z, g.cfg.World.Boxes)...)
		}
	}
	var gone []uint64
	if a := g.players[s]; a != nil && a.mode == survival {
		half, top := max(1, b.Size)/2, b.Y+max(1, b.Size)-1
		at := drops.Rest(b.X+half, top, b.Z+half, g.cfg.World.Boxes)
		for _, p := range piles(dug) {
			d, old := g.drops.SpawnMany(p.Block, int(p.Count), at, now)
			moved, gone = append(moved, d), append(gone, old...)
		}
	}
	g.changedLocked(moved, gone)
}

// piles counts what dug blocks drop, in the order each kind first appears.
func piles(dug []cellEdit) []inventory.Stack {
	var out []inventory.Stack
	for _, c := range dug {
		d := world.Drops(c.was)
		i := slices.IndexFunc(out, func(s inventory.Stack) bool { return s.Block == d })
		if i < 0 {
			out = append(out, inventory.Stack{Block: d})
			i = len(out) - 1
		}
		out[i].Count++
	}
	return out
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
		took := g.drops.Collect(a.pos, now, func(d drops.Drop) int { return d.Count - a.inv.Add(d.Block, d.Count) })
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
