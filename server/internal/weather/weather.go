// Package weather picks the weather from the wall clock, so the server and every client agree
// without sending anything; client/src/render/weather.ts mirrors it.
package weather

import "time"

// Kind is what the sky is doing.
type Kind int

const (
	Clear Kind = iota
	Rain       // snow, high up
	Storm      // heavy rain or snow with lightning
)

// Spell is how long each weather lasts.
const Spell = 4 * time.Minute

// At is the weather at now: about half clear, a third rain and the rest storms.
func At(now time.Time) Kind {
	roll := mix(uint32(now.UnixMilli()/Spell.Milliseconds())) % 100
	switch {
	case roll < 50:
		return Clear
	case roll < 85:
		return Rain
	}
	return Storm
}

// Wet reports whether anything falls from the sky.
func (k Kind) Wet() bool { return k != Clear }

func (k Kind) String() string { return [...]string{"clear", "rain", "storm"}[k] }

// mix scrambles a spell's number into an even spread of rolls.
func mix(h uint32) uint32 {
	h *= 0x9e3779b1
	h ^= h >> 16
	h *= 0x85ebca6b
	h ^= h >> 13
	h *= 0xc2b2ae35
	return h ^ h>>16
}
