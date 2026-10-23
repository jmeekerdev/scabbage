package weather

import (
	"testing"
	"time"
)

func TestHeldWeatherLastsFromSinceUntilUntil(t *testing.T) {
	now := time.UnixMilli(1_000_000_000)
	auto := At(now)
	other := Storm
	if auto == Storm {
		other = Clear
	}
	s := Sky{Held: true, Hold: other, Since: now, Until: now.Add(time.Minute)}
	if got := s.Weather(now.Add(time.Second)); got != other {
		t.Fatalf("while held: got %v, want %v", got, other)
	}
	if got := s.Weather(now.Add(-time.Second)); got != At(now.Add(-time.Second)) {
		t.Fatalf("before the hold: got %v", got)
	}
	if got := s.Weather(now.Add(time.Minute)); got != At(now.Add(time.Minute)) {
		t.Fatalf("after the hold: got %v", got)
	}
	s.Until = time.Time{}
	if got := s.Weather(now.Add(24 * time.Hour)); got != other {
		t.Fatalf("held for good: got %v, want %v", got, other)
	}
}

func TestOffsetToHourPutsTheClockThere(t *testing.T) {
	now := time.UnixMilli(1_234_567_890)
	for _, hour := range []float64{0, 6, 12, 18, 21.5} {
		s := Sky{DayOffset: OffsetToHour(hour, now)}
		if got := HourAt(s.Clock(now)); got < hour-0.01 || got > hour+0.01 {
			t.Errorf("asked for %v, got %v", hour, got)
		}
		if s.DayOffset < 0 || s.DayOffset >= DayLength {
			t.Errorf("offset %v is outside one day", s.DayOffset)
		}
	}
}

func TestTheDayStartsAtSixInTheMorning(t *testing.T) {
	if got := HourAt(time.UnixMilli(0)); got != 6 {
		t.Fatalf("got %v, want 6", got)
	}
	if got := HourAt(time.UnixMilli(DayLength.Milliseconds() / 4)); got != 12 {
		t.Fatalf("a quarter day in: got %v, want 12", got)
	}
}
