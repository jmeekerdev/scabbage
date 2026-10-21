package crafting

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"scabbage/internal/inventory"
	"scabbage/internal/world"
)

func TestDefaultRecipesAllParse(t *testing.T) {
	book := Default()
	if len(book) == 0 {
		t.Fatal("no recipes")
	}
	planks := book[0]
	if planks.Make != (inventory.Stack{Block: world.Planks, Count: 4}) || planks.From[0] != (inventory.Stack{Block: world.Log, Count: 1}) {
		t.Fatalf("first recipe = %+v", planks)
	}
}

func TestParseRejectsBadRecipes(t *testing.T) {
	for name, src := range map[string]string{
		"unknown block": `[{"make":"unobtainium","count":1,"from":{"log":1}}]`,
		"unknown input": `[{"make":"planks","count":1,"from":{"unobtainium":1}}]`,
		"no count":      `[{"make":"planks","from":{"log":1}}]`,
		"no inputs":     `[{"make":"planks","count":1,"from":{}}]`,
		"zero input":    `[{"make":"planks","count":1,"from":{"log":0}}]`,
		"too many":      `[{"make":"planks","count":1000,"from":{"log":1}}]`,
		"air":           `[{"make":"air","count":1,"from":{"log":1}}]`,
	} {
		if _, err := Parse([]byte(src)); !errors.Is(err, ErrBadRecipe) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}

func TestParseListsInputsInBlockOrder(t *testing.T) {
	book, err := Parse([]byte(`[{"make":"brick","count":2,"from":{"sand":1,"dirt":1}}]`))
	if err != nil {
		t.Fatal(err)
	}
	if f := book[0].From; len(f) != 2 || f[0].Block != world.Dirt || f[1].Block != world.Sand {
		t.Fatalf("from = %+v", f)
	}
}

func TestCraftTradesInputsForTheOutput(t *testing.T) {
	var inv inventory.Inventory
	inv[0] = inventory.Stack{Block: world.Log, Count: 2}
	if !Default()[0].Craft(&inv) {
		t.Fatal("could not make planks from a log")
	}
	if inv[0] != (inventory.Stack{Block: world.Log, Count: 1}) || inv[1] != (inventory.Stack{Block: world.Planks, Count: 4}) {
		t.Fatalf("inventory = %+v, %+v", inv[0], inv[1])
	}
}

func TestCraftNeedsEveryInput(t *testing.T) {
	var inv inventory.Inventory
	if Default()[0].Craft(&inv) || inv != (inventory.Inventory{}) {
		t.Fatal("made planks from nothing")
	}
}

func TestCraftNeedsRoomForTheOutput(t *testing.T) {
	var inv inventory.Inventory
	for i := range inv {
		inv[i] = inventory.Stack{Block: world.Stone, Count: inventory.StackMax}
	}
	inv[0] = inventory.Stack{Block: world.Log, Count: 2}
	before := inv
	if Default()[0].Craft(&inv) || inv != before {
		t.Fatal("crafted into a full inventory")
	}
}

func TestLoadFallsBackToTheDefaultBook(t *testing.T) {
	book, err := Load(filepath.Join(t.TempDir(), "recipes.json"))
	if err != nil || len(book) != len(Default()) {
		t.Fatalf("got %d recipes, %v", len(book), err)
	}
}

func TestLoadReadsTheServersOwnBook(t *testing.T) {
	path := filepath.Join(t.TempDir(), "recipes.json")
	if err := os.WriteFile(path, []byte(`[{"make":"glass","count":9,"from":{"dirt":1}}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	book, err := Load(path)
	if err != nil || len(book) != 1 || book[0].Make.Count != 9 {
		t.Fatalf("book = %+v, %v", book, err)
	}
}
