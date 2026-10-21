package gateway

import (
	"slices"
	"testing"

	"github.com/coder/websocket"

	pb "scabbage/internal/pb/scabbage/v1"
)

func talkTo(npc uint32) *pb.ClientMessage {
	return &pb.ClientMessage{Body: &pb.ClientMessage_TalkTo{TalkTo: &pb.TalkTo{Npc: npc}}}
}

func answer(dialogue, choice uint32) *pb.ClientMessage {
	return &pb.ClientMessage{Body: &pb.ClientMessage_Answer{Answer: &pb.Answer{Dialogue: dialogue, Choice: choice}}}
}

func npcsChanged(t *testing.T, c *websocket.Conn) *pb.NpcsChanged {
	t.Helper()
	return waitFor(t, c, func(m *pb.ServerMessage) bool { return m.GetNpcsChanged() != nil }).GetNpcsChanged()
}

func dialogue(t *testing.T, c *websocket.Conn) *pb.Dialogue {
	t.Helper()
	return waitFor(t, c, func(m *pb.ServerMessage) bool { return m.GetDialogue() != nil }).GetDialogue()
}

func questLog(t *testing.T, c *websocket.Conn) []*pb.Quest {
	t.Helper()
	return waitFor(t, c, func(m *pb.ServerMessage) bool { return m.GetQuestLog() != nil }).GetQuestLog().Quests
}

func TestCharactersStandOnScriptBlocksForEveryone(t *testing.T) {
	ts := startServer(t)
	admin, kid := ts.joinKid(t)
	x, y, z := ts.scriptBlock(t, admin, `npc("Ivy")`)
	for _, c := range []*websocket.Conn{admin, kid} {
		got := npcsChanged(t, c).Npcs
		if len(got) != 1 || got[0].Name != "Ivy" || got[0].Pos.X != x || got[0].Pos.Y != y+1 || got[0].Pos.Z != z {
			t.Fatalf("npcs %v", got)
		}
	}
	send(t, admin, saveScript(x, y, z, `x = 1`))
	if gone := npcsChanged(t, kid).Gone; len(gone) != 1 {
		t.Fatalf("gone %v", gone)
	}
}

func TestJoiningPlayersMeetTheCharacters(t *testing.T) {
	ts := startServer(t)
	admin := ts.joinAdmin(t)
	ts.scriptBlock(t, admin, `npc("Ivy")`)
	_ = admin.Close(websocket.StatusNormalClosure, "")
	waitPlayers(t, ts.hub, 0)
	admin = dial(t, ts.url)
	send(t, admin, hello(secret(1), nil))
	if got := npcsChanged(t, admin).Npcs; len(got) != 1 || got[0].Name != "Ivy" {
		t.Fatalf("npcs %v", got)
	}
}

func TestTalkingToACharacterAsksAndHearsTheAnswer(t *testing.T) {
	ts := startServer(t)
	admin := ts.joinAdmin(t)
	ts.scriptBlock(t, admin, `npc("Ivy")
		function on_talk(p) ask(p, "Will you help?", {"Yes", "No"}, function(a) say(p .. " picked " .. a) end) end`)
	ivy := npcsChanged(t, admin).Npcs[0]

	send(t, admin, talkTo(ivy.Id))
	d := dialogue(t, admin)
	if d.Speaker != "Ivy" || d.Text != "Will you help?" || !slices.Equal(d.Choices, []string{"Yes", "No"}) {
		t.Fatalf("dialogue %v", d)
	}
	send(t, admin, answer(d.Id+1, 0)) // a question never asked
	send(t, admin, answer(d.Id, 9))   // a choice never offered
	send(t, admin, answer(d.Id, 1))
	scriptSaid(t, admin, "Mom picked No")
}

func TestQuestsAndMemoriesOutlastLeaving(t *testing.T) {
	ts := startServer(t)
	admin := ts.joinAdmin(t)
	x, y, z := ts.scriptBlock(t, admin, `function on_touch(p)
		if recall(p, "met") then say("again") else remember(p, "met", true) say("first") end
		quest(p, "Lost sheep", "Find 3 wool")
	end`)
	send(t, admin, useBlock(x, y, z))
	scriptSaid(t, admin, "first")
	if got := questLog(t, admin); len(got) != 1 || got[0].Title != "Lost sheep" || got[0].Step != "Find 3 wool" {
		t.Fatalf("quests %v", got)
	}

	_ = admin.Close(websocket.StatusNormalClosure, "")
	waitPlayers(t, ts.hub, 0)
	admin = dial(t, ts.url)
	send(t, admin, hello(secret(1), nil))
	if got := recv(t, admin).GetWelcome().GetQuests().GetQuests(); len(got) != 1 {
		t.Fatalf("quests after rejoining %v", got)
	}
	send(t, admin, useBlock(x, y, z))
	scriptSaid(t, admin, "again")
}

func TestCreativePlayersHaveAnyItemsAQuestWants(t *testing.T) {
	ts := startServer(t)
	admin := ts.joinAdmin(t)
	x, y, z := ts.scriptBlock(t, admin, `function on_touch(p) say(tostring(take_item(p, "white_wool", 3))) end`)
	send(t, admin, useBlock(x, y, z))
	scriptSaid(t, admin, "true")
}
