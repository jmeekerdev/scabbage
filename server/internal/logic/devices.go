package logic

import (
	"errors"

	"scabbage/internal/world"
)

// MaxPanel bounds how many doors, trapdoors or fence gates open together.
const MaxPanel = 256

// displays is how many numbers a number display shows.
const displays = 10

// panelJoins is the ways each kind of door joins the ones beside it to open as one.
var panelJoins = map[world.Block][]Pos{
	world.Door:      {{Y: 1}, {Y: -1}},
	world.Trapdoor:  pushes[:4],
	world.FenceGate: sides[:],
}

// panel is the door, trapdoor floor or gate that p is part of, up to MaxPanel blocks.
func (u *updater) panel(p Pos) ([]Pos, error) {
	b, err := u.block(p)
	if err != nil {
		return nil, err
	}
	item := b.Item()
	out := []Pos{p}
	seen := map[Pos]bool{p: true}
	for i := 0; i < len(out) && len(out) < MaxPanel; i++ {
		for _, d := range panelJoins[item] {
			at := add(out[i], d)
			if seen[at] || len(out) >= MaxPanel {
				continue
			}
			seen[at] = true
			n, err := u.block(at)
			if errors.Is(err, world.ErrOutside) {
				continue
			}
			if err != nil {
				return nil, err
			}
			if n.Item() == item {
				out = append(out, at)
			}
		}
	}
	return out, nil
}

// pull moves the block two steps along d from the sticky piston at p back beside it, if it can.
func (u *updater) pull(p, d Pos) error {
	near, far := add(p, d), add(add(p, d), d)
	gap, err := u.block(near)
	if err != nil || (gap != world.Air && gap != world.Water) {
		return ignoreOutside(err)
	}
	b, err := u.block(far)
	if err != nil || b == world.Air || b == world.Water || !movable(b) {
		return ignoreOutside(err)
	}
	if !u.w.CanPush(p, near) || !u.w.CanPush(p, far) {
		return nil
	}
	if err := u.set(near, gap, b); err != nil {
		return err
	}
	return u.set(far, b, world.Air)
}

func ignoreOutside(err error) error {
	if errors.Is(err, world.ErrOutside) {
		return nil
	}
	return err
}

// showNumber has the number display at p show the counter beside it, or count its pulses.
func (u *updater) showNumber(p Pos, b world.Block) error {
	on, err := u.powered(p)
	if err != nil {
		return err
	}
	n, found, err := u.counterBeside(p)
	if err != nil {
		return err
	}
	if !found {
		n = state(b)
		if on && !isOn(b) {
			n = (n + 1) % displays
		}
	}
	return u.set(p, b, switched(displayShowing(n), on))
}

// counterBeside is the number a counter beside p shows, if there is one.
func (u *updater) counterBeside(p Pos) (int, bool, error) {
	for _, d := range sides {
		b, err := u.block(add(p, d))
		if err != nil {
			return 0, false, ignoreOutside(err)
		}
		if b.Item() == world.Counter {
			return shownCount(b), true, nil
		}
	}
	return 0, false, nil
}

// showOnDisplays updates the number displays beside the counter at p.
func (u *updater) showOnDisplays(p Pos) error {
	for _, d := range sides {
		at := add(p, d)
		b, err := u.block(at)
		if err != nil {
			return ignoreOutside(err)
		}
		if b.Item() == world.NumberDisplay {
			if err := u.showNumber(at, b); err != nil {
				return err
			}
		}
	}
	return nil
}

// shownCount is the digit a counter shows: up to 9, then back down.
func shownCount(b world.Block) int {
	if step := countStep(b); step <= lastCount {
		return step
	}
	return countCycle - countStep(b)
}

// displayShowing is the number display showing n, not fed.
func displayShowing(n int) world.Block { return world.NumberDisplay + world.Block(2*n) }

// tappedDisplay is the display b counted up by one, keeping whether it is fed.
func tappedDisplay(b world.Block) world.Block {
	return switched(displayShowing((state(b)+1)%displays), isOn(b))
}
