package access

import (
	"sync"
	"time"
)

const (
	maxCodeFailures = 5
	failureWindow   = 10 * time.Minute
)

// limiter counts bad join codes per IP in fixed windows. Stale entries are pruned lazily, so it needs no timer.
type limiter struct {
	mu   sync.Mutex
	hits map[string]window
}

type window struct {
	start    time.Time
	failures int
}

func newLimiter() *limiter { return &limiter{hits: map[string]window{}} }

func (l *limiter) blocked(ip string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	w, ok := l.hits[ip]
	if ok && now.Sub(w.start) >= failureWindow {
		delete(l.hits, ip)
		return false
	}
	return ok && w.failures >= maxCodeFailures
}

func (l *limiter) fail(ip string, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.prune(now)
	w, ok := l.hits[ip]
	if !ok || now.Sub(w.start) >= failureWindow {
		w = window{start: now}
	}
	w.failures++
	l.hits[ip] = w
}

func (l *limiter) prune(now time.Time) {
	for ip, w := range l.hits {
		if now.Sub(w.start) >= failureWindow {
			delete(l.hits, ip)
		}
	}
}
