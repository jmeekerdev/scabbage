package edit

import (
	"time"

	"scabbage/internal/world"
)

// jitter is the share of a dig time packets may arrive early by.
const jitter = 0.3

// Digger makes a survival player spend each block's mining time between breaks.
type Digger struct{ last time.Time }

// Dig reports whether b may break now, and if so starts timing the next dig.
func (d *Digger) Dig(b world.Block, now time.Time) bool {
	need := time.Duration(float64(world.Info(b).MineMs)*(1-jitter)) * time.Millisecond
	if now.Sub(d.last) < need {
		return false
	}
	d.last = now
	return true
}
