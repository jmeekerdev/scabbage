package gateway

import (
	"context"
	"math"
	"slices"
	"time"

	"scabbage/internal/edit"
	"scabbage/internal/logic"
	"scabbage/internal/motion"
	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/weather"
	"scabbage/internal/world"
)

// buttonPress is how long a pressed button stays on.
const buttonPress = time.Second

// circuits remembers which buttons and plates are down; guarded by the gateway's editMu.
type circuits struct {
	letGo   map[logic.Pos]time.Time // pressed buttons and when they pop back up
	pressed map[logic.Pos]bool      // plates someone stands on
	crossed map[logic.Pos]bool      // tripwires someone is inside
	held    map[logic.Pos]bool      // doors kept open for someone inside, to close once they leave
	beat    int64                   // the quarter second timers last changed on
	sensed  time.Time               // when sensors last looked around
}

// senseEvery is how often sensors look around.
const senseEvery = time.Second / 4

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
	switch b, _ := g.cfg.World.Block(p.X, p.Y, p.Z); {
	case b.Item() == world.ScriptBlock:
		g.touchScriptLocked(s, p)
		return true, nil
	case b == world.Chest:
		return true, g.openChest(s, p)
	case b == world.BossAltar:
		g.wakeBoss(s, p)
		return true, nil
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

// Schedule asks for the block at p to be stepped after `after`, rounded to whole ticks.
func (w circuitWorld) Schedule(p logic.Pos, after time.Duration, flip bool) {
	ticks := int(math.Round(after.Seconds() * float64(w.g.cfg.Hub.TickRate())))
	w.g.cfg.World.Schedule(world.Pos(p), ticks, flip)
}

func (w circuitWorld) Transmitters() []logic.Pos {
	var out []logic.Pos
	for _, p := range w.g.cfg.World.Transmitters() {
		out = append(out, logic.Pos(p))
	}
	return out
}

func (w circuitWorld) CanPush(piston, at logic.Pos) bool {
	from, inFrom := w.g.cfg.Zones.At(piston.X, piston.Z)
	to, inTo := w.g.cfg.Zones.At(at.X, at.Z)
	return inFrom == inTo && from.Name == to.Name
}

// broadcastChanges sends everyone a circuit's changes in one message, however many there are,
// and tells script blocks their power changed; call with editMu held.
func (g *Gateway) broadcastChanges(changes []logic.Change) {
	if len(changes) == 0 {
		return
	}
	defer g.powerScriptsLocked(changes)
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

// stepCircuits lets go of buttons whose time is up, switches directed blocks, beats timers, lets
// sensors look around, runs scripts' timers, trips tripwires and presses the plates players stand on.
func (g *Gateway) stepCircuits(now time.Time) error {
	feet, creatures := g.bodies(), g.creatures()
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
	if err := g.switchDueLocked(); err != nil {
		return err
	}
	if err := g.beatLocked(now); err != nil {
		return err
	}
	if err := g.senseLocked(now, feet, creatures); err != nil {
		return err
	}
	g.reportScriptsLocked(g.scripts.RunTimers(now), nil)
	g.tellScriptsAboutRoundsLocked(now)
	if err := g.releaseDoorsLocked(feet); err != nil {
		return err
	}
	if err := g.crossTripwiresLocked(slices.Concat(feet, creatures)); err != nil {
		return err
	}
	return g.pressPlatesLocked(feet)
}

// switchDueLocked makes the switches directed blocks and receivers asked for by now.
func (g *Gateway) switchDueLocked() error {
	due := g.cfg.World.TakeScheduled()
	if len(due) == 0 {
		return nil
	}
	at := make([]logic.Due, len(due))
	for i, d := range due {
		at[i] = logic.Due{Pos: logic.Pos(d.Pos), Flip: d.Flip}
	}
	changes, err := logic.Step(circuitWorld{g}, at)
	g.broadcastChanges(changes)
	return err
}

// creatures is where every mob stands.
func (g *Gateway) creatures() []motion.Vec {
	g.mu.Lock()
	defer g.mu.Unlock()
	var out []motion.Vec
	for _, m := range g.cfg.Mobs.All() {
		out = append(out, m.Pos)
	}
	return out
}

// beatLocked turns every loaded timer on or off for this quarter second.
func (g *Gateway) beatLocked(now time.Time) error {
	beat := now.UnixMilli() / 250
	if beat == g.circuits.beat {
		return nil
	}
	g.circuits.beat = beat
	var timers []logic.Pos
	for _, p := range g.cfg.World.Ticking() {
		timers = append(timers, logic.Pos(p))
	}
	changes, err := logic.Beat(circuitWorld{g}, timers, beat)
	g.broadcastChanges(changes)
	return err
}

// senseLocked lets every loaded sensor look around for players, creatures, blocks, the time or weather.
func (g *Gateway) senseLocked(now time.Time, feet, creatures []motion.Vec) error {
	if now.Sub(g.circuits.sensed) < senseEvery {
		return nil
	}
	g.circuits.sensed = now
	var at []logic.Pos
	for _, p := range g.cfg.World.Ticking() {
		at = append(at, logic.Pos(p))
	}
	senses := logic.Senses{Players: spots(feet), Creatures: spots(creatures), Now: now, Wet: weather.At(now).Wet()}
	changes, err := logic.Sense(circuitWorld{g}, at, senses)
	g.broadcastChanges(changes)
	return err
}

func spots(vs []motion.Vec) []logic.Spot {
	out := make([]logic.Spot, len(vs))
	for i, v := range vs {
		out[i] = logic.Spot(v)
	}
	return out
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
		now = g.markIfLocked(now, blockAt(f), world.Plate)
	}
	err := g.holdLocked(g.circuits.pressed, now, world.Plate)
	g.circuits.pressed = now
	return err
}

// crossTripwiresLocked turns on tripwires anyone's body is in and off those nobody is in.
func (g *Gateway) crossTripwiresLocked(bodies []motion.Vec) error {
	var now map[logic.Pos]bool
	for _, b := range bodies {
		for _, p := range cellsOf(b) {
			now = g.markIfLocked(now, p, world.Tripwire)
		}
	}
	err := g.holdLocked(g.circuits.crossed, now, world.Tripwire)
	g.circuits.crossed = now
	return err
}

// markIfLocked adds p to marked if the block there is item, making the map only when needed.
func (g *Gateway) markIfLocked(marked map[logic.Pos]bool, p logic.Pos, item world.Block) map[logic.Pos]bool {
	if b, err := g.cfg.World.Block(p.X, p.Y, p.Z); err != nil || b.Item() != item {
		return marked
	}
	if marked == nil {
		marked = map[logic.Pos]bool{}
	}
	marked[p] = true
	return marked
}

// holdLocked switches the block `off` and its on state at each of now on, and off at those only in was.
func (g *Gateway) holdLocked(was, now map[logic.Pos]bool, off world.Block) error {
	for p := range was {
		if !now[p] {
			if err := g.setIfLocked(p, off+1, off); err != nil {
				return err
			}
		}
	}
	for p := range now {
		if err := g.setIfLocked(p, off, off+1); err != nil {
			return err
		}
	}
	return nil
}

// cellsOf is every block a body standing at feet is partly inside.
func cellsOf(feet motion.Vec) []logic.Pos {
	const inside = 1e-6
	lo := blockAt(motion.Vec{X: feet.X - motion.HalfWidth, Y: feet.Y, Z: feet.Z - motion.HalfWidth})
	hi := blockAt(motion.Vec{X: feet.X + motion.HalfWidth - inside, Y: feet.Y + motion.Height - inside, Z: feet.Z + motion.HalfWidth - inside})
	var out []logic.Pos
	for x := lo.X; x <= hi.X; x++ {
		for y := lo.Y; y <= hi.Y; y++ {
			for z := lo.Z; z <= hi.Z; z++ {
				out = append(out, logic.Pos{X: x, Y: y, Z: z})
			}
		}
	}
	return out
}

func blockAt(v motion.Vec) logic.Pos {
	return logic.Pos{X: int32(math.Floor(v.X)), Y: int32(math.Floor(v.Y)), Z: int32(math.Floor(v.Z))}
}

// padUnder is the pad any part of the feet stand in, or else the block the feet's middle is in.
func (g *Gateway) padUnder(feet motion.Vec) logic.Pos {
	mid := blockAt(feet)
	edge := motion.HalfWidth - 0.05 // just inside the body, so touching a pad's side doesn't count
	for _, dx := range []float64{0, -edge, edge} {
		for _, dz := range []float64{0, -edge, edge} {
			p := blockAt(motion.Vec{X: feet.X + dx, Y: feet.Y, Z: feet.Z + dz})
			if b, err := g.cfg.World.Block(p.X, p.Y, p.Z); err == nil && isPad(b) {
				return p
			}
		}
	}
	return mid
}

func isPad(b world.Block) bool {
	return b == world.Portal || b == world.Checkpoint || b == world.Teleporter
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

// stepOnPad sends a player who just stepped onto a teleporter to its partner or a portal to its
// game, and remembers checkpoints.
func (s *session) stepOnPad(feet motion.Vec) {
	p := s.gw.padUnder(feet)
	if p == s.pad {
		return
	}
	s.pad = p
	b, err := s.gw.cfg.World.Block(p.X, p.Y, p.Z)
	switch {
	case err == nil && b == world.Portal:
		s.gw.enterPortal(s, p)
		return
	case err == nil && b == world.Checkpoint:
		s.gw.reachCheckpoint(s, p)
		return
	}
	to, ok := s.gw.partner(p)
	if !ok {
		return
	}
	s.teleportTo(motion.Vec{X: float64(to.X) + 0.5, Y: float64(to.Y), Z: float64(to.Z) + 0.5}, nil)
}
