package gateway

import (
	"testing"

	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/world"
)

func changedTo(b world.Block) func(*pb.ServerMessage) bool {
	return func(m *pb.ServerMessage) bool { return m.GetBlockChanged().GetBlock() == uint32(b) }
}

func TestOpeningAChestGivesLootOnce(t *testing.T) {
	ts := startServer(t)
	c := ts.survivalAdmin(t)
	x, y, z := ts.besideSpawn(t, 0, world.Chest)
	send(t, c, useBlock(x, y, z))
	recvWhere(t, c, changedTo(world.Chest+1))
	inv := recvWhere(t, c, isInventory).GetInventory()
	if inv.Slots[0].Block != uint32(world.GoldBlock) || inv.Slots[0].Count == 0 {
		t.Fatalf("slot 0 = %v", inv.Slots[0])
	}
	recvWhere(t, c, noticeOf("You found gold block"))
	if b, _ := ts.world.Block(x, y, z); b != world.Chest+1 {
		t.Fatalf("chest is %v", b)
	}
	send(t, c, useBlock(x, y, z))
	if got := recvWhere(t, c, func(m *pb.ServerMessage) bool { return m.GetBlockChanged() != nil }); got.GetBlockChanged().Block != uint32(world.Chest+1) {
		t.Fatalf("an open chest became %v", got)
	}
}

func TestAltarsWakeOneBossAtATime(t *testing.T) {
	ts := startServer(t)
	c := ts.joinAdmin(t)
	x, y, z := ts.besideSpawn(t, 0, world.BossAltar)
	send(t, c, useBlock(x, y, z))
	boss := func(m *pb.ServerMessage) bool {
		ms := m.GetMobsChanged().GetMobs()
		return len(ms) == 1 && ms[0].MaxHealth > 0 && ms[0].Health == ms[0].MaxHealth
	}
	got := recvWhere(t, c, boss).GetMobsChanged().Mobs[0]
	if got.Kind != pb.MobKind_MOB_KIND_KING_SLIME && got.Kind != pb.MobKind_MOB_KIND_GOLEM {
		t.Fatalf("woke %v", got)
	}
	recvWhere(t, c, noticeOf("wakes"))
	send(t, c, useBlock(x, y, z))
	recvWhere(t, c, noticeOf("already awake"))
}
