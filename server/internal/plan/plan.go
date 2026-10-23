// Package plan checks and stores plans: building designs players and agents save for /build.
package plan

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"
)

// MaxSide is the most blocks a plan may be on any side, as for templates.
const MaxSide = 32

const (
	maxAbout    = 200
	maxProblems = 5
	keep        = '.' // leaves the cell as it is
	air         = '_' // clears the cell
)

// Templates are the client's built-in templates (client/src/autopilot/templates); plans can't take their names.
var Templates = []string{"cabin", "tower", "wall", "floor", "stairs"}

// Plan is a design: palette keys in layers (bottom first) of rows (front first). x runs across the
// front, y up and z back from the front, as for templates.
type Plan struct {
	Name    string            `json:"name"`
	About   string            `json:"about,omitempty"`
	Palette map[string]string `json:"palette"`
	Layers  [][]string        `json:"layers"`
}

// Size is the plan's width, depth and height, taken from its first row and layer.
func (p Plan) Size() (w, d, h int) {
	h = len(p.Layers)
	if h > 0 {
		d = len(p.Layers[0])
	}
	if d > 0 {
		w = utf8.RuneCountInString(p.Layers[0][0])
	}
	return w, d, h
}

// Cells counts the cells the plan changes: all but those marked '.'.
func (p Plan) Cells() int {
	n := 0
	for _, layer := range p.Layers {
		for _, row := range layer {
			n += utf8.RuneCountInString(row) - strings.Count(row, string(keep))
		}
	}
	return n
}

// CheckName says why a name can't be a plan's, or nil if it can.
func CheckName(name string) error {
	if slices.Contains(Templates, name) {
		return fmt.Errorf("%s is a built-in template; pick another name", name)
	}
	ok := len(name) >= 2 && len(name) <= 24 && name[0] >= 'a' && name[0] <= 'z'
	for _, c := range name {
		ok = ok && (c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' || c == '-')
	}
	if !ok {
		return errors.New("plan names are 2-24 lowercase letters, digits, _ or -, starting with a letter")
	}
	return nil
}

// Check lower-cases the name and block names, and says everything wrong with the plan, or nil.
func (p *Plan) Check(isBlock func(name string) bool) error {
	p.Name = strings.ToLower(strings.TrimSpace(p.Name))
	var probs problems
	probs.add(CheckName(p.Name))
	if utf8.RuneCountInString(p.About) > maxAbout {
		probs.addf("about can be at most %d characters", maxAbout)
	}
	p.checkPalette(isBlock, &probs)
	p.checkLayers(&probs)
	if len(probs) == 0 && p.Cells() == 0 {
		probs.addf("the plan has no cells to build; '.' leaves a cell alone")
	}
	return probs.err()
}

func (p *Plan) checkPalette(isBlock func(string) bool, probs *problems) {
	keys := make([]string, 0, len(p.Palette))
	for k := range p.Palette {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := strings.ToLower(strings.TrimSpace(p.Palette[k]))
		p.Palette[k] = v
		switch {
		case utf8.RuneCountInString(k) != 1:
			probs.addf("palette keys are one character, not %q", k)
		case k == string(keep) || k == string(air):
			probs.addf("%q and %q already mean leave alone and air", keep, air)
		case v != "air" && !isBlock(v):
			probs.addf("there's no block named %s", v)
		}
	}
}

func (p *Plan) checkLayers(probs *problems) {
	w, d, h := p.Size()
	for _, side := range []struct {
		n    int
		word string
	}{{h, "tall"}, {d, "deep"}, {w, "wide"}} {
		if side.n == 0 {
			probs.addf("a plan needs at least one layer, row and cell")
			return
		}
		if side.n > MaxSide {
			probs.addf("a plan can be at most %d %s", MaxSide, side.word)
			return
		}
	}
	for i, layer := range p.Layers {
		if len(layer) != d {
			probs.addf("layer %d has %d rows, not %d", i+1, len(layer), d)
			continue
		}
		for j, row := range layer {
			p.checkRow(row, w, fmt.Sprintf("layer %d row %d", i+1, j+1), probs)
		}
	}
}

func (p *Plan) checkRow(row string, w int, where string, probs *problems) {
	if n := utf8.RuneCountInString(row); n != w {
		probs.addf("%s is %d long, not %d", where, n, w)
		return
	}
	for _, c := range row {
		if _, ok := p.Palette[string(c)]; !ok && c != keep && c != air {
			probs.addf("%s uses %c, which isn't in the palette", where, c)
			return
		}
	}
}

// problems gathers what's wrong with a plan, keeping the first few.
type problems []string

func (ps *problems) add(err error) {
	if err != nil {
		ps.addf("%s", err)
	}
}

func (ps *problems) addf(format string, args ...any) {
	if len(*ps) < maxProblems {
		*ps = append(*ps, fmt.Sprintf(format, args...))
	}
}

func (ps problems) err() error {
	if len(ps) == 0 {
		return nil
	}
	return errors.New(strings.Join(ps, "; "))
}
