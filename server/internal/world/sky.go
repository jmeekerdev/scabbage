package world

import (
	"time"

	"scabbage/internal/weather"
)

// Sky is the time and weather admins set for this world.
func (w *World) Sky() (weather.Sky, error) {
	var offset, held, since, until int64
	err := w.store.db.QueryRow(`SELECT day_offset_ms, held_weather, held_since_ms, held_until_ms
		FROM worlds WHERE id = ?`, w.meta.ID).Scan(&offset, &held, &since, &until)
	if err != nil {
		return weather.Sky{}, err
	}
	s := weather.Sky{DayOffset: time.Duration(offset) * time.Millisecond}
	if held >= 0 {
		s.Held, s.Hold, s.Since = true, weather.Kind(held), time.UnixMilli(since)
	}
	if until != 0 {
		s.Until = time.UnixMilli(until)
	}
	return s, nil
}

// SetSky saves the time and weather admins set.
func (w *World) SetSky(s weather.Sky) error {
	held, since, until := int64(-1), int64(0), int64(0)
	if s.Held {
		held, since = int64(s.Hold), s.Since.UnixMilli()
	}
	if !s.Until.IsZero() {
		until = s.Until.UnixMilli()
	}
	_, err := w.store.db.Exec(`UPDATE worlds SET day_offset_ms = ?, held_weather = ?, held_since_ms = ?,
		held_until_ms = ? WHERE id = ?`, s.DayOffset.Milliseconds(), held, since, until, w.meta.ID)
	return err
}
