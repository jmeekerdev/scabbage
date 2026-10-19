package gateway

import (
	"strings"
	"testing"

	"github.com/coder/websocket"

	pb "scabbage/internal/pb/scabbage/v1"
)

func sendChat(text string) *pb.ClientMessage {
	return &pb.ClientMessage{Body: &pb.ClientMessage_SendChat{SendChat: &pb.SendChat{Text: text}}}
}

func sendEmote(e pb.Emote) *pb.ClientMessage {
	return &pb.ClientMessage{Body: &pb.ClientMessage_SendEmote{SendEmote: &pb.SendEmote{Emote: e}}}
}

func recvChat(t *testing.T, c *websocket.Conn) *pb.Chat {
	t.Helper()
	msg := recv(t, c)
	if msg.GetChat() == nil {
		t.Fatalf("got %v, want Chat", msg)
	}
	return msg.GetChat()
}

func expectNotice(t *testing.T, c *websocket.Conn, contains string) {
	t.Helper()
	if ch := recvChat(t, c); !ch.Notice || !strings.Contains(ch.Text, contains) {
		t.Fatalf("got %v, want a notice containing %q", ch, contains)
	}
}

func TestChatReachesEveryoneIncludingTheSender(t *testing.T) {
	ts := startServer(t)
	admin, kid := ts.joinKid(t)

	send(t, kid, sendChat("  hi\nmom "))
	for _, c := range []*websocket.Conn{kid, admin} {
		ch := recvChat(t, c)
		if ch.Name != "Kid" || ch.Color != 3 || ch.Text != "hi mom" || ch.Notice {
			t.Fatalf("chat = %v", ch)
		}
	}
}

func TestBlankChatIsIgnored(t *testing.T) {
	ts := startServer(t)
	admin, _ := ts.joinKid(t)
	send(t, admin, sendChat(" \n "))
	expectPong(t, admin)
}

func TestFastChatAsksToSlowDown(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	for range 5 {
		send(t, c, sendChat("spam"))
		recvChat(t, c)
	}
	send(t, c, sendChat("spam"))
	expectNotice(t, c, "Slow down")
}

func TestEmoteIsShownToOthers(t *testing.T) {
	ts := startServer(t)
	admin, kid := ts.joinKid(t)
	send(t, admin, sendEmote(pb.Emote_EMOTE_WAVE))
	e := recv(t, kid).GetEmoteShown()
	if e == nil || e.Emote != pb.Emote_EMOTE_WAVE || e.Entity != 1 {
		t.Fatalf("emote = %v, want a wave from entity 1", e)
	}
	expectPong(t, admin) // the sender gets nothing back
}

func TestUnknownEmoteIsIgnored(t *testing.T) {
	ts := startServer(t)
	admin, kid := ts.joinKid(t)
	send(t, admin, sendEmote(pb.Emote(99)))
	expectPong(t, kid)
}

func TestApproveCommandLetsPendingPlayerIn(t *testing.T) {
	ts := startServer(t)
	admin := ts.joinAdmin(t)
	kid := ts.requestJoin(t, secret(2), "Kid")
	recv(t, admin) // pending list

	send(t, admin, sendChat("/approve kid"))
	if recv(t, kid).GetWelcome() == nil {
		t.Fatal("expected Welcome")
	}
	waitPlayers(t, ts.hub, 2)
}

func TestApproveCommandNeedsAWaitingName(t *testing.T) {
	ts := startServer(t)
	admin := ts.joinAdmin(t)
	send(t, admin, sendChat("/approve Nobody"))
	expectNotice(t, admin, "Nobody")
}

func TestCommandsNeedAnAdmin(t *testing.T) {
	ts := startServer(t)
	_, kid := ts.joinKid(t)
	send(t, kid, sendChat("/deny Mom"))
	expectNotice(t, kid, "admins")
}

func TestUnknownCommandGetsHelp(t *testing.T) {
	ts := startServer(t)
	admin := ts.joinAdmin(t)
	send(t, admin, sendChat("/fly"))
	expectNotice(t, admin, "/approve")
}
