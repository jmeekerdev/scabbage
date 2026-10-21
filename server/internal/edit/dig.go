package edit

import "time"

// jitter is the share of a dig time packets may arrive early by.
const jitter = 0.3

// Digger makes a survival player spend each dig's mining time between breaks.
type Digger struct{ last time.Time }

// Dig reports whether a dig taking `need` may break now, and if so starts timing the next one.
func (d *Digger) Dig(need time.Duration, now time.Time) bool {
	if now.Sub(d.last) < time.Duration(float64(need)*(1-jitter)) {
		return false
	}
	d.last = now
	return true
}
