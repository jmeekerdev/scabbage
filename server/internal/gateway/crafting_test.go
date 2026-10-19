package gateway

import (
	"testing"

	"scabbage/internal/crafting"
	pb "scabbage/internal/pb/scabbage/v1"
	"scabbage/internal/world"
)

func craft(recipe uint32) *pb.ClientMessage {
	return &pb.ClientMessage{Body: &pb.ClientMessage_Craft{Craft: &pb.Craft{Recipe: recipe}}}
}

func TestWelcomeListsTheRecipes(t *testing.T) {
	ts := startServer(t)
	c := dial(t, ts.url)
	code, _, _ := ts.svc.AdminInvite()
	send(t, c, hello(secret(1), &pb.JoinRequest{Code: code, Name: "Mom", Color: 7}))
	got := recv(t, c).GetWelcome().World.Recipes
	if len(got) != len(crafting.Default()) || got[0].Make.Block != uint32(world.Planks) || got[0].From[0].Block != uint32(world.Log) {
		t.Fatalf("recipes = %v", got)
	}
}

func TestCraftingTradesALogForPlanks(t *testing.T) {
	ts := startServer(t)
	c := ts.survivalAdmin(t)
	ts.mine(t, c, world.Log, world.Log)
	send(t, c, craft(0))
	if s := recvWhere(t, c, isInventory).GetInventory().Slots[0]; s.Block != uint32(world.Planks) || s.Count != 4 {
		t.Fatalf("slot 0 = %v", s)
	}
}

func TestCraftingWithoutTheInputsDoesNothing(t *testing.T) {
	ts := startServer(t)
	c := ts.survivalAdmin(t)
	send(t, c, craft(0))
	send(t, c, craft(9999))
	expectPong(t, c)
}
