package gateway

import (
	"context"
	"testing"
	"time"
)

var t0 = time.UnixMilli(1_000_000)

func TestChatLogListsLinesAfterAnId(t *testing.T) {
	var l chatLog
	l.add("Mom", "one", false, t0)
	l.add("Mom", "🤖 two", true, t0)
	l.add("Kid", "three", false, t0.Add(time.Second))
	got := l.since(1, t0.Add(3*time.Second))
	if len(got.Lines) != 2 || got.Lines[0].Text != "🤖 two" || !got.Lines[0].Agent || got.Last != 3 {
		t.Fatalf("got %+v", got)
	}
	if got.Lines[1].SecondsAgo != 2 {
		t.Errorf("seconds ago %d", got.Lines[1].SecondsAgo)
	}
}

func TestChatLogKeepsTheLastLines(t *testing.T) {
	var l chatLog
	for range chatKept + 5 {
		l.add("Mom", "hi", false, t0)
	}
	got := l.since(0, t0)
	if len(got.Lines) != chatKept || got.Lines[0].ID != 6 || got.Last != chatKept+5 {
		t.Fatalf("got %d lines from %d, last %d", len(got.Lines), got.Lines[0].ID, got.Last)
	}
}

func TestChatLogWaitsForAPlayersLine(t *testing.T) {
	var l chatLog
	l.add("Mom", "before", false, t0)
	woke := make(chan bool, 1)
	go func() { woke <- l.wait(context.Background(), 1, time.Second) }()
	time.Sleep(20 * time.Millisecond)
	l.add("Mom", "🤖 my own line", true, t0)
	select {
	case <-woke:
		t.Fatal("woke for the agent's own line")
	case <-time.After(50 * time.Millisecond):
	}
	l.add("Mom", "make it taller", false, t0)
	select {
	case ok := <-woke:
		if !ok {
			t.Fatal("wait said nothing came")
		}
	case <-time.After(time.Second):
		t.Fatal("never woke")
	}
}

func TestChatLogWaitReturnsAtOnceOrTimesOut(t *testing.T) {
	var l chatLog
	l.add("Mom", "already here", false, t0)
	if !l.wait(context.Background(), 0, time.Second) {
		t.Error("a line after 0 was already there")
	}
	start := time.Now()
	if l.wait(context.Background(), 1, 30*time.Millisecond) || time.Since(start) < 30*time.Millisecond {
		t.Error("should wait the full time, then say nothing came")
	}
}
