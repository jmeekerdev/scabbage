package gateway

import (
	"bytes"
	"context"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"google.golang.org/protobuf/proto"

	"scabbage/internal/access"
	"scabbage/internal/art"
	"scabbage/internal/mobs"
	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/sim"
	"scabbage/internal/store"
	"scabbage/internal/world"
)

const currentVersion = uint32(pb.ProtocolVersion_PROTOCOL_VERSION_CURRENT)

type testServer struct {
	hub   *sim.Hub
	svc   *access.Service
	world *world.World
	herd  *mobs.Herd
	url   string
}

func startServer(t *testing.T) *testServer {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	var gw *Gateway
	hub := sim.NewHub(sim.Config{TickRate: 20, OnTick: func(uint64) { gw.Tick() }})
	t.Cleanup(hub.Close)

	svc := access.New(db, access.Options{OnPendingChange: func() { gw.PendingChanged() }})
	w, err := world.Open(world.NewStore(db), "main", world.Defaults{Seed: 42, Radius: 64})
	if err != nil {
		t.Fatal(err)
	}
	pieces := art.New(db, art.Options{OnPendingChange: func() { gw.ArtChanged() }, Reserved: world.IsBuiltinName})
	herd := mobs.New(1, false) // spawned mobs would race with expected messages
	gw, err = New(Config{Hub: hub, Access: svc, Art: pieces, World: w, Mobs: herd, ServerName: "test", HelloTimeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(gw)
	t.Cleanup(srv.Close)
	return &testServer{hub: hub, svc: svc, world: w, herd: herd, url: "ws" + strings.TrimPrefix(srv.URL, "http")}
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

func secret(b byte) []byte { return bytes.Repeat([]byte{b}, access.SecretBytes) }

func hello(s []byte, join *pb.JoinRequest) *pb.ClientMessage {
	return &pb.ClientMessage{Body: &pb.ClientMessage_Hello{Hello: &pb.Hello{
		ProtocolVersion: currentVersion, ClientName: "tester", DeviceSecret: s, Join: join,
	}}}
}

func ping(ms float64) *pb.ClientMessage {
	return &pb.ClientMessage{Body: &pb.ClientMessage_Ping{Ping: &pb.Ping{ClientTimeMs: ms}}}
}

func decide(id uint64, approve bool) *pb.ClientMessage {
	return &pb.ClientMessage{Body: &pb.ClientMessage_DecideJoin{DecideJoin: &pb.DecideJoin{RequestId: id, Approve: approve}}}
}

func expectDisconnect(t *testing.T, c *websocket.Conn, want pb.DisconnectCode) {
	t.Helper()
	if d := recv(t, c).GetDisconnect(); d == nil || d.Code != want {
		t.Fatalf("got %v, want Disconnect %v", d, want)
	}
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

// joinAdmin connects as the first admin via the bootstrap invite and returns the connection.
func (ts *testServer) joinAdmin(t *testing.T) *websocket.Conn {
	t.Helper()
	code, _, err := ts.svc.AdminInvite()
	if err != nil {
		t.Fatal(err)
	}
	c := dial(t, ts.url)
	send(t, c, hello(secret(1), &pb.JoinRequest{Code: code, Name: "Mom", Color: 7}))
	w := recv(t, c).GetWelcome()
	if w == nil || !w.You.Admin || w.You.Name != "Mom" {
		t.Fatalf("admin welcome = %v", w)
	}
	if p := recv(t, c).GetPendingRequests(); p == nil || len(p.Requests) != 0 {
		t.Fatalf("admin should get an empty pending list, got %v", p)
	}
	if a := recv(t, c).GetArtRequests(); a == nil {
		t.Fatal("admin should get the art waiting for approval")
	}
	if p := recv(t, c).GetPlayerList(); len(p.GetPlayers()) != 1 {
		t.Fatalf("admin should get the player list, got %v", p)
	}
	return c
}

// joinConsole connects the admin's device as the admin console, past its opening messages.
func (ts *testServer) joinConsole(t *testing.T) *websocket.Conn {
	t.Helper()
	_ = ts.joinAdmin(t).Close(websocket.StatusNormalClosure, "")
	waitPlayers(t, ts.hub, 0)
	c := dial(t, ts.url)
	msg := hello(secret(1), nil)
	msg.GetHello().Console = true
	send(t, c, msg)
	recv(t, c) // welcome
	recv(t, c) // pending joins
	recv(t, c) // pending art
	recv(t, c) // players
	return c
}

// requestJoin connects a new device with a fresh regular code and expects JoinPending.
func (ts *testServer) requestJoin(t *testing.T, s []byte, name string) *websocket.Conn {
	t.Helper()
	code, err := ts.svc.CreateCode(1, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	c := dial(t, ts.url)
	send(t, c, hello(s, &pb.JoinRequest{Code: code.Code, Name: name, Color: 3}))
	if recv(t, c).GetJoinPending() == nil {
		t.Fatal("expected JoinPending")
	}
	return c
}

func TestAdminWelcomeJoinsHubAndLeavingIdles(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	waitPlayers(t, ts.hub, 1)

	_ = c.Close(websocket.StatusNormalClosure, "")
	waitPlayers(t, ts.hub, 0)
	if ts.hub.Running() {
		t.Fatal("hub should be idle after last player disconnects")
	}
}

func TestRegisteredDeviceReconnects(t *testing.T) {
	ts := startServer(t)
	_ = ts.joinAdmin(t).Close(websocket.StatusNormalClosure, "")

	c := dial(t, ts.url)
	send(t, c, hello(secret(1), nil))
	if w := recv(t, c).GetWelcome(); w == nil || w.You.Name != "Mom" || w.TickRateHz != 20 || w.ServerName != "test" {
		t.Fatalf("welcome = %v", w)
	}
}

func TestUnknownDeviceNeedsJoinCode(t *testing.T) {
	ts := startServer(t)
	c := dial(t, ts.url)
	send(t, c, hello(secret(9), nil))
	expectDisconnect(t, c, pb.DisconnectCode_DISCONNECT_CODE_NEED_JOIN_CODE)
}

func TestBadJoinCodeIsRejected(t *testing.T) {
	ts := startServer(t)
	c := dial(t, ts.url)
	send(t, c, hello(secret(9), &pb.JoinRequest{Code: "NOPE2345", Name: "Kid"}))
	expectDisconnect(t, c, pb.DisconnectCode_DISCONNECT_CODE_BAD_JOIN_CODE)
}

func TestApprovalLetsPendingPlayerIn(t *testing.T) {
	ts := startServer(t)
	admin := ts.joinAdmin(t)
	kid := ts.requestJoin(t, secret(2), "Kid")

	list := recv(t, admin).GetPendingRequests()
	if list == nil || len(list.Requests) != 1 || list.Requests[0].Name != "Kid" {
		t.Fatalf("admin pending list = %v", list)
	}
	send(t, kid, ping(5))
	if recv(t, kid).GetPong() == nil {
		t.Fatal("pending players should still get pongs")
	}

	send(t, admin, decide(list.Requests[0].Id, true))
	if w := recv(t, kid).GetWelcome(); w == nil || w.You.Name != "Kid" || w.You.Admin {
		t.Fatalf("kid welcome = %v", w)
	}
	if list := recv(t, admin).GetPendingRequests(); list == nil || len(list.Requests) != 0 {
		t.Fatalf("pending list after approval = %v", list)
	}
	waitPlayers(t, ts.hub, 2)
}

func TestDenialDisconnectsPendingPlayer(t *testing.T) {
	ts := startServer(t)
	admin := ts.joinAdmin(t)
	kid := ts.requestJoin(t, secret(2), "Kid")
	list := recv(t, admin).GetPendingRequests()

	send(t, admin, decide(list.Requests[0].Id, false))
	expectDisconnect(t, kid, pb.DisconnectCode_DISCONNECT_CODE_DENIED)
}

func TestNonAdminCannotDecide(t *testing.T) {
	ts := startServer(t)
	admin := ts.joinAdmin(t)
	kid := ts.requestJoin(t, secret(2), "Kid")
	list := recv(t, admin).GetPendingRequests()
	send(t, admin, decide(list.Requests[0].Id, true))
	recv(t, kid) // Welcome
	recvJoined(t, kid)

	send(t, kid, decide(1, true))
	expectDisconnect(t, kid, pb.DisconnectCode_DISCONNECT_CODE_NOT_ADMIN)
}

func TestAdminCreatesUsableJoinCode(t *testing.T) {
	ts := startServer(t)
	admin := ts.joinAdmin(t)
	send(t, admin, &pb.ClientMessage{Body: &pb.ClientMessage_CreateJoinCode{CreateJoinCode: &pb.CreateJoinCode{Uses: 2, TtlHours: 1}}})
	created := recv(t, admin).GetJoinCodeCreated()
	if created == nil || created.Uses != 2 || len(created.Code) == 0 {
		t.Fatalf("created = %v", created)
	}

	c := dial(t, ts.url)
	send(t, c, hello(secret(3), &pb.JoinRequest{Code: created.Code, Name: "Dad"}))
	if recv(t, c).GetJoinPending() == nil {
		t.Fatal("expected JoinPending with the admin's code")
	}
}

func TestConsoleRequiresAdminAndSkipsHub(t *testing.T) {
	ts := startServer(t)
	_ = ts.joinAdmin(t).Close(websocket.StatusNormalClosure, "")
	waitPlayers(t, ts.hub, 0)

	c := dial(t, ts.url)
	msg := hello(secret(1), nil)
	msg.GetHello().Console = true
	send(t, c, msg)
	if recv(t, c).GetWelcome() == nil {
		t.Fatal("expected console Welcome")
	}
	time.Sleep(20 * time.Millisecond)
	if ts.hub.Players() != 0 {
		t.Fatal("console sessions must not wake the world")
	}
}

func TestConsoleRejectsNonAdmin(t *testing.T) {
	ts := startServer(t)
	admin := ts.joinAdmin(t)
	kid := ts.requestJoin(t, secret(2), "Kid")
	list := recv(t, admin).GetPendingRequests()
	send(t, admin, decide(list.Requests[0].Id, true))
	recv(t, kid)
	_ = kid.Close(websocket.StatusNormalClosure, "")

	c := dial(t, ts.url)
	msg := hello(secret(2), nil)
	msg.GetHello().Console = true
	send(t, c, msg)
	expectDisconnect(t, c, pb.DisconnectCode_DISCONNECT_CODE_NOT_ADMIN)
}

func TestVersionMismatchIsRejected(t *testing.T) {
	ts := startServer(t)
	c := dial(t, ts.url)
	msg := hello(secret(1), nil)
	msg.GetHello().ProtocolVersion++
	send(t, c, msg)
	expectDisconnect(t, c, pb.DisconnectCode_DISCONNECT_CODE_PROTOCOL)
	waitPlayers(t, ts.hub, 0)
}

func TestMessageBeforeHelloIsRejected(t *testing.T) {
	ts := startServer(t)
	c := dial(t, ts.url)
	send(t, c, ping(1))
	expectDisconnect(t, c, pb.DisconnectCode_DISCONNECT_CODE_PROTOCOL)
}

func TestPingGetsPong(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	send(t, c, ping(1234.5))
	p := recv(t, c).GetPong()
	if p == nil || p.ClientTimeMs != 1234.5 || p.ServerTimeMs <= 0 {
		t.Fatalf("unexpected pong: %v", p)
	}
}
