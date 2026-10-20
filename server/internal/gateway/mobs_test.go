package gateway

import (
	"testing"
	"time"

	"scabbage/internal/mobs"
	"scabbage/internal/motion"
	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/world"
)

func hitMob(id uint32) *pb.ClientMessage {
	return &pb.ClientMessage{Body: &pb.ClientMessage_HitMob{HitMob: &pb.HitMob{Mob: id}}}
}

func mobsWith(id uint32) func(*pb.ServerMessage) bool {
	return func(m *pb.ServerMessage) bool {
		for _, mob := range m.GetMobsChanged().GetMobs() {
			if mob.Id == id {
				return true
			}
		}
		return false
	}
}

// sheepBesideSpawn puts a sheep on stone right next to the spawn point.
func (ts *testServer) sheepBesideSpawn(t *testing.T) mobs.Mob {
	t.Helper()
	x, y, z := ts.besideSpawn(t, 0, world.Air)
	return ts.herd.Add(mobs.Sheep, motion.Vec{X: float64(x) + 0.5, Y: float64(y), Z: float64(z) + 0.5}, time.Now())
}

func TestJoinersSeeTheMobs(t *testing.T) {
	ts := startServer(t)
	sheep := ts.sheepBesideSpawn(t)
	c := ts.joinAdmin(t)
	got := recvWhere(t, c, mobsWith(sheep.ID)).GetMobsChanged().Mobs[0]
	if got.Kind != pb.MobKind_MOB_KIND_SHEEP || got.Pos == nil {
		t.Fatalf("mob = %v", got)
	}
}

func TestShearingASheepGivesWool(t *testing.T) {
	ts := startServer(t)
	sheep := ts.sheepBesideSpawn(t)
	c := ts.survivalAdmin(t)
	send(t, c, hitMob(sheep.ID))
	hit := func(m *pb.ServerMessage) bool { return mobsWith(sheep.ID)(m) && m.GetMobsChanged().Mobs[0].Hurt }
	if got := recvWhere(t, c, hit).GetMobsChanged().Mobs[0]; !got.Sheared {
		t.Fatalf("hit sheep = %v", got)
	}
	if s := recvWhere(t, c, isInventory).GetInventory().Slots[0]; s.Block != uint32(world.WhiteWool) || s.Count != 1 {
		t.Fatalf("slot 0 = %v", s)
	}
}

func TestConsoleCannotHitMobs(t *testing.T) {
	ts := startServer(t)
	c := ts.joinConsole(t)
	send(t, c, hitMob(1))
	expectClosed(t, c)
}
