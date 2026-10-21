package journal

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"scabbage/internal/store"
)

const amy = 1

func TestRememberAndForget(t *testing.T) {
	var j Journal
	if err := j.Remember("sheep", "n3"); err != nil {
		t.Fatal(err)
	}
	if v, ok := j.Recall("sheep"); !ok || v != "n3" {
		t.Fatalf("recall = %q, %v", v, ok)
	}
	if err := j.Remember("sheep", ""); err != nil {
		t.Fatal(err)
	}
	if _, ok := j.Recall("sheep"); ok {
		t.Fatal("an empty value should forget")
	}
}

func TestMemoryFills(t *testing.T) {
	var j Journal
	for i := range MaxMemories {
		if err := j.Remember(fmt.Sprint(i), "x"); err != nil {
			t.Fatal(err)
		}
	}
	if err := j.Remember("one more", "x"); !errors.Is(err, ErrFull) {
		t.Fatalf("err = %v, want ErrFull", err)
	}
	if err := j.Remember("0", "changed"); err != nil {
		t.Fatalf("changing a memory should still work: %v", err)
	}
}

func TestQuestsNewestFirst(t *testing.T) {
	var j Journal
	mustChange(t)(j.SetQuest("Lost sheep", "Find 3 wool"))
	mustChange(t)(j.SetQuest("Old well", "Look down the well"))
	mustChange(t)(j.SetQuest("Lost sheep", "Bring the wool to Ivy"))
	want := []Quest{{Title: "Lost sheep", Step: "Bring the wool to Ivy"}, {Title: "Old well", Step: "Look down the well"}}
	if got := j.Quests(); !reflect.DeepEqual(got, want) {
		t.Fatalf("quests = %+v", got)
	}
	if changed, _ := j.SetQuest("Lost sheep", "Bring the wool to Ivy"); changed {
		t.Fatal("the same step again is no change")
	}
}

func TestFinishingAQuest(t *testing.T) {
	var j Journal
	mustChange(t)(j.SetQuest("Lost sheep", "Find 3 wool"))
	mustChange(t)(j.Finish("Lost sheep"))
	if got := j.Quests(); !reflect.DeepEqual(got, []Quest{{Title: "Lost sheep", Done: true}}) {
		t.Fatalf("quests = %+v", got)
	}
	if changed, _ := j.Finish("Lost sheep"); changed {
		t.Fatal("finishing twice is no change")
	}
}

func TestFullLogDropsOldestFinished(t *testing.T) {
	var j Journal
	mustChange(t)(j.Finish("first"))
	for i := 1; i < MaxQuests; i++ {
		mustChange(t)(j.SetQuest(fmt.Sprint(i), "go"))
	}
	mustChange(t)(j.SetQuest("new", "go"))
	if len(j.Quests()) != MaxQuests || j.Quests()[MaxQuests-1].Title == "first" {
		t.Fatal("the finished quest should make room")
	}
	if _, err := j.SetQuest("another", "go"); !errors.Is(err, ErrFull) {
		t.Fatalf("err = %v, want ErrFull", err)
	}
}

func TestSavesAndLoads(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`INSERT INTO players (id, name, color, created_ms) VALUES (?, 'Amy', 0, 0)`, amy); err != nil {
		t.Fatal(err)
	}
	j, err := Load(db, amy)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := j.Unsaved(); ok {
		t.Fatal("a fresh journal has nothing to save")
	}
	_ = j.Remember("sheep", "n3")
	mustChange(t)(j.SetQuest("Old well", "Look"))
	mustChange(t)(j.SetQuest("Lost sheep", "Find wool"))
	saved, ok := j.Unsaved()
	if !ok {
		t.Fatal("changes should need saving")
	}
	if _, again := j.Unsaved(); again {
		t.Fatal("handing out a save clears the change")
	}
	if err := Store(db, amy, saved); err != nil {
		t.Fatal(err)
	}
	back, err := Load(db, amy)
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := back.Recall("sheep"); v != "n3" || !reflect.DeepEqual(back.Quests(), j.Quests()) {
		t.Fatalf("loaded %q and %+v", v, back.Quests())
	}
}

// mustChange fails the test unless the change it is handed happened without error.
func mustChange(t *testing.T) func(bool, error) {
	return func(changed bool, err error) {
		t.Helper()
		if err != nil || !changed {
			t.Fatalf("changed = %v, err = %v", changed, err)
		}
	}
}
