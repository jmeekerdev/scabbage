// Package crafting turns blocks a player carries into other blocks, following a recipe book.
package crafting

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"

	"scabbage/internal/inventory"
	"scabbage/internal/world"
)

//go:embed recipes.json
var defaultRecipes []byte

var ErrBadRecipe = errors.New("crafting: bad recipe")

// Recipe makes one stack from a handful of others, in any arrangement.
type Recipe struct {
	Make inventory.Stack
	From []inventory.Stack // in block order
}

// Default is the built-in recipe book.
func Default() []Recipe {
	book, err := Parse(defaultRecipes)
	if err != nil {
		panic(err) // the embedded file is checked by tests
	}
	return book
}

// Load reads the recipe book at path, or the default one when there is no file.
func Load(path string) ([]Recipe, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return nil, err
	}
	return Parse(data)
}

type recipeJSON struct {
	Make  string         `json:"make"`
	Count int            `json:"count"`
	From  map[string]int `json:"from"`
}

// Parse reads a recipe book: a JSON list of {"make": name, "count": n, "from": {name: n}}.
func Parse(data []byte) ([]Recipe, error) {
	var raw []recipeJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	book := make([]Recipe, len(raw))
	for i, r := range raw {
		var err error
		if book[i], err = r.recipe(); err != nil {
			return nil, fmt.Errorf("recipe %d: %w", i, err)
		}
	}
	return book, nil
}

func (r recipeJSON) recipe() (Recipe, error) {
	out, err := stack(r.Make, r.Count)
	if err != nil || len(r.From) == 0 {
		return Recipe{}, errors.Join(ErrBadRecipe, err)
	}
	rec := Recipe{Make: out}
	for name, n := range r.From {
		in, err := stack(name, n)
		if err != nil {
			return Recipe{}, err
		}
		rec.From = append(rec.From, in)
	}
	slices.SortFunc(rec.From, func(a, b inventory.Stack) int { return int(a.Block) - int(b.Block) })
	return rec, nil
}

func stack(name string, n int) (inventory.Stack, error) {
	b, ok := world.Builtin(name)
	if !ok || b == world.Air {
		return inventory.Stack{}, fmt.Errorf("%w: no block %q", ErrBadRecipe, name)
	}
	if n < 1 || n > inventory.StackMax {
		return inventory.Stack{}, fmt.Errorf("%w: %d %s", ErrBadRecipe, n, name)
	}
	return inventory.Stack{Block: b, Count: uint16(n)}, nil
}

// Craft trades the recipe's inputs for its output, or leaves inv alone when it can't.
func (r Recipe) Craft(inv *inventory.Inventory) bool {
	next := *inv
	for _, in := range r.From {
		if !next.Remove(in.Block, int(in.Count)) {
			return false
		}
	}
	if next.Add(r.Make.Block, int(r.Make.Count)) > 0 {
		return false
	}
	*inv = next
	return true
}
