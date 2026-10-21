package scripts

import (
	"slices"
	"testing"
)

func TestCharactersStandOnTheirScriptBlocks(t *testing.T) {
	e := newEnv(t)
	e.save(t, here, `npc("Ivy")`)
	got := e.svc.NPCs()
	if len(got) != 1 || got[0].Name != "Ivy" || got[0].At != here {
		t.Fatalf("npcs %+v", got)
	}
	if n, ok := e.svc.NPCByID(got[0].ID); !ok || n != got[0] {
		t.Fatalf("by id %+v %v", n, ok)
	}
	if !slices.Equal(e.npcs, []string{"+Ivy"}) {
		t.Fatalf("events %q", e.npcs)
	}
}

func TestCharactersOutliveRestarts(t *testing.T) {
	e := newEnv(t)
	e.save(t, here, `npc("Ivy")`)
	e.start(t)
	if got := e.svc.NPCs(); len(got) != 1 || got[0].Name != "Ivy" {
		t.Fatalf("npcs after a restart %+v", got)
	}
}

func TestSavingAgainReplacesTheCharacter(t *testing.T) {
	e := newEnv(t)
	e.save(t, here, `npc("Ivy")`)
	e.save(t, here, `npc("Ivy")`)
	e.save(t, here, `npc("Bo")`)
	e.save(t, here, `say("no one here")`)
	if !slices.Equal(e.npcs, []string{"+Ivy", "-Ivy", "+Ivy", "-Ivy", "+Bo", "-Bo"}) {
		t.Fatalf("events %q", e.npcs)
	}
	e.start(t)
	if got := e.svc.NPCs(); len(got) != 0 {
		t.Fatalf("npcs %+v", got)
	}
}

func TestRenamingKeepsTheCharacter(t *testing.T) {
	e := newEnv(t)
	e.save(t, here, `npc("Ivy") function on_touch() npc("Granny Ivy") end`)
	id := e.svc.NPCs()[0].ID
	e.svc.Touch(here, "bob", t0)
	if got := e.svc.NPCs(); got[0].ID != id || got[0].Name != "Granny Ivy" {
		t.Fatalf("npcs %+v", got)
	}
	e.start(t)
	if got := e.svc.NPCs(); got[0].Name != "Granny Ivy" {
		t.Fatalf("the new name should be saved, got %+v", got)
	}
}

func TestDeletingTheScriptTakesItsCharacter(t *testing.T) {
	e := newEnv(t)
	e.save(t, here, `npc("Ivy")`)
	if err := e.svc.Delete(here); err != nil {
		t.Fatal(err)
	}
	if len(e.svc.NPCs()) != 0 || !slices.Equal(e.npcs, []string{"+Ivy", "-Ivy"}) {
		t.Fatalf("npcs %+v, events %q", e.svc.NPCs(), e.npcs)
	}
}

func TestTalkingAndAnswering(t *testing.T) {
	e := newEnv(t)
	e.save(t, here, `npc("Ivy") function on_talk(p)
		ask(p, "Help?", {"Yes", "No"}, function(a) say(a) end)
	end`)
	if f := e.svc.Talk(here, "bob", t0); len(f) != 0 {
		t.Fatal(f)
	}
	if f := e.svc.Answer(here, "bob", "Yes", t0); len(f) != 0 {
		t.Fatal(f)
	}
	if !slices.Equal(e.said, []string{"bob: Help? Yes/No", "Yes"}) {
		t.Fatalf("said %q", e.said)
	}
}

func TestWipeForgetsEveryScriptAndCharacter(t *testing.T) {
	e := newEnv(t)
	e.save(t, here, `npc("Ivy")`)
	if err := e.svc.Wipe(); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := e.svc.Source(here); ok || len(e.svc.NPCs()) != 0 {
		t.Fatal("a script or character outlived the wipe")
	}
	e.start(t)
	if len(e.svc.NPCs()) != 0 {
		t.Fatal("a character came back after a restart")
	}
}
