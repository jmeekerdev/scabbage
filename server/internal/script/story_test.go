package script

import (
	"slices"
	"strings"
	"testing"
)

func TestScriptsNameTheirCharacter(t *testing.T) {
	_, h := load(t, `npc("Granny Ivy")`)
	if h.npc != "Granny Ivy" {
		t.Fatalf("npc = %q", h.npc)
	}
	if _, err := Load(`npc("  ")`, &fakeHost{}, t0); err == nil {
		t.Fatal("a blank name should be refused")
	}
}

func TestTalkingRunsOnTalk(t *testing.T) {
	s, h := load(t, `function on_talk(player) talk(player, "Hello, " .. player) end`)
	if err := s.Talk("ada", t0); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(h.talks, []string{`ada: Hello, ada []`}) {
		t.Fatalf("talks %q", h.talks)
	}
}

func TestAskWaitsForTheAnswer(t *testing.T) {
	s, h := load(t, `function on_talk(player)
		ask(player, "Will you help?", {"Yes", "No"}, function(answer) say(player .. " said " .. answer) end)
	end`)
	if err := s.Talk("ada", t0); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(h.talks, []string{`ada: Will you help? ["Yes" "No"]`}) {
		t.Fatalf("talks %q", h.talks)
	}
	for _, who := range []string{"bo", "Ada", "ada"} {
		if err := s.Answer(who, "Yes", t0); err != nil {
			t.Fatal(err)
		}
	}
	if !slices.Equal(h.said, []string{"ada said Yes"}) {
		t.Fatalf("said %q: only ada's first answer counts, whatever her name's case", h.said)
	}
}

func TestAskNeedsOneToFourChoices(t *testing.T) {
	for _, choices := range []string{`{}`, `{"a", "b", "c", "d", "e"}`, `"Yes"`} {
		s, _ := load(t, `function on_talk(p) ask(p, "?", `+choices+`, function() end) end`)
		if err := s.Talk("ada", t0); err == nil {
			t.Errorf("choices %s should be refused", choices)
		}
	}
}

func TestWaitingAnswersAreCapped(t *testing.T) {
	s, _ := load(t, `function on_touch(p) ask(p, "?", {"ok"}, function() end) end`)
	for i := range MaxAsks + 5 {
		if err := s.Touch(strings.Repeat("p", i+1), t0); err != nil {
			t.Fatal(err)
		}
	}
	if len(s.answers) > MaxAsks {
		t.Fatalf("%d answers waiting", len(s.answers))
	}
}

func TestRememberKeepsKinds(t *testing.T) {
	s, h := load(t, `function on_touch(p)
		remember(p, "sheep", 3) remember(p, "name", "3") remember(p, "met", true) remember(p, "gone", "x")
		remember(p, "gone", nil)
		say(type(recall(p, "sheep")) .. " " .. type(recall(p, "name")) .. " " ..
			tostring(recall(p, "met")) .. " " .. tostring(recall(p, "gone")) .. " " .. recall(p, "sheep") + 1)
	end`)
	if err := s.Touch("ada", t0); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(h.said, []string{"number string true nil 4"}) {
		t.Fatalf("said %q", h.said)
	}
	if _, ok := h.memory["ada gone"]; !ok || h.memory["ada gone"] != "" {
		t.Fatal("remembering nil should forget, by remembering nothing")
	}
}

func TestRememberRefusesTablesAndHugeValues(t *testing.T) {
	for _, value := range []string{`{}`, `string.format("%99s", "") .. string.format("%99s", "") .. "xxxx"`} {
		s, _ := load(t, `function on_touch(p) remember(p, "k", `+value+`) end`)
		if err := s.Touch("ada", t0); err == nil {
			t.Errorf("remembering %s should be refused", value)
		}
	}
}

func TestQuestsStartAndFinish(t *testing.T) {
	s, h := load(t, `function on_touch(p) quest(p, "Lost sheep", "Find 3 wool") finish_quest(p, "Old well") end`)
	if err := s.Touch("ada", t0); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(h.quests, []string{"ada Lost sheep: Find 3 wool", "ada Old well done"}) {
		t.Fatalf("quests %q", h.quests)
	}
}

func TestTakeItemSaysWhetherItWorked(t *testing.T) {
	s, h := load(t, `function on_touch(p) if take_item(p, "white_wool", 3) then say("thanks") else say("bring wool") end end`)
	for _, has := range []bool{false, true} {
		h.hasItems = has
		if err := s.Touch("ada", t0); err != nil {
			t.Fatal(err)
		}
	}
	if !slices.Equal(h.said, []string{"bring wool", "thanks"}) || h.given[0] != "ada white_wool -3" {
		t.Fatalf("said %q, given %q", h.said, h.given)
	}
}
