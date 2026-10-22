package logic

import "scabbage/internal/world"

const (
	and = iota
	or
	xor
	nand
	nor
	not
)

func gateMode(b world.Block) int { return setting(b) }

// gateInput reports whether the gate b at p should give power, judging the inputs joined to it.
func (u *updater) gateInput(p Pos, b world.Block) (bool, error) {
	joined, lit := 0, 0
	for _, at := range u.gateInputs(p, b) {
		n, err := u.block(at)
		if err != nil {
			return false, err
		}
		if !canGiveInto(at, n, p) {
			continue
		}
		joined++
		if on, err := u.givesInto(at, p); err != nil {
			return false, err
		} else if on {
			lit++
		}
	}
	return judge(gateMode(b), joined, lit), nil
}

// gateInputs are where the gate b at p reads power: its back, and for all but not gates,
// the sides at right angles to its facing that aren't above or below it.
func (u *updater) gateInputs(p Pos, b world.Block) []Pos {
	out := []Pos{back(p, b)}
	if gateMode(b) == not {
		return out
	}
	return append(out, flatSides(p, b)...)
}

// flatSides are the blocks beside the directed block b at p, at right angles to its facing and
// not above or below it.
func flatSides(p Pos, b world.Block) []Pos {
	d := pushes[facing(b)]
	sides := []Pos{{X: -d.Z, Z: d.X}, {X: d.Z, Z: -d.X}}
	if d.Y != 0 {
		sides = pushes[:4]
	}
	out := make([]Pos, len(sides))
	for i, s := range sides {
		out[i] = add(p, s)
	}
	return out
}

// canGiveInto reports whether the block b at `at` is something that could power `to`, on or off.
func canGiveInto(at Pos, b world.Block, to Pos) bool {
	switch b.Item() {
	case world.Switch, world.Button, world.Plate, world.Timer, world.Timers, world.Tripwire, world.Counter, world.Receiver:
		return true
	}
	return isWire(b) || (isDirected(b) && front(at, b) == to)
}

// judge is whether a gate in mode gives power, with `lit` of its `joined` inputs on.
func judge(mode, joined, lit int) bool {
	switch mode {
	case and:
		return joined > 0 && lit == joined
	case or:
		return lit > 0
	case xor:
		return lit == 1
	case nand:
		return joined == 0 || lit < joined
	}
	return lit == 0 // nor and not
}
