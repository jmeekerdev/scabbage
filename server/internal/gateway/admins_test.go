package gateway

import (
	"context"
	"testing"
	"time"

	"github.com/coder/websocket"

	pb "scabbage/internal/pb/scabbage/v1"
)

func setAdmin(id uint64, admin bool) *pb.ClientMessage {
	return &pb.ClientMessage{Body: &pb.ClientMessage_SetAdmin{SetAdmin: &pb.SetAdmin{PlayerId: id, Admin: admin}}}
}

func playerListOf(n int) func(*pb.ServerMessage) bool {
	return func(m *pb.ServerMessage) bool { return len(m.GetPlayerList().GetPlayers()) == n }
}

// playerNamed finds name in the admin's latest two-player list.
func playerNamed(t *testing.T, admin *websocket.Conn, name string) *pb.PlayerInfo {
	t.Helper()
	for _, p := range recvWhere(t, admin, playerListOf(2)).GetPlayerList().Players {
		if p.Name == name {
			return p
		}
	}
	t.Fatalf("%s is not in the player list", name)
	return nil
}

// expectClosed waits for the server to drop c without a Disconnect message.
func expectClosed(t *testing.T, c *websocket.Conn) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second) // past a wipe's kickDelay
	defer cancel()
	for {
		if _, _, err := c.Read(ctx); err != nil {
			if ctx.Err() != nil {
				t.Fatal("connection stayed open")
			}
			return
		}
	}
}

func TestAdminsGetThePlayerList(t *testing.T) {
	ts := startServer(t)
	admin, _ := ts.admitKid(t)
	if kid := playerNamed(t, admin, "Kid"); kid.Admin {
		t.Fatalf("kid = %v", kid)
	}
}

func TestPromotedPlayerReconnectsAsAdmin(t *testing.T) {
	ts := startServer(t)
	admin, kid := ts.admitKid(t)
	send(t, admin, setAdmin(playerNamed(t, admin, "Kid").Id, true))
	recvWhere(t, admin, noticeOf("Kid is an admin now"))
	if !playerNamed(t, admin, "Kid").Admin {
		t.Fatal("list doesn't show Kid as an admin")
	}
	expectClosed(t, kid)

	again := dial(t, ts.url)
	send(t, again, hello(secret(2), nil))
	if w := recv(t, again).GetWelcome(); !w.GetYou().GetAdmin() {
		t.Fatalf("welcome = %v, want admin", w)
	}
}

func TestAdminsCantChangeThemselves(t *testing.T) {
	ts := startServer(t)
	admin, _ := ts.admitKid(t)
	send(t, admin, setAdmin(playerNamed(t, admin, "Mom").Id, false))
	recvWhere(t, admin, noticeOf("your own"))
}

func TestOnlyAdminsSetAdmin(t *testing.T) {
	ts := startServer(t)
	_, kid := ts.admitKid(t)
	send(t, kid, setAdmin(2, true))
	recvWhere(t, kid, func(m *pb.ServerMessage) bool {
		return m.GetDisconnect().GetCode() == pb.DisconnectCode_DISCONNECT_CODE_NOT_ADMIN
	})
}

func TestAdminCommandsWorkByName(t *testing.T) {
	ts := startServer(t)
	admin, kid := ts.admitKid(t)
	send(t, admin, sendChat("/admin kid"))
	recvWhere(t, admin, noticeOf("Kid is an admin now"))
	expectClosed(t, kid)
	send(t, admin, sendChat("/unadmin KID"))
	recvWhere(t, admin, noticeOf("Kid is no longer an admin"))
}

func TestAdminCommandNeedsAKnownName(t *testing.T) {
	ts := startServer(t)
	admin := ts.joinAdmin(t)
	send(t, admin, sendChat("/admin Nobody"))
	expectNotice(t, admin, "Nobody")
}
