package logic

import (
	"time"

	"scabbage/internal/world"
)

// delayStep is how much longer each of a delay's settings waits than the one before.
const delayStep = time.Second / 4

// setting is which of its item's ways the directed block b is set, e.g. a gate's mode.
func setting(b world.Block) int { return state(b) / len(pushes) }

// nextSetting is the gate, delay or capacitor b set to its next way, facing the same way.
func nextSetting(b world.Block) world.Block {
	count := world.Delays
	switch b.Item() {
	case world.Gate:
		count = len(world.GateModes)
	case world.Capacitor:
		count = world.Capacitors
	}
	next := (setting(b) + 1) % count
	return b + world.Block(2*len(pushes)*(next-setting(b)))
}

// delayOf is how long the delay b waits before passing a change on.
func delayOf(b world.Block) time.Duration { return time.Duration(setting(b)+1) * delayStep }

// relayInput reports whether the relay b at p should pass power: its back and a side both on.
func (u *updater) relayInput(p Pos, b world.Block) (bool, error) {
	if on, err := u.givesInto(back(p, b), p); err != nil || !on {
		return false, err
	}
	for _, s := range flatSides(p, b) {
		if on, err := u.givesInto(s, p); err != nil || on {
			return on, err
		}
	}
	return false, nil
}

// nextChannel is the wireless block b on its next channel, still on or off.
func nextChannel(b world.Block) world.Block {
	return switched(b.Item()+world.Block(2*((state(b)+1)%world.Channels)), isOn(b))
}

// channelOn reports whether any loaded transmitter on channel is on.
func (u *updater) channelOn(channel int) (bool, error) {
	for _, p := range u.w.Transmitters() {
		b, err := u.block(p)
		if err != nil {
			return false, err
		}
		if b.Item() == world.Transmitter && state(b) == channel && isOn(b) {
			return true, nil
		}
	}
	return false, nil
}
