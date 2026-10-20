package gateway

import (
	"context"
	"math"
	"time"

	"scabbage/internal/edit"
	"scabbage/internal/logic"
	"scabbage/internal/motion"
	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/world"
)

// buttonPress is how long a pressed button stays on.
const buttonPress = time.Second

// circuits remembers which buttons and plates are down; guarded by the gateway's editMu.
type circuits struct {
	letGo   map[logic.Pos]time.Time // pressed buttons and when they pop back up
	pressed map[logic.Pos]bool      // plates someone stands on
}

// useBlock presses a switch or button, or tells the sender what is really there.
func (s *session) useBlock(ctx context.Context, m *pb.UseBlock) error {
	if s.console {
		return reject(pb.DisconnectCode_DISCONNECT_CODE_PROTOCOL, "the console has no hands")
	}
	p := logic.Pos{X: m.GetPos().GetX(), Y: m.GetPos().GetY(), Z: m.GetPos().GetZ()}
	if s.edits.Take(time.Now()) {
		if used, err := s.gw.use(s, p); err != nil || used {
			return err
		}
	}
	current, _ := s.gw.cfg.World.Block(p.X, p.Y, p.Z)
	return s.write(ctx, blockChanged(p.X, p.Y, p.Z, current))
}

// use presses the block at p if it is usable and in reach; zones don't stop anyone pressing.
func (g *Gateway) use(s *session, p logic.Pos) (bool, error) {
	g.editMu.Lock()
	defer g.editMu.Unlock()
	current, err := g.cfg.World.Block(p.X, p.Y, p.Z)
	if err != nil || !edit.InReach(s.body.Pos(), p.X, p.Y, p.Z) {
		return false, nil
	}
	next, ok := logic.Use(current)
	if !ok {
		return false, nil
	}
	if next.Item() == world.Button {
		g.circuits.letGo[p] = time.Now().Add(buttonPress)
	}
	return true, g.switchLocked(p, current, next)
}

// switchLocked turns the logic block at p to next and powers its circuits; call with editMu held.
func (g *Gateway) switchLocked(p logic.Pos, current, next world.Block) error {
	if current == next {
		return nil
	}
	if err := g.cfg.World.SetBlock(p.X, p.Y, p.Z, next); err != nil {
		return err
	}
	g.broadcast(blockChanged(p.X, p.Y, p.Z, next))
	return g.powerLocked(p)
}

// powerLocked updates the circuits around p and tells everyone what lit up or went dark.
func (g *Gateway) powerLocked(p logic.Pos) error {
	changes, err := logic.Update(g.cfg.World, p)
	for _, c := range changes {
		g.broadcast(blockChanged(c.X, c.Y, c.Z, c.Block))
	}
	return err
}

// stepCircuits lets go of buttons whose time is up and presses the plates players stand on.
func (g *Gateway) stepCircuits(now time.Time) error {
	feet := g.bodies()
	g.editMu.Lock()
	defer g.editMu.Unlock()
	for p, at := range g.circuits.letGo {
		if now.Before(at) {
			continue
		}
		delete(g.circuits.letGo, p)
		if err := g.setIfLocked(p, world.Button+1, world.Button); err != nil {
			return err
		}
	}
	return g.pressPlatesLocked(feet)
}

// pressPlatesLocked turns on plates under someone's feet and off those nobody stands on.
func (g *Gateway) pressPlatesLocked(feet []motion.Vec) error {
	var now map[logic.Pos]bool
	for _, f := range feet {
		p := logic.Pos{X: int32(math.Floor(f.X)), Y: int32(math.Floor(f.Y)), Z: int32(math.Floor(f.Z))}
		if b, err := g.cfg.World.Block(p.X, p.Y, p.Z); err == nil && b.Item() == world.Plate {
			if now == nil {
				now = map[logic.Pos]bool{}
			}
			now[p] = true
		}
	}
	for p := range g.circuits.pressed {
		if !now[p] {
			if err := g.setIfLocked(p, world.Plate+1, world.Plate); err != nil {
				return err
			}
		}
	}
	for p := range now {
		if err := g.setIfLocked(p, world.Plate, world.Plate+1); err != nil {
			return err
		}
	}
	g.circuits.pressed = now
	return nil
}

// setIfLocked switches the block at p to next only if it is still was.
func (g *Gateway) setIfLocked(p logic.Pos, was, next world.Block) error {
	current, err := g.cfg.World.Block(p.X, p.Y, p.Z)
	if err != nil || current != was {
		return err
	}
	return g.switchLocked(p, current, next)
}
