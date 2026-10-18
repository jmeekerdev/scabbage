package gateway

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"google.golang.org/protobuf/proto"

	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/sim"
)

const currentVersion = uint32(pb.ProtocolVersion_PROTOCOL_VERSION_CURRENT)

func startServer(t *testing.T) (*sim.Hub, string) {
	t.Helper()
	hub := sim.NewHub(sim.Config{TickRate: 20})
	t.Cleanup(hub.Close)
	srv := httptest.NewServer(New(Config{Hub: hub, ServerName: "test", HelloTimeout: time.Second}))
	t.Cleanup(srv.Close)
	return hub, "ws" + strings.TrimPrefix(srv.URL, "http")
}

func dial(t *testing.T, url string) *websocket.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	c, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = c.CloseNow() })
	return c
}

func send(t *testing.T, c *websocket.Conn, msg *pb.ClientMessage) {
	t.Helper()
	b, err := proto.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := c.Write(ctx, websocket.MessageBinary, b); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func recv(t *testing.T, c *websocket.Conn) *pb.ServerMessage {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, b, err := c.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var msg pb.ServerMessage
	if err := proto.Unmarshal(b, &msg); err != nil {
		t.Fatal(err)
	}
	return &msg
}

func hello(version uint32) *pb.ClientMessage {
	return &pb.ClientMessage{Body: &pb.ClientMessage_Hello{Hello: &pb.Hello{ProtocolVersion: version, ClientName: "tester"}}}
}

func waitPlayers(t *testing.T, hub *sim.Hub, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for hub.Players() != want {
		if time.Now().After(deadline) {
			t.Fatalf("players = %d, want %d", hub.Players(), want)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestHelloGetsWelcomeAndJoinsHub(t *testing.T) {
	hub, url := startServer(t)
	c := dial(t, url)
	send(t, c, hello(currentVersion))

	w := recv(t, c).GetWelcome()
	if w == nil {
		t.Fatal("expected Welcome")
	}
	if w.ProtocolVersion != currentVersion || w.TickRateHz != 20 || w.ServerName != "test" {
		t.Fatalf("unexpected welcome: %v", w)
	}
	waitPlayers(t, hub, 1)

	_ = c.Close(websocket.StatusNormalClosure, "")
	waitPlayers(t, hub, 0)
	if hub.Running() {
		t.Fatal("hub should be idle after last player disconnects")
	}
}

func TestVersionMismatchIsRejected(t *testing.T) {
	hub, url := startServer(t)
	c := dial(t, url)
	send(t, c, hello(currentVersion+1))

	if d := recv(t, c).GetDisconnect(); d == nil || !strings.Contains(d.Reason, "version") {
		t.Fatalf("expected version Disconnect, got %v", d)
	}
	waitPlayers(t, hub, 0)
}

func TestMessageBeforeHelloIsRejected(t *testing.T) {
	_, url := startServer(t)
	c := dial(t, url)
	send(t, c, &pb.ClientMessage{Body: &pb.ClientMessage_Ping{Ping: &pb.Ping{ClientTimeMs: 1}}})

	if d := recv(t, c).GetDisconnect(); d == nil {
		t.Fatal("expected Disconnect")
	}
}

func TestPingGetsPong(t *testing.T) {
	_, url := startServer(t)
	c := dial(t, url)
	send(t, c, hello(currentVersion))
	recv(t, c)
	send(t, c, &pb.ClientMessage{Body: &pb.ClientMessage_Ping{Ping: &pb.Ping{ClientTimeMs: 1234.5}}})

	p := recv(t, c).GetPong()
	if p == nil || p.ClientTimeMs != 1234.5 || p.ServerTimeMs <= 0 {
		t.Fatalf("unexpected pong: %v", p)
	}
}
