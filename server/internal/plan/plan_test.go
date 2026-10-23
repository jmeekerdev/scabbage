package plan

import (
	"strings"
	"testing"
)

func isBlock(name string) bool { return name == "planks" || name == "glass" || name == "stone" }

func hut() Plan {
	return Plan{
		Name:    "Hut",
		About:   "a tiny hut",
		Palette: map[string]string{"p": "Planks", "g": "glass", "m": "stone"},
		Layers: [][]string{
			{"ppp", "ppp"},
			{"p_p", "pgp"},
			{"mmm", "m.m"},
		},
	}
}

func TestCheckAcceptsAGoodPlanAndTidiesIt(t *testing.T) {
	p := hut()
	if err := p.Check(isBlock); err != nil {
		t.Fatal(err)
	}
	if p.Name != "hut" || p.Palette["p"] != "planks" {
		t.Errorf("not lower-cased: %q %q", p.Name, p.Palette["p"])
	}
	if w, d, h := p.Size(); w != 3 || d != 2 || h != 3 {
		t.Errorf("size %d %d %d", w, d, h)
	}
	if n := p.Cells(); n != 17 {
		t.Errorf("cells %d, want 17 (every cell but the one '.')", n)
	}
}

func TestCheckRefusesBadPlans(t *testing.T) {
	cases := map[string]struct {
		edit func(*Plan)
		want string
	}{
		"short name":    {func(p *Plan) { p.Name = "h" }, "names are 2-24"},
		"spaced name":   {func(p *Plan) { p.Name = "my hut" }, "names are 2-24"},
		"template name": {func(p *Plan) { p.Name = "Cabin" }, "built-in template"},
		"long about":    {func(p *Plan) { p.About = strings.Repeat("a", 201) }, "about"},
		"no layers":     {func(p *Plan) { p.Layers = nil }, "at least one layer"},
		"short row":     {func(p *Plan) { p.Layers[1][0] = "pp" }, "layer 2 row 1 is 2 long, not 3"},
		"missing row":   {func(p *Plan) { p.Layers[2] = p.Layers[2][:1] }, "layer 3 has 1 rows, not 2"},
		"unknown key":   {func(p *Plan) { p.Layers[0][1] = "pxp" }, "layer 1 row 2 uses x"},
		"long key":      {func(p *Plan) { p.Palette["pp"] = "stone" }, "one character"},
		"reserved key":  {func(p *Plan) { p.Palette["."] = "stone" }, "already mean"},
		"unknown block": {func(p *Plan) { p.Palette["g"] = "lava" }, "no block named lava"},
		"all dots":      {func(p *Plan) { p.Layers = [][]string{{"."}} }, "no cells"},
		"too tall": {func(p *Plan) {
			for len(p.Layers) <= MaxSide {
				p.Layers = append(p.Layers, p.Layers[0])
			}
		}, "at most 32 tall"},
		"too wide": {func(p *Plan) {
			row := strings.Repeat("p", MaxSide+1)
			p.Layers = [][]string{{row}}
		}, "at most 32 wide"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			p := hut()
			c.edit(&p)
			err := p.Check(isBlock)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Errorf("got %v, want %q", err, c.want)
			}
		})
	}
}

func TestCheckListsSeveralProblemsAtOnce(t *testing.T) {
	p := hut()
	p.Layers[0][1] = "pp"
	p.Layers[1][1] = "pzp"
	err := p.Check(isBlock)
	if err == nil || !strings.Contains(err.Error(), "layer 1 row 2") || !strings.Contains(err.Error(), "layer 2 row 2") {
		t.Errorf("got %v", err)
	}
}

func TestCheckNameAllowsLettersDigitsDashesAndUnderscores(t *testing.T) {
	for _, ok := range []string{"hut", "big-hut_2", "a1"} {
		if err := CheckName(ok); err != nil {
			t.Errorf("%s: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "2hut", "hut!", strings.Repeat("a", 25), "tower"} {
		if CheckName(bad) == nil {
			t.Errorf("%q passed", bad)
		}
	}
}
