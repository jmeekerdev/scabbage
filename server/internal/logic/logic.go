// Package logic powers circuits: switches, buttons, plates, timers and counters send power down
// wire to lamps, doors, pistons and note blocks.
package logic

import (
	"errors"
	"slices"
	"time"

	"scabbage/internal/world"
)

const (
	// MaxWires bounds how much wire one update walks, so a giant circuit can't stall the server.
	MaxWires = 4096
	// MaxPush is the longest row of blocks a piston moves: twelve meters.
	MaxPush = 12 * world.Meter
	// maxRounds bounds how many counters one update may switch on or off in a chain.
	maxRounds = 16
	// lastCount is where a counter starts giving power and turns to count back down to 0.
	lastCount = 9
	// countCycle is how many pulses take a counter up to lastCount and back to 0.
	countCycle = 2 * lastCount
	notes      = 8
)

// Pos is a block position; it converts to and from world.Pos.
type Pos struct{ X, Y, Z int32 }

// World is the block storage circuits read and change.
type World interface {
	Block(x, y, z int32) (world.Block, error)
	SetBlock(x, y, z int32, b world.Block) error
	CanPush(piston, at Pos) bool // whether a piston may move the block at `at`, e.g. outside zones
	Occupied(p Pos) bool         // whether someone stands in p, so a door there mustn't close
	// Schedule asks for the block at p to be stepped after a wait, at least till the next tick,
	// or flipped then if flip is set; a block already waiting keeps its wait, unless a longer
	// judged one is asked.
	Schedule(p Pos, after time.Duration, flip bool)
	// Transmitters lists the loaded transmitters; whenever one changes, every receiver is stepped.
	Transmitters() []Pos
}

// Due is a block whose wait is over; Flip means it switches whatever its input is by now.
type Due struct {
	Pos
	Flip bool
}

// Change is a block a circuit update switched on or off.
type Change struct {
	Pos
	Block world.Block
}

var sides = [6]Pos{{X: 1}, {X: -1}, {Y: 1}, {Y: -1}, {Z: 1}, {Z: -1}}

// pushes is the way each piston facing pushes: north, east, south, west, up, down.
var pushes = [6]Pos{{Z: -1}, {X: 1}, {Z: 1}, {X: -1}, {Y: 1}, {Y: -1}}

func add(p, d Pos) Pos { return Pos{X: p.X + d.X, Y: p.Y + d.Y, Z: p.Z + d.Z} }

func isWire(b world.Block) bool     { return world.Info(b).Conducts }
func isDirected(b world.Block) bool { return world.Info(b).Directed }
func isOn(b world.Block) bool       { return world.Info(b).Powered }
func state(b world.Block) int       { return int(world.Info(b).Variant / 2) }
func facing(b world.Block) int      { return state(b) % len(pushes) }

// isSource reports whether b sends power into the wire and machines beside it.
func isSource(b world.Block) bool {
	switch b.Item() {
	case world.Switch, world.Button, world.Plate, world.Timer, world.Timers, world.Tripwire, world.Receiver,
		world.PlayerDetector, world.CreatureDetector, world.Clock, world.BlockDetector,
		world.WeatherSensor:
		return isOn(b)
	case world.Counter:
		return countStep(b) >= lastCount
	case world.Toggle:
		return state(b) == 1
	}
	return false
}

// front is where the directed block b at p gives its power.
func front(p Pos, b world.Block) Pos { return add(p, pushes[facing(b)]) }

// back is where the directed block b at p takes its power from.
func back(p Pos, b world.Block) Pos {
	d := pushes[facing(b)]
	return Pos{X: p.X - d.X, Y: p.Y - d.Y, Z: p.Z - d.Z}
}

// gives reports whether the block b at `at` sends power into `to` beside it.
func gives(at Pos, b world.Block, to Pos) bool {
	if isDirected(b) {
		return isOn(b) && front(at, b) == to
	}
	return isSource(b)
}

// switched is a logic block in its on or off state.
func switched(b world.Block, on bool) world.Block {
	b -= world.Block(world.Info(b).Variant % 2)
	if on {
		return b + 1
	}
	return b
}

// Involves reports whether placing or breaking b can change a circuit.
func Involves(b world.Block) bool {
	switch b.Item() {
	case world.Wire, world.Switch, world.Button, world.Plate, world.SignalLamp,
		world.Door, world.Piston, world.Timer, world.Counter, world.NoteBlock, world.ScriptBlock,
		world.Transmitter, world.Receiver, world.Toggle,
		world.PlayerDetector, world.CreatureDetector, world.Clock, world.BlockDetector,
		world.WeatherSensor, world.StickyPiston, world.Trapdoor, world.FenceGate, world.LaunchPad,
		world.NumberDisplay, world.Timers, world.Tripwire:
		return true
	}
	return isDirected(b) || isWire(b)
}

// Use presses the block at p by hand: switches flip, buttons press, doors swing,
// counters reset and note blocks tune up. It reports whether p was usable.
func Use(w World, p Pos) ([]Change, bool, error) {
	u := updater{w: w}
	b, err := u.block(p)
	if err != nil {
		return nil, false, err
	}
	var next world.Block
	switch b.Item() {
	case world.Switch:
		next = switched(b, !isOn(b))
	case world.Button:
		next = switched(b, true)
	case world.Counter:
		next = switched(world.Counter, isOn(b))
	case world.NoteBlock:
		next = switched(world.NoteBlock+world.Block(2*((state(b)+1)%notes)), isOn(b))
	case world.Door, world.Trapdoor, world.FenceGate:
		return u.changes, true, u.setPanel(p, !isOn(b))
	case world.NumberDisplay:
		next = tappedDisplay(b)
	case world.Timer, world.Timers:
		next = slowerTimer(b)
	case world.Toggle:
		next = toggled(!isSource(b), isOn(b))
	case world.Gate, world.Delay, world.Capacitor:
		next = nextSetting(b)
		if err := u.set(p, b, next); err != nil {
			return u.changes, true, err
		}
		return u.changes, true, u.schedule(p, next)
	case world.Transmitter:
		next = nextChannel(b)
	case world.Clock:
		next = nextClockMode(b)
	case world.Receiver:
		u.w.Schedule(p, 0, false)
		next = nextChannel(b)
	default:
		return nil, false, nil
	}
	if err := u.set(p, b, next); err != nil || isSource(b) == isSource(next) {
		return u.changes, true, err
	}
	return u.update(p)
}

// Set changes the logic block at p and re-powers the circuits around it.
func Set(w World, p Pos, b world.Block) ([]Change, error) {
	u := updater{w: w}
	current, err := u.block(p)
	if err != nil || current == b {
		return nil, err
	}
	if err := u.set(p, current, b); err != nil {
		return nil, err
	}
	_, _, err = u.update(p)
	return u.changes, err
}

// Beat turns each timer among `timers` on or off for the given quarter second since the epoch.
func Beat(w World, timers []Pos, quarter int64) ([]Change, error) {
	var out []Change
	for _, p := range timers {
		b, err := w.Block(p.X, p.Y, p.Z)
		if err != nil {
			continue
		}
		beats, ok := timerBeats(b)
		if !ok {
			continue
		}
		changes, err := Set(w, p, switched(b, (quarter/int64(beats))%2 == 0))
		if err != nil {
			return out, err
		}
		out = append(out, changes...)
	}
	return out, nil
}

// Step makes the switches due this tick, each judged by the power before any of them.
func Step(w World, due []Due) ([]Change, error) {
	u := updater{w: w}
	var flips []Change
	for _, d := range due {
		b, err := u.block(d.Pos)
		if err != nil {
			return nil, err
		}
		flip, err := u.changed(d.Pos, b)
		if err != nil {
			return nil, err
		}
		if flip || (d.Flip && isDirected(b)) {
			flips = append(flips, Change{d.Pos, b})
		}
	}
	for _, f := range flips {
		if err := u.set(f.Pos, f.Block, switched(f.Block, !isOn(f.Block))); err != nil {
			return u.changes, err
		}
	}
	for _, f := range flips {
		if _, _, err := u.update(f.Pos); err != nil {
			return u.changes, err
		}
	}
	return u.changes, nil
}

// Update re-powers every circuit touching p after the block there changed.
func Update(w World, p Pos) ([]Change, error) {
	changes, _, err := (&updater{w: w}).update(p)
	return changes, err
}

// Partner is where the teleporter at p sends players: the next teleporter joined to it by wire.
func Partner(w World, p Pos) (Pos, bool, error) {
	u := updater{w: w, seen: map[Pos]bool{}}
	var pads []Pos
	for _, d := range sides {
		_, near, err := u.wiresFrom(add(p, d))
		if err != nil {
			return p, false, err
		}
		for _, at := range near {
			if b, err := u.block(at); err == nil && b == world.Teleporter && !slices.Contains(pads, at) {
				pads = append(pads, at)
			}
		}
	}
	pads = slices.DeleteFunc(pads, func(at Pos) bool { return at == p })
	if len(pads) == 0 {
		return p, false, nil
	}
	slices.SortFunc(pads, comparePos)
	i, _ := slices.BinarySearchFunc(pads, p, comparePos)
	return pads[i%len(pads)], true, nil
}

func comparePos(a, b Pos) int {
	switch {
	case a.X != b.X:
		return int(a.X) - int(b.X)
	case a.Y != b.Y:
		return int(a.Y) - int(b.Y)
	}
	return int(a.Z) - int(b.Z)
}

type updater struct {
	w       World
	seen    map[Pos]bool
	changes []Change
	again   []Pos // counters that started or stopped giving power
}

func (u *updater) block(p Pos) (world.Block, error) { return u.w.Block(p.X, p.Y, p.Z) }

func (u *updater) set(p Pos, was, now world.Block) error {
	if was == now {
		return nil
	}
	u.changes = append(u.changes, Change{p, now})
	if err := u.w.SetBlock(p.X, p.Y, p.Z, now); err != nil || now.Item() != world.Counter {
		return err
	}
	return u.showOnDisplays(p)
}

// update re-powers around p, then around each counter that changed, a bounded number of times.
func (u *updater) update(p Pos) ([]Change, bool, error) {
	queue := []Pos{p}
	for rounds := 0; len(queue) > 0 && rounds < maxRounds; rounds++ {
		if err := u.round(queue[0]); err != nil {
			return u.changes, true, err
		}
		queue = append(queue[1:], u.again...)
		u.again = nil
	}
	return u.changes, true, nil
}

// round powers the circuits beside p, then the machines beside them.
func (u *updater) round(p Pos) error {
	u.seen = map[Pos]bool{}
	around := []Pos{p}
	for _, d := range sides {
		around = append(around, add(p, d))
	}
	machines := slices.Clone(around)
	for _, at := range around {
		found, err := u.circuit(at)
		if err != nil {
			return err
		}
		machines = append(machines, found...)
	}
	for _, at := range machines {
		if err := u.run(at); err != nil {
			return err
		}
	}
	return nil
}

// circuit powers the wire joined to start if any source touches it, and returns what else is beside it.
func (u *updater) circuit(start Pos) ([]Pos, error) {
	wires, near, err := u.wiresFrom(start)
	if err != nil || len(wires) == 0 {
		return nil, err
	}
	powered, err := u.feeds(wires, near, nil)
	if err != nil {
		return nil, err
	}
	for _, c := range wires {
		if err := u.set(c.Pos, c.Block, switched(c.Block, powered)); err != nil {
			return nil, err
		}
	}
	return near, nil
}

// feeds reports whether a block in near, other than skip, gives power into any of wires.
func (u *updater) feeds(wires []Change, near []Pos, skip *Pos) (bool, error) {
	for _, at := range near {
		if skip != nil && at == *skip {
			continue
		}
		b, err := u.block(at)
		if err != nil {
			return false, err
		}
		if isDirected(b) {
			f := front(at, b)
			if isOn(b) && slices.ContainsFunc(wires, func(c Change) bool { return c.Pos == f }) {
				return true, nil
			}
		} else if isSource(b) {
			return true, nil
		}
	}
	return false, nil
}

// wiresFrom walks the wire joined to start, returning it and the blocks next to it.
func (u *updater) wiresFrom(start Pos) ([]Change, []Pos, error) {
	var wires []Change
	var near []Pos
	queue := []Pos{start}
	for len(queue) > 0 && len(wires) < MaxWires {
		at := queue[0]
		queue = queue[1:]
		if u.seen[at] {
			continue
		}
		b, err := u.block(at)
		if err != nil {
			return nil, nil, err
		}
		if !isWire(b) {
			if len(wires) > 0 {
				near = append(near, at)
			}
			continue
		}
		u.seen[at] = true
		wires = append(wires, Change{at, b})
		for _, d := range sides {
			queue = append(queue, add(at, d))
		}
		steps, err := u.steps(at, b)
		if err != nil {
			return nil, nil, err
		}
		queue = append(queue, steps...)
	}
	return wires, near, nil
}

// steps are the wires joined to the wire b at p one block up or down a step.
func (u *updater) steps(p Pos, b world.Block) ([]Pos, error) {
	if b.Item() != world.Wire {
		return nil, nil
	}
	var out []Pos
	for _, d := range pushes[:4] {
		for _, dy := range []int32{1, -1} {
			to := add(add(p, d), Pos{Y: dy})
			joined, err := u.stepJoins(p, to)
			if err != nil {
				return nil, err
			}
			if joined {
				out = append(out, to)
			}
		}
	}
	return out, nil
}

// stepJoins reports whether wire at `to`, a step above or below the wire at p, joins it:
// the block over the lower of the two must leave room for the wire to bend.
func (u *updater) stepJoins(p, to Pos) (bool, error) {
	b, err := u.block(to)
	if err != nil || b.Item() != world.Wire {
		return false, err
	}
	lower := p
	if to.Y < p.Y {
		lower = to
	}
	over, err := u.block(add(lower, Pos{Y: 1}))
	return err == nil && !world.Info(over).Solid, err
}

// powered reports whether a source or powered wire is beside p.
func (u *updater) powered(p Pos) (bool, error) {
	for _, d := range sides {
		if on, err := u.givesInto(add(p, d), p); err != nil || on {
			return on, err
		}
	}
	return false, nil
}

// givesInto reports whether powered wire or a source at `from` sends power into `to` beside it.
func (u *updater) givesInto(from, to Pos) (bool, error) {
	b, err := u.block(from)
	return err == nil && ((isWire(b) && isOn(b)) || gives(from, b, to)), err
}

// input reports whether the machine b at p has power coming in.
func (u *updater) input(p Pos, b world.Block) (bool, error) {
	switch {
	case b.Item() == world.Counter || b.Item() == world.Toggle:
		return u.fedFromElsewhere(p)
	case b.Item() == world.Gate:
		return u.gateInput(p, b)
	case b.Item() == world.Relay:
		return u.relayInput(p, b)
	case b.Item() == world.Receiver:
		return u.channelOn(state(b))
	case isDirected(b):
		return u.givesInto(back(p, b), p)
	}
	return u.powered(p)
}

// fedFromElsewhere reports whether power reaches p from a source other than p itself,
// so a counter giving power still sees its own input go off and on.
func (u *updater) fedFromElsewhere(p Pos) (bool, error) {
	seen := u.seen
	defer func() { u.seen = seen }()
	u.seen = map[Pos]bool{}
	for _, d := range sides {
		n, err := u.block(add(p, d))
		if err != nil || gives(add(p, d), n, p) {
			return err == nil, err
		}
		wires, near, err := u.wiresFrom(add(p, d))
		if err != nil {
			return false, err
		}
		if fed, err := u.feeds(wires, near, &p); err != nil || fed {
			return fed, err
		}
	}
	return false, nil
}

// run lets the machine at p follow the power beside it.
func (u *updater) run(p Pos) error {
	b, err := u.block(p)
	if err != nil {
		return err
	}
	if isDirected(b) {
		return u.schedule(p, b)
	}
	switch b.Item() {
	case world.SignalLamp, world.NoteBlock, world.Piston, world.StickyPiston, world.Counter, world.ScriptBlock,
		world.Transmitter, world.Toggle, world.LaunchPad:
	case world.Door, world.Trapdoor, world.FenceGate:
		on, err := u.panelPowered(p)
		if err != nil {
			return err
		}
		return u.setPanel(p, on)
	case world.NumberDisplay:
		return u.showNumber(p, b)
	default:
		return nil
	}
	on, err := u.input(p, b)
	if err != nil || on == isOn(b) {
		return err
	}
	switch b.Item() {
	case world.Piston, world.StickyPiston:
		if err := u.pushOrPull(p, b, on); err != nil {
			return err
		}
	case world.Counter:
		return u.count(p, b, on)
	case world.Toggle:
		return u.flipToggle(p, b, on)
	}
	return u.set(p, b, switched(b, on))
}

// schedule asks for the directed block b at p to switch on the next tick if its input changed;
// a delay switches after its wait whatever its input does meanwhile, and a capacitor whose
// input stopped is rechecked after it has held its power long enough.
func (u *updater) schedule(p Pos, b world.Block) error {
	flip, err := u.changed(p, b)
	switch {
	case !flip:
	case b.Item() == world.Delay:
		u.w.Schedule(p, delayOf(b), true)
	case b.Item() == world.Capacitor && isOn(b):
		u.w.Schedule(p, holdOf(b), false)
	default:
		u.w.Schedule(p, 0, false)
	}
	return err
}

// changed reports whether b at p is a directed block or receiver whose input no longer matches
// its output.
func (u *updater) changed(p Pos, b world.Block) (bool, error) {
	if !isDirected(b) && b.Item() != world.Receiver {
		return false, nil
	}
	on, err := u.input(p, b)
	return err == nil && on != isOn(b), err
}

// countStep is how far through its cycle a counter is: up from 0 to 9, then down past it.
func countStep(b world.Block) int {
	if b >= world.CountingDown {
		return countCycle - 1 - int(b-world.CountingDown)/2
	}
	return state(b)
}

// counterAt is the counter showing the given step of its cycle, not yet powered.
func counterAt(step int) world.Block {
	if step <= lastCount {
		return world.Counter + world.Block(2*step)
	}
	return world.CountingDown + world.Block(2*(countCycle-1-step))
}

// count notes the power coming into a counter, counting each time it arrives.
func (u *updater) count(p Pos, b world.Block, on bool) error {
	step := countStep(b)
	if on {
		step = (step + 1) % countCycle
	}
	next := switched(counterAt(step), on)
	if isSource(next) != isSource(b) {
		u.again = append(u.again, p)
	}
	return u.set(p, b, next)
}

func (u *updater) panelPowered(p Pos) (bool, error) {
	panel, err := u.panel(p)
	if err != nil {
		return false, err
	}
	for _, at := range panel {
		if on, err := u.powered(at); err != nil || on {
			return on, err
		}
	}
	return false, nil
}

// setPanel opens or closes the whole door, trapdoor floor or gate that p is part of; it won't
// close on anyone.
func (u *updater) setPanel(p Pos, open bool) error {
	panel, err := u.panel(p)
	if err != nil || (!open && slices.ContainsFunc(panel, u.w.Occupied)) {
		return err
	}
	for _, at := range panel {
		b, err := u.block(at)
		if err != nil {
			return err
		}
		if err := u.set(at, b, switched(b, open)); err != nil {
			return err
		}
	}
	return nil
}

// pushOrPull moves blocks for the piston b at p as its power starts, or as a sticky one's stops.
func (u *updater) pushOrPull(p Pos, b world.Block, on bool) error {
	d := pushes[facing(b)]
	switch {
	case on:
		return u.push(p, d)
	case b.Item() == world.StickyPiston:
		return u.pull(p, d)
	}
	return nil
}

// push moves the row of blocks in front of the piston at p one step along d, if it can.
func (u *updater) push(p Pos, d Pos) error {
	row := []Change{}
	for at := add(p, d); len(row) <= MaxPush; at = add(at, d) {
		b, err := u.block(at)
		if errors.Is(err, world.ErrOutside) || !u.w.CanPush(p, at) {
			return nil
		}
		if err != nil {
			return err
		}
		if b == world.Air || b == world.Water {
			return u.shift(row, d)
		}
		if !movable(b) {
			return nil
		}
		row = append(row, Change{at, b})
	}
	return nil
}

// shift moves each block in row one step along d, leaving air where the row began.
func (u *updater) shift(row []Change, d Pos) error {
	for i := len(row) - 1; i >= 0; i-- {
		to := add(row[i].Pos, d)
		was, err := u.block(to)
		if err != nil {
			return err
		}
		if err := u.set(to, was, row[i].Block); err != nil {
			return err
		}
	}
	if len(row) == 0 {
		return nil
	}
	return u.set(row[0].Pos, row[0].Block, world.Air)
}

func movable(b world.Block) bool {
	return b != world.Bedrock && !Involves(b) && !world.Info(b).Usable
}
