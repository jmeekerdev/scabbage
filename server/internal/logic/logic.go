// Package logic powers circuits: switches, buttons, plates, timers and counters send power down
// wire to lamps, doors, pistons and note blocks.
package logic

import (
	"errors"
	"slices"

	"scabbage/internal/world"
)

const (
	// MaxWires bounds how much wire one update walks, so a giant circuit can't stall the server.
	MaxWires = 4096
	// MaxPush is the longest row of blocks a piston moves.
	MaxPush = 12
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

func isWire(b world.Block) bool { return b.Item() == world.Wire }
func isOn(b world.Block) bool   { return world.Info(b).Powered }
func state(b world.Block) int   { return int(world.Info(b).Variant / 2) }

// isSource reports whether b sends power into the wire and machines beside it.
func isSource(b world.Block) bool {
	switch b.Item() {
	case world.Switch, world.Button, world.Plate, world.Timer:
		return isOn(b)
	case world.Counter:
		return countStep(b) >= lastCount
	}
	return false
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
		world.Door, world.Piston, world.Timer, world.Counter, world.NoteBlock, world.ScriptBlock:
		return true
	}
	return false
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
	case world.Door:
		return u.changes, true, u.setDoor(p, !isOn(b))
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

// Beat turns the timers at each of `timers` on or off.
func Beat(w World, timers []Pos, on bool) ([]Change, error) {
	var out []Change
	for _, p := range timers {
		b, err := w.Block(p.X, p.Y, p.Z)
		if err != nil || b.Item() != world.Timer {
			continue
		}
		changes, err := Set(w, p, switched(b, on))
		if err != nil {
			return out, err
		}
		out = append(out, changes...)
	}
	return out, nil
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
	return u.w.SetBlock(p.X, p.Y, p.Z, now)
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
	powered := false
	for _, at := range near {
		b, err := u.block(at)
		if err != nil {
			return nil, err
		}
		powered = powered || isSource(b)
	}
	for _, c := range wires {
		if err := u.set(c.Pos, c.Block, switched(c.Block, powered)); err != nil {
			return nil, err
		}
	}
	return near, nil
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
	}
	return wires, near, nil
}

// powered reports whether a source or powered wire is beside p.
func (u *updater) powered(p Pos) (bool, error) {
	for _, d := range sides {
		n, err := u.block(add(p, d))
		if err != nil {
			return false, err
		}
		if isSource(n) || n == world.Wire+1 {
			return true, nil
		}
	}
	return false, nil
}

// input reports whether the machine b at p has power coming in.
func (u *updater) input(p Pos, b world.Block) (bool, error) {
	if b.Item() == world.Counter {
		return u.fedFromElsewhere(p)
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
		if err != nil || isSource(n) {
			return err == nil, err
		}
		_, near, err := u.wiresFrom(add(p, d))
		if err != nil {
			return false, err
		}
		for _, at := range near {
			if b, err := u.block(at); err != nil || (at != p && isSource(b)) {
				return err == nil, err
			}
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
	switch b.Item() {
	case world.SignalLamp, world.NoteBlock, world.Piston, world.Counter, world.ScriptBlock:
	case world.Door:
		on, err := u.doorPowered(p)
		if err != nil {
			return err
		}
		return u.setDoor(p, on)
	default:
		return nil
	}
	on, err := u.input(p, b)
	if err != nil || on == isOn(b) {
		return err
	}
	switch b.Item() {
	case world.Piston:
		if on {
			if err := u.push(p, pushes[state(b)]); err != nil {
				return err
			}
		}
	case world.Counter:
		return u.count(p, b, on)
	}
	return u.set(p, b, switched(b, on))
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

// door is the column of doors that p is part of, however tall; the world's height bounds it.
func (u *updater) door(p Pos) ([]Pos, error) {
	column := []Pos{p}
	for _, dy := range []int32{1, -1} {
		for at := (Pos{X: p.X, Y: p.Y + dy, Z: p.Z}); ; at.Y += dy {
			b, err := u.block(at)
			if errors.Is(err, world.ErrOutside) {
				break
			}
			if err != nil {
				return nil, err
			}
			if b.Item() != world.Door {
				break
			}
			column = append(column, at)
		}
	}
	return column, nil
}

func (u *updater) doorPowered(p Pos) (bool, error) {
	column, err := u.door(p)
	if err != nil {
		return false, err
	}
	for _, at := range column {
		if on, err := u.powered(at); err != nil || on {
			return on, err
		}
	}
	return false, nil
}

// setDoor opens or closes the whole door that p is part of; it won't close on anyone.
func (u *updater) setDoor(p Pos, open bool) error {
	column, err := u.door(p)
	if err != nil || (!open && slices.ContainsFunc(column, u.w.Occupied)) {
		return err
	}
	for _, at := range column {
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
