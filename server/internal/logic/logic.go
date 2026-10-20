// Package logic powers circuits: switches, buttons and pressure plates send power down wire to lamps.
package logic

import (
	"slices"

	"scabbage/internal/world"
)

// MaxWires bounds how much wire one update walks, so a giant circuit can't stall the server.
const MaxWires = 4096

// Pos is a block position.
type Pos struct{ X, Y, Z int32 }

// World is the block storage circuits read and change.
type World interface {
	Block(x, y, z int32) (world.Block, error)
	SetBlock(x, y, z int32, b world.Block) error
}

// Change is a block a circuit update switched on or off.
type Change struct {
	Pos
	Block world.Block
}

var sides = [6]Pos{{1, 0, 0}, {-1, 0, 0}, {0, 1, 0}, {0, -1, 0}, {0, 0, 1}, {0, 0, -1}}

func (p Pos) add(d Pos) Pos { return Pos{p.X + d.X, p.Y + d.Y, p.Z + d.Z} }

func isWire(b world.Block) bool   { return b.Item() == world.Wire }
func isLamp(b world.Block) bool   { return b.Item() == world.SignalLamp }
func isSource(b world.Block) bool { return world.Info(b).Powered && !isWire(b) && !isLamp(b) }

// switched is a logic block in its on or off state.
func switched(b world.Block, on bool) world.Block {
	if on {
		return b.Item() + 1
	}
	return b.Item()
}

// Involves reports whether placing or breaking b can change a circuit.
func Involves(b world.Block) bool {
	switch b.Item() {
	case world.Wire, world.Switch, world.Button, world.Plate, world.SignalLamp:
		return true
	}
	return false
}

// Use is what a switch or button becomes when a player presses it.
func Use(b world.Block) (world.Block, bool) {
	switch b {
	case world.Switch, world.Switch + 1:
		return switched(b, b == world.Switch), true
	case world.Button, world.Button + 1:
		return world.Button + 1, true
	}
	return b, false
}

// Update re-powers every circuit touching p after the block there changed.
func Update(w World, p Pos) ([]Change, error) {
	u := updater{w: w, seen: map[Pos]bool{}}
	around := []Pos{p}
	for _, d := range sides {
		around = append(around, p.add(d))
	}
	lamps := slices.Clone(around)
	for _, at := range around {
		found, err := u.circuit(at)
		if err != nil {
			return nil, err
		}
		lamps = append(lamps, found...)
	}
	for _, at := range lamps {
		if err := u.lamp(at); err != nil {
			return nil, err
		}
	}
	return u.changes, nil
}

type updater struct {
	w       World
	seen    map[Pos]bool
	changes []Change
}

func (u *updater) block(p Pos) (world.Block, error) { return u.w.Block(p.X, p.Y, p.Z) }

func (u *updater) set(p Pos, was, now world.Block) error {
	if was == now {
		return nil
	}
	u.changes = append(u.changes, Change{p, now})
	return u.w.SetBlock(p.X, p.Y, p.Z, now)
}

// circuit powers the wire joined to start if any source touches it, and returns the lamps beside it.
func (u *updater) circuit(start Pos) ([]Pos, error) {
	wires, near, err := u.wiresFrom(start)
	if err != nil || len(wires) == 0 {
		return nil, err
	}
	powered := false
	var lamps []Pos
	for _, at := range near {
		b, err := u.block(at)
		if err != nil {
			return nil, err
		}
		powered = powered || isSource(b)
		if isLamp(b) {
			lamps = append(lamps, at)
		}
	}
	for _, c := range wires {
		if err := u.set(c.Pos, c.Block, switched(c.Block, powered)); err != nil {
			return nil, err
		}
	}
	return lamps, nil
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
			queue = append(queue, at.add(d))
		}
	}
	return wires, near, nil
}

// lamp lights a lamp when a source or powered wire is beside it.
func (u *updater) lamp(p Pos) error {
	b, err := u.block(p)
	if err != nil || !isLamp(b) {
		return err
	}
	on := false
	for _, d := range sides {
		n, err := u.block(p.add(d))
		if err != nil {
			return err
		}
		on = on || isSource(n) || n == world.Wire+1
	}
	return u.set(p, b, switched(b, on))
}
