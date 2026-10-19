package gateway

import (
	"scabbage/internal/crafting"
	"scabbage/internal/inventory"
	pb "scabbage/internal/pb/scabbage/v1"
)

func stackProto(s inventory.Stack) *pb.ItemStack {
	return &pb.ItemStack{Block: uint32(s.Block), Count: uint32(s.Count)}
}

func recipesProto(book []crafting.Recipe) []*pb.Recipe {
	out := make([]*pb.Recipe, len(book))
	for i, r := range book {
		out[i] = &pb.Recipe{Make: stackProto(r.Make)}
		for _, in := range r.From {
			out[i].From = append(out[i].From, stackProto(in))
		}
	}
	return out
}

// craft makes a recipe from a survival player's inventory; anything it can't make is ignored.
func (s *session) craft(m *pb.Craft) error {
	if s.console {
		return reject(pb.DisconnectCode_DISCONNECT_CODE_PROTOCOL, "the console has no pockets")
	}
	book := s.gw.cfg.Recipes
	s.gw.mu.Lock()
	defer s.gw.mu.Unlock()
	a := s.gw.players[s]
	if a == nil || a.mode != survival || m.Recipe >= uint32(len(book)) || !book[m.Recipe].Craft(&a.inv) {
		return nil
	}
	a.invDirty = true
	s.push(inventoryMessage(&a.inv))
	return nil
}
