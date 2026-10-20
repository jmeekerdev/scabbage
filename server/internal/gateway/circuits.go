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
	held    map[logic.Pos]bool      // doors kept open for someone inside, to close once they leave
	beat    int64                   // the second timers last changed on
}

// useBlock presses a usable block, or tells the sender what is really there.
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
	if !edit.InReach(s.body.Pos(), p.X, p.Y, p.Z) {
		return false, nil
	}
	changes, used, err := logic.Use(circuitWorld{g}, p)
	g.broadcastChanges(changes)
	if b, _ := g.cfg.World.Block(p.X, p.Y, p.Z); used && b == world.Button+1 {
		g.circuits.letGo[p] = time.Now().Add(buttonPress)
	}
	return used, err
}

// circuitWorld is the world as circuits see it: pistons may not push into or out of zones.
type circuitWorld struct{ g *Gateway }

func (w circuitWorld) Block(x, y, z int32) (world.Block, error) { return w.g.cfg.World.Block(x, y, z) }

func (w circuitWorld) SetBlock(x, y, z int32, b world.Block) error {
	return w.g.cfg.World.SetBlock(x, y, z, b)
}

// Occupied tells circuits whether a player is in p, remembering it so the door there closes later.
func (w circuitWorld) Occupied(p logic.Pos) bool {
	in := edit.Occupied(p.X, p.Y, p.Z, w.g.bodies())
	if in {
		w.g.circuits.held[p] = true
	}
	return in
}

func (w circuitWorld) CanPush(piston, at logic.Pos) bool {
	from, inFrom := w.g.cfg.Zones.At(piston.X, piston.Z)
	to, inTo := w.g.cfg.Zones.At(at.X, at.Z)
	return inFrom == inTo && from.Name == to.Name
}

// broadcastChanges sends everyone a circuit's changes in one message, however many there are.
func (g *Gateway) broadcastChanges(changes []logic.Change) {
	if len(changes) == 0 {
		return
	}
	batch := &pb.BlockBatch{Blocks: make([]*pb.BlockChanged, 0, len(changes))}
	for _, c := range changes {
		batch.Blocks = append(batch.Blocks, blockChanged(c.X, c.Y, c.Z, c.Block).GetBlockChanged())
	}
	g.broadcast(&pb.ServerMessage{Body: &pb.ServerMessage_BlockBatch{BlockBatch: batch}})
}

// powerLocked updates the circuits around p and tells everyone what changed; call with editMu held.
func (g *Gateway) powerLocked(p logic.Pos) error {
	changes, err := logic.Update(circuitWorld{g}, p)
	g.broadcastChanges(changes)
	return err
}

// stepCircuits lets go of buttons whose time is up, beats timers and presses the plates players stand on.
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
	if err := g.beatLocked(now); err != nil {
		return err
	}
	if err := g.releaseDoorsLocked(feet); err != nil {
		return err
	}
	return g.pressPlatesLocked(feet)
}

// beatLocked turns every loaded timer on for one second and off for the next.
func (g *Gateway) beatLocked(now time.Time) error {
	beat := now.Unix()
	if beat == g.circuits.beat {
		return nil
	}
	g.circuits.beat = beat
	var timers []logic.Pos
	for _, p := range g.cfg.World.Ticking() {
		timers = append(timers, logic.Pos(p))
	}
	changes, err := logic.Beat(circuitWorld{g}, timers, beat%2 == 0)
	g.broadcastChanges(changes)
	return err
}

// releaseDoorsLocked lets held doors follow their power again once nobody is inside.
func (g *Gateway) releaseDoorsLocked(feet []motion.Vec) error {
	for p := range g.circuits.held {
		if edit.Occupied(p.X, p.Y, p.Z, feet) {
			continue
		}
		delete(g.circuits.held, p)
		if err := g.powerLocked(p); err != nil {
			return err
		}
	}
	return nil
}

// pressPlatesLocked turns on plates under someone's feet and off those nobody stands on.
func (g *Gateway) pressPlatesLocked(feet []motion.Vec) error {
	var now map[logic.Pos]bool
	for _, f := range feet {
		p := blockAt(f)
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

func blockAt(v motion.Vec) logic.Pos {
	return logic.Pos{X: int32(math.Floor(v.X)), Y: int32(math.Floor(v.Y)), Z: int32(math.Floor(v.Z))}
}

// setIfLocked switches the block at p to next only if it is still was.
func (g *Gateway) setIfLocked(p logic.Pos, was, next world.Block) error {
	current, err := g.cfg.World.Block(p.X, p.Y, p.Z)
	if err != nil || current != was {
		return err
	}
	changes, err := logic.Set(circuitWorld{g}, p, next)
	g.broadcastChanges(changes)
	return err
}

// partner is where the teleporter at p sends players, if p is one joined to another.
func (g *Gateway) partner(p logic.Pos) (logic.Pos, bool) {
	g.editMu.Lock()
	defer g.editMu.Unlock()
	if b, err := g.cfg.World.Block(p.X, p.Y, p.Z); err != nil || b != world.Teleporter {
		return p, false
	}
	to, ok, err := logic.Partner(circuitWorld{g}, p)
	if err != nil {
		g.cfg.Logger.Error("teleporter", "err", err)
	}
	return to, ok && err == nil
}

// stepOnPad sends a player who just stepped onto a teleporter to its partner.
func (s *session) stepOnPad(feet motion.Vec) {
	p := blockAt(feet)
	if p == s.pad {
		return
	}
	s.pad = p
	to, ok := s.gw.partner(p)
	if !ok {
		return
	}
	s.pad = to
	s.teleportTo(motion.Vec{X: float64(to.X) + 0.5, Y: float64(to.Y), Z: float64(to.Z) + 0.5})
}
