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

// cellEdit is one block of a brush that may change, and what it was.
type cellEdit struct {
	edit.Request
	was world.Block
}

// editBlock applies the allowed blocks of a brush for everyone, and tells the sender what is
// really in the rest.
func (s *session) editBlock(ctx context.Context, m *pb.EditBlock) error {
	if s.console {
		return reject(pb.DisconnectCode_DISCONNECT_CODE_PROTOCOL, "the console has no hands")
	}
	p := m.GetPos()
	b := edit.Brush{X: p.GetX(), Y: p.GetY(), Z: p.GetZ(), Size: int32(min(m.Size, edit.MaxBrush)),
		Block: world.Block(min(m.Block, math.MaxUint16))}
	refused := b.Cells()
	if s.edits.Take(time.Now()) {
		var err error
		if refused, err = s.gw.edit(s, b, m.Slot); err != nil {
			return err
		}
	}
	if len(refused) == 0 {
		return nil
	}
	return s.write(ctx, changesMessage(s.gw.actual(refused)))
}

// edit makes the brush's allowed changes and broadcasts them, all under editMu, returning the
// blocks it left alone.
func (g *Gateway) edit(s *session, b edit.Brush, slot uint32) ([]edit.Request, error) {
	g.editMu.Lock()
	defer g.editMu.Unlock()
	if !b.InReach(s.body.Pos()) {
		return b.Cells(), nil
	}
	now := time.Now()
	cells, refused := g.editable(s, b.Cells())
	cells, unpaid := g.spend(s, b, cells, slot, now)
	refused = append(refused, unpaid...)
	changes := make([]world.Change, 0, len(cells))
	for _, c := range cells {
		if err := g.cfg.World.SetBlock(c.X, c.Y, c.Z, c.Block); err != nil {
			return refused, err
		}
		changes = append(changes, world.Change{Pos: world.Pos{X: c.X, Y: c.Y, Z: c.Z}, Block: c.Block, Was: c.was})
	}
	if len(changes) == 0 {
		return refused, nil
	}
	g.broadcast(changesMessage(changes))
	if b.Block == world.Air {
		g.broke(s, b, cells, now)
	}
	return refused, g.afterEdits(cells)
}

// editable splits a brush's cells into those s may change and those it may not.
func (g *Gateway) editable(s *session, cells []edit.Request) (ok []cellEdit, refused []edit.Request) {
	bodies := g.bodies()
	for _, r := range cells {
		was, err := g.cfg.World.Block(r.X, r.Y, r.Z)
		if err == nil && g.cfg.Zones.CanBuild(s.player.ID, s.player.Admin, r.X, r.Z) && edit.Allowed(r, was, bodies) {
			ok = append(ok, cellEdit{r, was})
		} else {
			refused = append(refused, r)
		}
	}
	return ok, refused
}

// afterEdits forgets broken script blocks and updates the circuits the edits touched.
func (g *Gateway) afterEdits(cells []cellEdit) error {
	for _, c := range cells {
		if c.was.Item() == world.ScriptBlock {
			if err := g.scripts.Delete(world.Pos{X: c.X, Y: c.Y, Z: c.Z}); err != nil {
				return err
			}
		}
		if logic.Involves(c.was) || logic.Involves(c.Block) {
			if err := g.powerLocked(logic.Pos{X: c.X, Y: c.Y, Z: c.Z}); err != nil {
				return err
			}
		}
	}
	return nil
}

// actual is what is really in each of cells now; air outside the world.
func (g *Gateway) actual(cells []edit.Request) []world.Change {
	out := make([]world.Change, len(cells))
	for i, r := range cells {
		b, _ := g.cfg.World.Block(r.X, r.Y, r.Z)
		out[i] = world.Change{Pos: world.Pos{X: r.X, Y: r.Y, Z: r.Z}, Block: b}
	}
	return out
}

// changesMessage tells of one change as BlockChanged, and of more as one BlockBatch.
func changesMessage(changes []world.Change) *pb.ServerMessage {
	if len(changes) == 1 {
		c := changes[0]
		return blockChanged(c.X, c.Y, c.Z, c.Block)
	}
	batch := &pb.BlockBatch{Blocks: make([]*pb.BlockChanged, 0, len(changes))}
	for _, c := range changes {
		batch.Blocks = append(batch.Blocks, blockChanged(c.X, c.Y, c.Z, c.Block).GetBlockChanged())
	}
	return &pb.ServerMessage{Body: &pb.ServerMessage_BlockBatch{BlockBatch: batch}}
}

func blockChanged(x, y, z int32, b world.Block) *pb.ServerMessage {
	return &pb.ServerMessage{Body: &pb.ServerMessage_BlockChanged{BlockChanged: &pb.BlockChanged{
		Pos: &pb.BlockPos{X: x, Y: y, Z: z}, Block: uint32(b),
	}}}
}
