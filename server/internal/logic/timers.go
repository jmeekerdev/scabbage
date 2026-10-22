package logic

import (
	"slices"

	"scabbage/internal/world"
)

// timerSpeeds are every timer item's quarter seconds on, then off, from fastest.
var timerSpeeds = func() []int {
	out := append(slices.Clone(world.TimerBeats), timerBeat)
	slices.Sort(out)
	return out
}()

// timerBeat is how many quarter seconds the first timer stays on, then off.
const timerBeat = 4

// timerBeats is how many quarter seconds the timer b stays on, then off; ok is false for non-timers.
func timerBeats(b world.Block) (int, bool) {
	switch b.Item() {
	case world.Timer:
		return timerBeat, true
	case world.Timers:
		return world.TimerBeats[state(b)], true
	}
	return 0, false
}

// slowerTimer is the timer b at the next slower speed, or the fastest after the slowest, still on or off.
func slowerTimer(b world.Block) world.Block {
	beats, _ := timerBeats(b)
	next := timerSpeeds[(slices.Index(timerSpeeds, beats)+1)%len(timerSpeeds)]
	return switched(timerWith(next), isOn(b))
}

// timerWith is the timer that stays on, then off, for `beats` quarter seconds.
func timerWith(beats int) world.Block {
	if i := slices.Index(world.TimerBeats, beats); i >= 0 {
		return world.Timers + world.Block(2*i)
	}
	return world.Timer
}
