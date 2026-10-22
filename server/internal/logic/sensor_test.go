package logic

import (
	"testing"
	"time"

	"scabbage/internal/mobs"
	"scabbage/internal/world"
)

const (
	blockDetectorEast = world.BlockDetector + 2
	clockNoon         = world.Clock + 8
)

func sense(t *testing.T, g grid, s Senses) []Change {
	t.Helper()
	var at []Pos
	for p := range g {
		at = append(at, p)
	}
	changes, err := Sense(g, at, s)
	if err != nil {
		t.Fatal(err)
	}
	return changes
}

// partOfDay is the time a fraction of the way through a day, which starts at sunrise.
func partOfDay(f float64) time.Time {
	return time.UnixMilli(int64(f * float64(mobs.DayLength.Milliseconds())))
}

func TestAPlayerDetectorGivesPowerWhileSomeoneIsNear(t *testing.T) {
	g := line(world.PlayerDetector, wire, lamp)
	sense(t, g, Senses{Players: []Spot{{X: 0.5, Y: 0, Z: 3*world.Metre + 0.5}}})
	if g[Pos{2, 0, 0}] != lit {
		t.Fatalf("not lit: %v", g)
	}
	sense(t, g, Senses{Players: []Spot{{X: 0.5, Y: 0, Z: 5*world.Metre + 0.5}}})
	if g[Pos{2, 0, 0}] != lamp || g[Pos{}] != world.PlayerDetector {
		t.Fatalf("still lit: %v", g)
	}
}

func TestACreatureDetectorNoticesOnlyCreatures(t *testing.T) {
	g := line(world.CreatureDetector, lamp)
	near := []Spot{{X: 2, Y: 1, Z: 0}}
	if sense(t, g, Senses{Players: near}); g[Pos{1, 0, 0}] != lamp {
		t.Fatalf("lit by a player: %v", g)
	}
	if sense(t, g, Senses{Creatures: near}); g[Pos{1, 0, 0}] != lit {
		t.Fatalf("not lit: %v", g)
	}
}

func TestAClockSensorGivesPowerInItsPartOfTheDay(t *testing.T) {
	cases := []struct {
		at   float64
		live []string
	}{
		{0, []string{"day", "dawn"}},
		{0.25, []string{"day", "noon"}},
		{0.5, []string{"day", "dusk"}},
		{0.75, []string{"night", "midnight"}},
	}
	for _, c := range cases {
		for m, mode := range world.ClockModes {
			g := line(world.Clock + world.Block(2*m))
			sense(t, g, Senses{Now: partOfDay(c.at)})
			want := mode == c.live[0] || mode == c.live[1]
			if isOn(g[Pos{}]) != want {
				t.Errorf("%s at %v: on %v", mode, c.at, !want)
			}
		}
	}
}

func TestTappingAClockSensorCyclesItsPartOfTheDay(t *testing.T) {
	g := line(clockNoon + 1)
	if !use(t, g, Pos{}) || g[Pos{}] != world.Clock+10+1 {
		t.Fatalf("grid %v", g)
	}
	use(t, g, Pos{})
	if g[Pos{}] != world.Clock+1 {
		t.Fatalf("didn't wrap: %v", g)
	}
}

func TestABlockDetectorGivesPowerWhileABlockIsInFrontOfIt(t *testing.T) {
	g := line(lamp, blockDetectorEast, world.Stone)
	sense(t, g, Senses{})
	if g[Pos{}] != lit {
		t.Fatalf("not lit: %v", g)
	}
	g[Pos{2, 0, 0}] = world.Water
	sense(t, g, Senses{})
	if g[Pos{}] != lamp {
		t.Fatalf("still lit: %v", g)
	}
}

func TestAWeatherSensorGivesPowerWhileItRainsOrSnows(t *testing.T) {
	g := line(world.WeatherSensor, lamp)
	if sense(t, g, Senses{Wet: true}); g[Pos{1, 0, 0}] != lit {
		t.Fatalf("not lit: %v", g)
	}
	if sense(t, g, Senses{}); g[Pos{1, 0, 0}] != lamp {
		t.Fatalf("still lit: %v", g)
	}
}

func TestSensingLeavesOtherBlocksAlone(t *testing.T) {
	g := line(world.Timer, lamp, world.Stone)
	if changes := sense(t, g, Senses{Players: []Spot{{}}}); len(changes) != 0 {
		t.Fatalf("changes %v", changes)
	}
}

func TestSensorsAreInvolved(t *testing.T) {
	for _, b := range []world.Block{world.PlayerDetector, world.CreatureDetector, clockNoon, blockDetectorEast, world.WeatherSensor} {
		if !Involves(b) {
			t.Errorf("%s not involved", world.Info(b).Name)
		}
	}
}
