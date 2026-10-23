package weather

import "time"

// DayLength is one whole day and night; it starts at sunrise, which clocks show as 6:00.
const DayLength = 20 * time.Minute

// Sky is what admins changed about the shared clock and weather.
type Sky struct {
	DayOffset time.Duration // added to the wall clock for the time of day, within one day
	Held      bool          // an admin holds the weather at Hold from Since
	Hold      Kind
	Since     time.Time
	Until     time.Time // zero holds it until an admin lets go
}

// Clock is the time to read the time of day from.
func (s Sky) Clock(now time.Time) time.Time { return now.Add(s.DayOffset) }

// Weather is the held weather while the hold lasts, and otherwise the wall clock's.
func (s Sky) Weather(now time.Time) Kind {
	if s.Held && !now.Before(s.Since) && (s.Until.IsZero() || now.Before(s.Until)) {
		return s.Hold
	}
	return At(now)
}

// HourAt is the time of day on a clock, from 0 up to 24.
func HourAt(clock time.Time) float64 {
	day := DayLength.Milliseconds()
	into := float64(clock.UnixMilli()%day) / float64(day)
	return mod(6+24*into, 24)
}

// OffsetToHour is the DayOffset that makes the clock show hour now.
func OffsetToHour(hour float64, now time.Time) time.Duration {
	day := DayLength.Milliseconds()
	want := int64(mod(hour-6, 24) / 24 * float64(day))
	return time.Duration(mod(float64(want-now.UnixMilli()%day), float64(day))) * time.Millisecond
}

func mod(a, b float64) float64 {
	r := a - b*float64(int64(a/b))
	if r < 0 {
		r += b
	}
	return r
}
