package weather

import (
	"testing"
	"time"
)

func at(spell int64) time.Time { return time.UnixMilli(spell*Spell.Milliseconds() + 1) }

// The client's weather.test.ts checks the same spells, so both always agree.
func TestSpellsMatchTheClient(t *testing.T) {
	want := []Kind{
		Clear, Clear, Clear, Rain, Clear, Rain, Rain, Clear, Clear, Rain, Rain, Clear,
		Clear, Clear, Rain, Clear, Clear, Storm, Clear, Rain,
	}
	for i, k := range want {
		if got := At(at(int64(i))); got != k {
			t.Errorf("spell %d: got %v, want %v", i, got, k)
		}
	}
}

func TestWeatherHoldsForASpell(t *testing.T) {
	start := at(1000)
	if At(start) != At(start.Add(Spell-2*time.Millisecond)) {
		t.Fatal("weather changed within a spell")
	}
}

func TestWeatherIsMostlyClear(t *testing.T) {
	count := map[Kind]int{}
	for i := range int64(1000) {
		count[At(at(i))]++
	}
	if count[Clear] < 400 || count[Rain] < 250 || count[Storm] < 80 {
		t.Fatalf("unexpected mix %v", count)
	}
}

func TestOnlyClearIsDry(t *testing.T) {
	if Clear.Wet() || !Rain.Wet() || !Storm.Wet() {
		t.Fatal("rain and storms are wet, clear skies are not")
	}
}
