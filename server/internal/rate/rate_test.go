package rate

import (
	"testing"
	"time"
)

func TestBucketRefillsOverTime(t *testing.T) {
	t0 := time.Unix(100, 0)
	b := New(t0, 10, 20)
	n := 0
	for b.Take(t0) {
		n++
	}
	if n != 20 {
		t.Fatalf("took %d at once, want 20", n)
	}
	if !b.Take(t0.Add(time.Second / 10)) {
		t.Fatal("bucket should refill")
	}
	if b.Take(t0.Add(time.Second / 10)) {
		t.Fatal("bucket refilled too much")
	}
}

func TestBucketNeverHoldsMoreThanItsBurst(t *testing.T) {
	t0 := time.Unix(100, 0)
	b := New(t0, 1, 2)
	later := t0.Add(time.Hour)
	n := 0
	for b.Take(later) {
		n++
	}
	if n != 2 {
		t.Fatalf("took %d after a long wait, want the burst of 2", n)
	}
}
