package gateway

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/weather"
)

const (
	timeHelp    = "Try /time dawn, day, noon, dusk, night or midnight, or a time like /time 18:30."
	weatherHelp = "Try /weather clear, rain or storm, with minutes to hold it (/weather rain 10), or /weather auto."
)

// Names for parts of the day, as hours on the clock.
var dayHours = map[string]float64{"dawn": 6, "day": 9, "morning": 9, "noon": 12, "dusk": 18, "evening": 18, "night": 21, "midnight": 0}

var weatherNames = map[string]weather.Kind{"clear": weather.Clear, "rain": weather.Rain, "snow": weather.Rain, "storm": weather.Storm}

// sky is the time and weather admins set.
func (g *Gateway) sky() weather.Sky { return *g.skyNow.Load() }

// setSky saves sky and tells everyone, with a line saying what changed.
func (g *Gateway) setSky(sky weather.Sky, said string) error {
	if err := g.cfg.World.SetSky(sky); err != nil {
		return err
	}
	g.skyNow.Store(&sky)
	g.mu.Lock()
	defer g.mu.Unlock()
	g.broadcastLocked(&pb.ServerMessage{Body: &pb.ServerMessage_Sky{Sky: skyMessage(sky)}})
	g.broadcastLocked(noticeMessage(said))
	return nil
}

func skyMessage(s weather.Sky) *pb.Sky {
	out := &pb.Sky{DayOffsetMs: s.DayOffset.Milliseconds()}
	if s.Held {
		out.Held = pb.Weather(s.Hold + 1)
		out.HeldSinceMs = s.Since.UnixMilli()
	}
	if !s.Until.IsZero() {
		out.HeldUntilMs = s.Until.UnixMilli()
	}
	return out
}

// timeCommand moves the clock everyone shares to a part of the day or an hour.
func (s *session) timeCommand(ctx context.Context, arg string) error {
	sky := s.gw.sky()
	now := time.Now()
	arg = strings.ToLower(strings.TrimSpace(arg))
	if arg == "" {
		return s.notice(ctx, fmt.Sprintf("It's %s. %s", clockText(weather.HourAt(sky.Clock(now))), timeHelp))
	}
	hour, ok := dayHours[arg]
	if !ok {
		if hour, ok = parseHour(arg); !ok {
			return s.notice(ctx, timeHelp)
		}
	}
	sky.DayOffset = weather.OffsetToHour(hour, now)
	return s.gw.setSky(sky, fmt.Sprintf("🕒 %s set the time to %s.", s.player.Name, clockText(hour)))
}

// weatherCommand holds the weather, for some minutes or until /weather auto lets it go.
func (s *session) weatherCommand(ctx context.Context, arg string) error {
	sky := s.gw.sky()
	now := time.Now()
	f := strings.Fields(strings.ToLower(arg))
	switch {
	case len(f) == 0:
		return s.notice(ctx, fmt.Sprintf("It's %s. %s", sky.Weather(now), weatherHelp))
	case len(f) == 1 && f[0] == "auto":
		if sky.Held && (sky.Until.IsZero() || sky.Until.After(now)) {
			sky.Until = now
		}
		return s.gw.setSky(sky, fmt.Sprintf("🌦️ %s let the weather change by itself again.", s.player.Name))
	}
	kind, ok := weatherNames[f[0]]
	minutes := 0.0
	if ok && len(f) == 2 {
		var err error
		minutes, err = strconv.ParseFloat(f[1], 64)
		ok = err == nil && minutes > 0 && minutes <= 24*60
	}
	if !ok || len(f) > 2 {
		return s.notice(ctx, weatherHelp)
	}
	sky.Held, sky.Hold, sky.Since, sky.Until = true, kind, now, time.Time{}
	said := fmt.Sprintf("🌦️ %s set the weather to %s until /weather auto.", s.player.Name, kind)
	if minutes > 0 {
		sky.Until = now.Add(time.Duration(minutes * float64(time.Minute)))
		said = fmt.Sprintf("🌦️ %s set the weather to %s for %v minutes.", s.player.Name, kind, minutes)
	}
	return s.gw.setSky(sky, said)
}

// parseHour reads "18", "18:30" or "6:05" as hours from 0 up to 24.
func parseHour(s string) (float64, bool) {
	h, m, hasMinutes := strings.Cut(s, ":")
	hours, err := strconv.Atoi(h)
	if err != nil || hours < 0 || hours > 23 {
		return 0, false
	}
	minutes := 0
	if hasMinutes {
		if minutes, err = strconv.Atoi(m); err != nil || len(m) != 2 || minutes > 59 {
			return 0, false
		}
	}
	return float64(hours) + float64(minutes)/60, true
}

// clockText shows hours as a 24-hour clock, like 18:30.
func clockText(hour float64) string {
	minutes := int(math.Round(hour*60)) % (24 * 60)
	return fmt.Sprintf("%d:%02d", minutes/60, minutes%60)
}
