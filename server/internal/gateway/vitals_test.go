package gateway

import (
	"testing"

	"github.com/coder/websocket"

	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/vitals"
	"scabbage/internal/world"
)

func setGameMode(id uint64, m pb.GameMode) *pb.ClientMessage {
	return &pb.ClientMessage{Body: &pb.ClientMessage_SetGameMode{SetGameMode: &pb.SetGameMode{PlayerId: id, Mode: m}}}
}

func vitalsWith(health uint32) func(*pb.ServerMessage) bool {
	return func(m *pb.ServerMessage) bool { return m.GetVitals() != nil && m.GetVitals().Health == health }
}

// survivalAdmin joins as the admin and switches to survival, reading the replies, which race.
func (ts *testServer) survivalAdmin(t *testing.T) *websocket.Conn {
	t.Helper()
	c := ts.joinAdmin(t)
	send(t, c, setGameMode(1, pb.GameMode_GAME_MODE_SURVIVAL))
	for range 3 {
		msg := recv(t, c)
		if v := msg.GetVitals(); v != nil && (v.Mode != pb.GameMode_GAME_MODE_SURVIVAL || v.Health != vitals.Max) {
			t.Fatalf("vitals = %v", v)
		}
		if msg.GetVitals() == nil && !noticeOf("Mom plays survival now")(msg) && msg.GetPlayerList() == nil {
			t.Fatalf("got %v", msg)
		}
	}
	return c
}

// dropTen jumps the player up ten blocks above spawn and lands back on it.
func (ts *testServer) dropTen(t *testing.T, c *websocket.Conn) {
	x, y, z := ts.spawn()
	send(t, c, move(x, y+10, z, 0))
	send(t, c, move(x, y, z, 0))
}

func TestWelcomeCarriesTheServerModeAndFullHealth(t *testing.T) {
	ts := startServer(t)
	c := dial(t, ts.url)
	code, _, _ := ts.svc.AdminInvite()
	send(t, c, hello(secret(1), &pb.JoinRequest{Code: code, Name: "Mom", Color: 7}))
	w := recv(t, c).GetWelcome()
	if w.Vitals.GetMode() != pb.GameMode_GAME_MODE_CREATIVE || w.Vitals.GetHealth() != vitals.Max {
		t.Fatalf("vitals = %v", w.Vitals)
	}
	if w.You.Mode != pb.GameMode_GAME_MODE_CREATIVE {
		t.Fatalf("you = %v", w.You)
	}
}

func TestLongFallsHurtInSurvival(t *testing.T) {
	ts := startServer(t)
	c := ts.survivalAdmin(t)
	ts.dropTen(t, c)
	v := recv(t, c).GetVitals()
	if v.GetHealth() != vitals.Max-7 || v.Hurt != pb.Harm_HARM_FALL {
		t.Fatalf("vitals = %v", v)
	}
}

func TestFallsNeverHurtInCreative(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	ts.dropTen(t, c)
	expectPong(t, c)
}

func TestFaintingSendsThePlayerBackToSpawnHealed(t *testing.T) {
	ts := startServer(t)
	c := ts.survivalAdmin(t)
	x, y, z := ts.spawn()
	for d := int32(1); d <= 25; d++ {
		if err := ts.world.SetBlock(0, int32(y)-d, 0, world.Air); err != nil {
			t.Fatal(err)
		}
	}
	send(t, c, move(x, y-0.5, z, 0))
	send(t, c, move(x, y-25, z, 0))
	if v := recv(t, c).GetVitals(); v.GetHealth() != 0 || v.Hurt != pb.Harm_HARM_FALL {
		t.Fatalf("vitals = %v", v)
	}
	sx, sy, sz := ts.spawn()
	if tp := recv(t, c).GetTeleport(); tp.GetId() != 1 || tp.Pos.X != sx || tp.Pos.Y != sy || tp.Pos.Z != sz {
		t.Fatalf("teleport = %v", tp)
	}
	if v := recv(t, c).GetVitals(); v.GetHealth() != vitals.Max || v.Hurt != pb.Harm_HARM_UNSPECIFIED {
		t.Fatalf("vitals = %v", v)
	}
}

func TestBackToCreativeHealsAtOnce(t *testing.T) {
	ts := startServer(t)
	c := ts.survivalAdmin(t)
	ts.dropTen(t, c)
	recvWhere(t, c, vitalsWith(vitals.Max-7))
	send(t, c, setGameMode(1, pb.GameMode_GAME_MODE_CREATIVE))
	if v := recvWhere(t, c, vitalsWith(vitals.Max)).GetVitals(); v.Mode != pb.GameMode_GAME_MODE_CREATIVE {
		t.Fatalf("vitals = %v", v)
	}
}

func TestAdminsSetModesByChat(t *testing.T) {
	ts := startServer(t)
	admin, kid := ts.admitKid(t)
	send(t, admin, sendChat("/survival kid"))
	recvWhere(t, admin, noticeOf("Kid plays survival now"))
	if v := recvWhere(t, kid, func(m *pb.ServerMessage) bool { return m.GetVitals() != nil }).GetVitals(); v.Mode != pb.GameMode_GAME_MODE_SURVIVAL {
		t.Fatalf("kid's vitals = %v", v)
	}
	if p := playerNamed(t, admin, "Kid"); p.Mode != pb.GameMode_GAME_MODE_SURVIVAL {
		t.Fatalf("kid = %v", p)
	}
}

func TestOnlyAdminsSetModes(t *testing.T) {
	ts := startServer(t)
	_, kid := ts.admitKid(t)
	send(t, kid, setGameMode(1, pb.GameMode_GAME_MODE_SURVIVAL))
	recvWhere(t, kid, func(m *pb.ServerMessage) bool {
		return m.GetDisconnect().GetCode() == pb.DisconnectCode_DISCONNECT_CODE_NOT_ADMIN
	})
}
