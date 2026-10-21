package gateway

import (
	"context"
	"math"
	"time"

	"scabbage/internal/edit"
	"scabbage/internal/logic"
	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/world"
)

// editBlock applies an allowed edit for everyone, or tells the sender what is really there.
func (s *session) editBlock(ctx context.Context, m *pb.EditBlock) error {
	if s.console {
		return reject(pb.DisconnectCode_DISCONNECT_CODE_PROTOCOL, "the console has no hands")
	}
	p := m.GetPos()
	r := edit.Request{X: p.GetX(), Y: p.GetY(), Z: p.GetZ(), Block: world.Block(min(m.Block, math.MaxUint16))}
	if s.edits.Take(time.Now()) {
		if applied, err := s.gw.edit(s, r, m.Slot); err != nil || applied {
			return err
		}
	}
	current, _ := s.gw.cfg.World.Block(r.X, r.Y, r.Z) // air outside the world
	return s.write(ctx, blockChanged(r.X, r.Y, r.Z, current))
}

// edit changes the block if allowed and broadcasts the change, all under editMu.
func (g *Gateway) edit(s *session, r edit.Request, slot uint32) (bool, error) {
	g.editMu.Lock()
	defer g.editMu.Unlock()
	now := time.Now()
	current, err := g.cfg.World.Block(r.X, r.Y, r.Z)
	if err != nil || !g.cfg.Zones.CanBuild(s.player.ID, s.player.Admin, r.X, r.Z) || !edit.Allowed(s.body.Pos(), r, current, g.bodies()) || !g.spend(s, r, current, slot, now) {
		return false, nil
	}
	if err := g.cfg.World.SetBlock(r.X, r.Y, r.Z, r.Block); err != nil {
		return false, err
	}
	g.broadcast(blockChanged(r.X, r.Y, r.Z, r.Block))
	if r.Block == world.Air {
		g.broke(s, r, current, now)
	}
	if current.Item() == world.ScriptBlock {
		if err := g.scripts.Delete(world.Pos{X: r.X, Y: r.Y, Z: r.Z}); err != nil {
			return true, err
		}
	}
	if logic.Involves(current) || logic.Involves(r.Block) {
		return true, g.powerLocked(logic.Pos{X: r.X, Y: r.Y, Z: r.Z})
	}
	return true, nil
}

func blockChanged(x, y, z int32, b world.Block) *pb.ServerMessage {
	return &pb.ServerMessage{Body: &pb.ServerMessage_BlockChanged{BlockChanged: &pb.BlockChanged{
		Pos: &pb.BlockPos{X: x, Y: y, Z: z}, Block: uint32(b),
	}}}
}
