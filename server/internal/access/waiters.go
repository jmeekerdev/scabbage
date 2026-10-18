package access

import "sync"

// waiters lets pending sessions block until an admin decides their request.
type waiters struct {
	mu   sync.Mutex
	next int
	byID map[int64]map[int]chan bool
}

func newWaiters() *waiters { return &waiters{byID: map[int64]map[int]chan bool{}} }

func (w *waiters) add(id int64) (<-chan bool, func()) {
	w.mu.Lock()
	defer w.mu.Unlock()
	ch := make(chan bool, 1)
	key := w.next
	w.next++
	if w.byID[id] == nil {
		w.byID[id] = map[int]chan bool{}
	}
	w.byID[id][key] = ch
	return ch, func() { w.remove(id, key) }
}

func (w *waiters) remove(id int64, key int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.byID[id], key)
	if len(w.byID[id]) == 0 {
		delete(w.byID, id)
	}
}

func (w *waiters) deliver(id int64, approved bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, ch := range w.byID[id] {
		ch <- approved // buffered 1 and delivered at most once
	}
	delete(w.byID, id)
}
