package world

import (
	"testing"
	"time"

	"scabbage/internal/weather"
)

func TestSkyIsSavedWithTheWorld(t *testing.T) {
	db := openDB(t)
	w := openWorld(t, db)
	if got, err := w.Sky(); err != nil || got != (weather.Sky{}) {
		t.Fatalf("a new world's sky: got %+v, %v", got, err)
	}
	since := time.UnixMilli(5000)
	want := weather.Sky{DayOffset: 90 * time.Second, Held: true, Hold: weather.Storm, Since: since}
	if err := w.SetSky(want); err != nil {
		t.Fatal(err)
	}
	got, err := openWorld(t, db).Sky()
	if err != nil || got.DayOffset != want.DayOffset || !got.Held || got.Hold != weather.Storm ||
		!got.Since.Equal(since) || !got.Until.IsZero() {
		t.Fatalf("got %+v, %v; want %+v", got, err, want)
	}
}
