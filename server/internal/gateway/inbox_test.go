package gateway

import (
	"context"
	"testing"
	"time"
)

func TestInboxHandsEachMessageOverOnce(t *testing.T) {
	var b aiInbox
	b.post("build a treehouse", t0)
	b.post("near the oaks", t0.Add(time.Second))
	got := b.take(context.Background(), 0, t0.Add(3*time.Second))
	if len(got) != 2 || got[0].Text != "build a treehouse" || got[0].SecondsAgo != 3 || got[1].SecondsAgo != 2 {
		t.Fatalf("got %+v", got)
	}
	if again := b.take(context.Background(), 0, t0); len(again) != 0 {
		t.Fatalf("handed over twice: %+v", again)
	}
}

func TestInboxWaitsForAMessage(t *testing.T) {
	var b aiInbox
	got := make(chan []aiMessage, 1)
	go func() { got <- b.take(context.Background(), time.Second, t0) }()
	time.Sleep(20 * time.Millisecond)
	if !b.post("hello", t0) {
		t.Error("a waiting listener should count as heard")
	}
	select {
	case msgs := <-got:
		if len(msgs) != 1 || msgs[0].Text != "hello" {
			t.Fatalf("got %+v", msgs)
		}
	case <-time.After(time.Second):
		t.Fatal("never woke")
	}
	start := time.Now()
	if msgs := b.take(context.Background(), 30*time.Millisecond, t0); len(msgs) != 0 || time.Since(start) < 30*time.Millisecond {
		t.Fatal("should wait the full time, then hand over nothing")
	}
}

func TestInboxSaysWhetherAnAgentListensLately(t *testing.T) {
	var b aiInbox
	if b.post("anyone?", t0) {
		t.Error("no agent has listened")
	}
	b.take(context.Background(), 0, t0)
	if !b.post("now?", t0.Add(listenGrace/2)) {
		t.Error("an agent listened just now")
	}
	if b.post("later?", t0.Add(2*listenGrace)) {
		t.Error("the agent stopped listening long ago")
	}
}

func TestInboxKeepsTheLatestMessages(t *testing.T) {
	var b aiInbox
	for range inboxKept + 3 {
		b.post("hi", t0)
	}
	if got := b.take(context.Background(), 0, t0); len(got) != inboxKept {
		t.Fatalf("kept %d", len(got))
	}
}
