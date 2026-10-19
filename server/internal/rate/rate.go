// Package rate limits how often one player may do something.
package rate

import (
	"math"
	"time"
)

// Bucket is a token bucket: up to burst actions at once, refilling perSecond.
type Bucket struct {
	perSecond, burst float64
	tokens           float64
	last             time.Time
}

// New starts with a full bucket.
func New(now time.Time, perSecond, burst float64) *Bucket {
	return &Bucket{perSecond: perSecond, burst: burst, tokens: burst, last: now}
}

// Take spends one token if any are left.
func (b *Bucket) Take(now time.Time) bool {
	dt := max(now.Sub(b.last).Seconds(), 0)
	b.tokens, b.last = math.Min(b.tokens+dt*b.perSecond, b.burst), now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}
