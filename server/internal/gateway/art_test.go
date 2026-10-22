package gateway

import (
	"bytes"
	"slices"
	"testing"

	"github.com/coder/websocket"

	"scabbage/internal/art"
	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/world"
)

func submitArt(kind pb.ArtKind, name string, px []byte) *pb.ClientMessage {
	return &pb.ClientMessage{Body: &pb.ClientMessage_SubmitArt{SubmitArt: &pb.SubmitArt{Kind: kind, Name: name, Pixels: px}}}
}

func decideArt(id uint64, ok bool) *pb.ClientMessage {
	return &pb.ClientMessage{Body: &pb.ClientMessage_DecideArt{DecideArt: &pb.DecideArt{Id: id, Approve: ok}}}
}

func opaque(n int, gray byte) []byte { return bytes.Repeat([]byte{gray, gray, gray, 255}, n) }

var (
	blockArt = opaque(art.BlockSize*art.BlockSize, 9)
	skinArt  = opaque(art.SkinWidth*art.SkinHeight, 9)
)

// recvWhere reads messages until one matches, skipping others whose order races with it.
func recvWhere(t *testing.T, c *websocket.Conn, match func(*pb.ServerMessage) bool) *pb.ServerMessage {
	t.Helper()
	for range 10 {
		if msg := recv(t, c); match(msg) {
			return msg
		}
	}
	t.Fatal("expected message never came")
	return nil
}

func noticeOf(contains string) func(*pb.ServerMessage) bool {
	return func(m *pb.ServerMessage) bool {
		return m.GetChat().GetNotice() && bytes.Contains([]byte(m.GetChat().GetText()), []byte(contains))
	}
}

func TestArtKindsMatchTheWire(t *testing.T) {
	if art.Skin != art.Kind(pb.ArtKind_ART_KIND_SKIN) || art.Block != art.Kind(pb.ArtKind_ART_KIND_BLOCK) ||
		art.Texture != art.Kind(pb.ArtKind_ART_KIND_TEXTURE) {
		t.Fatal("art.Kind values differ from pb.ArtKind")
	}
}

func TestApprovedBlockReachesEveryone(t *testing.T) {
	ts := startServer(t)
	admin, kid := ts.admitKid(t)
	send(t, kid, submitArt(pb.ArtKind_ART_KIND_BLOCK, "Flower", blockArt))
	recvWhere(t, kid, noticeOf("admin"))
	reqs := recvWhere(t, admin, func(m *pb.ServerMessage) bool { return len(m.GetArtRequests().GetRequests()) == 1 })
	r := reqs.GetArtRequests().Requests[0]
	if r.Name != "Flower" || r.Author != "Kid" || !bytes.Equal(r.Pixels, blockArt) {
		t.Fatalf("request = %v", r)
	}

	send(t, admin, decideArt(r.Id, true))
	for _, c := range []*websocket.Conn{admin, kid} {
		changed := recvWhere(t, c, func(m *pb.ServerMessage) bool { return m.GetBlocksChanged() != nil }).GetBlocksChanged()
		i := slices.IndexFunc(changed.Blocks, func(b *pb.BlockType) bool { return b.Id == uint32(world.FirstCustom) })
		if i < 0 || changed.Blocks[i].Name != "Flower" || !bytes.Equal(changed.Blocks[i].Pixels, blockArt) {
			t.Fatalf("painted block missing from %v", changed.Blocks)
		}
	}
	recvWhere(t, kid, noticeOf("Flower"))
	if !world.FirstCustom.Valid() {
		t.Fatal("approved block is not registered")
	}
}

func TestNewArrivalsSeeCustomBlocksAndSkins(t *testing.T) {
	ts := startServer(t)
	admin := ts.joinAdmin(t)
	send(t, admin, submitArt(pb.ArtKind_ART_KIND_SKIN, "", skinArt))
	recvWhere(t, admin, noticeOf("look"))
	send(t, admin, submitArt(pb.ArtKind_ART_KIND_TEXTURE, "grass_top", blockArt))
	recvWhere(t, admin, noticeOf("grass_top"))

	kid := ts.requestJoin(t, secret(2), "Kid")
	list := recvWhere(t, admin, func(m *pb.ServerMessage) bool { return len(m.GetPendingRequests().GetRequests()) == 1 })
	send(t, admin, decide(list.GetPendingRequests().Requests[0].Id, true))
	w := recv(t, kid).GetWelcome()
	if tex := w.GetWorld().GetTextures(); len(tex) != 1 || tex[0].Name != "grass_top" {
		t.Fatalf("textures = %v", tex)
	}
	if !bytes.Equal(onlyPlayer(t, recvJoined(t, kid), "Mom").Skin, skinArt) {
		t.Fatal("newcomer doesn't see the admin's skin")
	}
}

func TestApprovedSkinIsShownToOthers(t *testing.T) {
	ts := startServer(t)
	admin, kid := ts.admitKid(t)
	adminEntity := onlyPlayer(t, recvJoined(t, kid), "Mom").Entity
	send(t, kid, submitArt(pb.ArtKind_ART_KIND_SKIN, "", skinArt))
	reqs := recvWhere(t, admin, func(m *pb.ServerMessage) bool { return len(m.GetArtRequests().GetRequests()) == 1 })
	send(t, admin, decideArt(reqs.GetArtRequests().Requests[0].Id, true))
	changed := recvWhere(t, admin, func(m *pb.ServerMessage) bool { return m.GetSkinChanged() != nil }).GetSkinChanged()
	if changed.Entity == adminEntity || !bytes.Equal(changed.Skin, skinArt) {
		t.Fatalf("skin changed = %v", changed)
	}
	recvWhere(t, kid, noticeOf("look"))
}

func TestResetSkinIsShownToOthers(t *testing.T) {
	ts := startServer(t)
	admin, kid := ts.admitKid(t)
	send(t, kid, submitArt(pb.ArtKind_ART_KIND_SKIN, "", nil))
	changed := recvWhere(t, admin, func(m *pb.ServerMessage) bool { return m.GetSkinChanged() != nil }).GetSkinChanged()
	if len(changed.Skin) != 0 {
		t.Fatalf("skin changed = %v", changed)
	}
	recvWhere(t, kid, noticeOf("original look"))
}

func TestBadArtGetsANotice(t *testing.T) {
	ts := startServer(t)
	_, kid := ts.admitKid(t)
	send(t, kid, submitArt(pb.ArtKind_ART_KIND_BLOCK, "Flower", []byte{1, 2, 3}))
	recvWhere(t, kid, noticeOf("wrong size"))
}

func TestOnlyAdminsDecideArt(t *testing.T) {
	ts := startServer(t)
	_, kid := ts.admitKid(t)
	send(t, kid, decideArt(1, true))
	recvWhere(t, kid, func(m *pb.ServerMessage) bool {
		return m.GetDisconnect().GetCode() == pb.DisconnectCode_DISCONNECT_CODE_NOT_ADMIN
	})
}
