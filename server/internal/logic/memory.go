package logic

import (
	"time"

	"scabbage/internal/world"
)

// holdStep is how much longer each of a capacitor's settings holds power than the one before.
const holdStep = time.Second

// holdOf is how long the capacitor b keeps giving power after its input stops.
func holdOf(b world.Block) time.Duration { return time.Duration(setting(b)+1) * holdStep }

// toggled is a toggle giving power or not, still noting whether power is coming in.
func toggled(giving, fed bool) world.Block {
	b := world.Toggle
	if giving {
		b += 2
	}
	return switched(b, fed)
}

// flipToggle notes the power coming into the toggle b at p, flipping it each time power arrives.
func (u *updater) flipToggle(p Pos, b world.Block, on bool) error {
	next := toggled(isSource(b) != on, on)
	if isSource(next) != isSource(b) {
		u.again = append(u.again, p)
	}
	return u.set(p, b, next)
}
