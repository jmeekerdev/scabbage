package logic

import (
	"math"
	"time"

	"scabbage/internal/mobs"
	"scabbage/internal/world"
)

// senseRange is how near, in blocks, someone must be for a detector to notice them.
const senseRange = 4 * world.Meter

// Spot is a point in the world, such as a player's feet.
type Spot struct{ X, Y, Z float64 }

// Senses is what sensors can notice: who is where, the time of day and whether anything falls.
type Senses struct {
	Players, Creatures []Spot
	Now                time.Time // the clock to read the time of day from
	Wet                bool
}

// Sense switches each sensor among at on or off to match what it notices.
func Sense(w World, at []Pos, s Senses) ([]Change, error) {
	var out []Change
	for _, p := range at {
		b, err := w.Block(p.X, p.Y, p.Z)
		if err != nil {
			return out, err
		}
		notices, sensor, err := noticed(w, p, b, s)
		if err != nil {
			return out, err
		}
		if !sensor || notices == isOn(b) {
			continue
		}
		changes, err := Set(w, p, switched(b, notices))
		out = append(out, changes...)
		if err != nil {
			return out, err
		}
	}
	return out, nil
}

// noticed reports whether the sensor b at p notices what it watches for, and whether b is a sensor.
func noticed(w World, p Pos, b world.Block, s Senses) (bool, bool, error) {
	switch b.Item() {
	case world.PlayerDetector:
		return anyNear(p, s.Players), true, nil
	case world.CreatureDetector:
		return anyNear(p, s.Creatures), true, nil
	case world.Clock:
		return inPartOfDay(state(b), s.Now), true, nil
	case world.WeatherSensor:
		return s.Wet, true, nil
	case world.BlockDetector:
		f := front(p, b)
		ahead, err := w.Block(f.X, f.Y, f.Z)
		return ahead != world.Air && ahead != world.Water, true, err
	}
	return false, false, nil
}

// anyNear reports whether any of spots is within senseRange of the middle of p.
func anyNear(p Pos, spots []Spot) bool {
	for _, s := range spots {
		dx, dy, dz := s.X-float64(p.X)-0.5, s.Y-float64(p.Y)-0.5, s.Z-float64(p.Z)-0.5
		if dx*dx+dy*dy+dz*dz <= senseRange*senseRange {
			return true
		}
	}
	return false
}

// inPartOfDay reports whether now falls in the part of the day named by world.ClockModes[mode].
func inPartOfDay(mode int, now time.Time) bool {
	day := mobs.DayLength.Milliseconds()
	angle := float64(now.UnixMilli()%day) / float64(day) * 2 * math.Pi
	height, rising := math.Sin(angle), math.Cos(angle) > 0
	low := math.Abs(height) < 0.3
	switch world.ClockModes[mode] {
	case "day":
		return !mobs.Night(now)
	case "night":
		return mobs.Night(now)
	case "dawn":
		return low && rising
	case "dusk":
		return low && !rising
	case "noon":
		return height > 0.9
	}
	return height < -0.9
}

// nextClockMode is the clock sensor b set to the next part of the day, still on or off.
func nextClockMode(b world.Block) world.Block {
	return switched(world.Clock+world.Block(2*((state(b)+1)%len(world.ClockModes))), isOn(b))
}
