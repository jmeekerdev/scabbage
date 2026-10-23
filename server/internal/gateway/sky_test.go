package gateway

import (
	"testing"
	"time"

	"github.com/coder/websocket"

	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/weather"
)

func recvSky(t *testing.T, c *websocket.Conn) *pb.Sky {
	t.Helper()
	return recvWhere(t, c, func(m *pb.ServerMessage) bool { return m.GetSky() != nil }).GetSky()
}

func TestTimeMovesTheClockForEveryoneAndIsSaved(t *testing.T) {
	ts := startServer(t)
	admin, kid := ts.joinKid(t)
	send(t, admin, sendChat("/time noon"))
	for _, c := range []*websocket.Conn{admin, kid} {
		sky := recvSky(t, c)
		clock := time.Now().Add(time.Duration(sky.DayOffsetMs) * time.Millisecond)
		if h := weather.HourAt(clock); h < 11.9 || h > 12.1 {
			t.Fatalf("hour after /time noon = %v", h)
		}
		recvWhere(t, c, noticeOf("12:00"))
	}
	saved, err := ts.gw.cfg.World.Sky()
	if err != nil || saved.DayOffset == 0 {
		t.Fatalf("saved sky = %+v, %v", saved, err)
	}
	_ = kid.Close(websocket.StatusNormalClosure, "")
	if w := ts.rejoin(t, admin); w.GetSky().GetDayOffsetMs() != saved.DayOffset.Milliseconds() {
		t.Fatalf("welcome sky = %v", w.GetSky())
	}
}

func TestWeatherHoldsForMinutesAndAutoLetsGo(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	send(t, c, sendChat("/weather storm 5"))
	sky := recvSky(t, c)
	left := time.Until(time.UnixMilli(sky.HeldUntilMs))
	if sky.Held != pb.Weather_WEATHER_STORM || left < 4*time.Minute || left > 5*time.Minute {
		t.Fatalf("after /weather storm 5: %v", sky)
	}
	recvWhere(t, c, noticeOf("storm"))
	send(t, c, sendChat("/weather auto"))
	sky = recvSky(t, c)
	if time.Until(time.UnixMilli(sky.HeldUntilMs)) > time.Second {
		t.Fatalf("after /weather auto: %v", sky)
	}
}

func TestTimeAndWeatherAreForAdminsAndExplainThemselves(t *testing.T) {
	ts := startServer(t)
	admin, kid := ts.joinKid(t)
	send(t, kid, sendChat("/weather rain"))
	expectNotice(t, kid, "Only admins")
	send(t, admin, sendChat("/time teatime"))
	expectNotice(t, admin, "/time dawn")
	send(t, admin, sendChat("/weather hail"))
	expectNotice(t, admin, "/weather clear")
}
