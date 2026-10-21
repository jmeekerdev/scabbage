package world

import "math"

// featureKind is something that stands on the ground: a tree, bush or stone.
type featureKind uint8

const (
	noFeature featureKind = iota
	oakTree
	pineTree
	slimTree
	bush
	boulder
	rock
)

func (k featureKind) String() string {
	return [...]string{"nothing", "oak", "pine", "slim tree", "bush", "boulder", "rock"}[k]
}

// featureOdds is how many of each kind grow per thousand square metres.
var featureOdds = [...]struct {
	kind     featureKind
	perMille uint64
}{{oakTree, 6}, {pineTree, 4}, {slimTree, 3}, {bush, 14}, {boulder, 3}, {rock, 25}}

const (
	featureReach = 3          // metres a feature spreads from its square
	featureRoom  = 14 * Metre // headroom a feature needs under the sky
)

// feature is one thing rooted in a square metre whose low corner is x,z, on ground of height ground.
type feature struct {
	kind         featureKind
	x, z, ground int32
	r            uint64 // random bits for its size and shape
}

// featureAt returns the feature rooted in the square metre mx,mz, if any.
func (g *GenV3) featureAt(mx, mz int32) (feature, bool) {
	r := hash(g.features, mx, mz)
	f := feature{kind: pickFeature(r % 1000), x: mx * Metre, z: mz * Metre, r: r >> 10}
	if f.kind == noFeature {
		return feature{}, false
	}
	f.ground = g.SurfaceY(f.x+Metre/2, f.z+Metre/2)
	floor := int32(seaLevel + beachRise)
	if f.kind == rock {
		floor = seaLevel
	}
	if f.ground <= floor || f.ground+featureRoom >= MaxY {
		return feature{}, false
	}
	return f, true
}

func pickFeature(roll uint64) featureKind {
	for _, o := range featureOdds {
		if roll < o.perMille {
			return o.kind
		}
		roll -= o.perMille
	}
	return noFeature
}

// placeFeatures draws every feature that touches chunk p: stone and trunks first, then foliage
// into the air around them.
func (g *GenV3) placeFeatures(p ChunkPos, cv canvas) {
	ox, oy, oz := p.Origin()
	lo, hi := floorDiv(ox, Metre)-featureReach, floorDiv(ox+Size-1, Metre)+featureReach
	loZ, hiZ := floorDiv(oz, Metre)-featureReach, floorDiv(oz+Size-1, Metre)+featureReach
	var fs []feature
	for mx := lo; mx <= hi; mx++ {
		for mz := loZ; mz <= hiZ; mz++ {
			if f, ok := g.featureAt(mx, mz); ok && f.ground-Metre < oy+Size && f.ground+featureRoom >= oy {
				fs = append(fs, f)
			}
		}
	}
	for _, f := range fs {
		f.solid(cv)
	}
	for _, f := range fs {
		f.foliage(cv)
	}
}

// solid draws the feature's stone or trunk, over whatever is there.
func (f feature) solid(cv canvas) {
	switch f.kind {
	case oakTree, pineTree, slimTree:
		f.tree().trunk(cv.put)
	case boulder:
		f.boulder(cv)
	case rock:
		f.rock(cv.put)
	}
}

// foliage draws the feature's leaves into air and plants.
func (f feature) foliage(cv canvas) {
	switch f.kind {
	case oakTree, pineTree, slimTree:
		f.tree().crown(cv)
	case bush:
		f.bush(cv)
	}
}

// middle is the centre of the feature's square metre.
func (f feature) middle() (float64, float64) {
	return float64(f.x) + Metre/2.0, float64(f.z) + Metre/2.0
}

// bush is a low mound of leaves.
func (f feature) bush(cv canvas) {
	cx, cz := f.middle()
	rx, ry := 3.5+float64(f.r%3)*0.75, 2.5+float64(f.r>>4%3)*0.5
	blob(cv, cx, float64(f.ground)+1.5, cz, rx, ry, rx, f.r, func(x, y, z int32) { cv.put(x, y, z, Leaves, true) })
}

// boulder is a half-buried lump of stone flecked with cobbles.
func (f feature) boulder(cv canvas) {
	cx, cz := f.middle()
	r := 3.0 + float64(f.r%5)
	blob(cv, cx, float64(f.ground)+0.5, cz, r, r*0.75, r*(0.7+float64(f.r>>4%4)*0.1), f.r, func(x, y, z int32) {
		b := Stone
		if hash3(f.r, x, y, z)%6 == 0 {
			b = Cobblestone
		}
		cv.put(x, y, z, b, false)
	})
}

// rock is a few stones lying on the ground, with one on top when they make a full square.
func (f feature) rock(put putFunc) {
	x, y, z := f.x+1, f.ground+1, f.z+1
	b := []Block{Stone, Cobblestone}[f.r%2]
	put(x, y, z, b, false)
	square := true
	for i, d := range [][2]int32{{1, 0}, {0, 1}, {1, 1}} {
		if f.r>>(2+i)&1 == 1 {
			put(x+d[0], y, z+d[1], b, false)
		} else {
			square = false
		}
	}
	if square {
		put(x, y+1, z, b, false)
	}
}

// blob calls put for the blocks on the canvas inside an ellipsoid around cx,cy,cz with radii
// rx,ry,rz, its edge roughened by seed.
func blob(cv canvas, cx, cy, cz, rx, ry, rz float64, seed uint64, put func(x, y, z int32)) {
	x0, x1 := cv.clamp(0, int32(math.Floor(cx-rx)), int32(math.Ceil(cx+rx)))
	y0, y1 := cv.clamp(1, int32(math.Floor(cy-ry)), int32(math.Ceil(cy+ry)))
	z0, z1 := cv.clamp(2, int32(math.Floor(cz-rz)), int32(math.Ceil(cz+rz)))
	for x := x0; x <= x1; x++ {
		for y := y0; y <= y1; y++ {
			for z := z0; z <= z1; z++ {
				dx, dy, dz := (float64(x)+0.5-cx)/rx, (float64(y)+0.5-cy)/ry, (float64(z)+0.5-cz)/rz
				d := float64(dx*dx) + float64(dy*dy) + float64(dz*dz)
				if d <= 0.75 || d <= 1 && d <= 1-float64(hash3(seed, x, y, z)%64)/256 {
					put(x, y, z)
				}
			}
		}
	}
}

// hash3 mixes a seed and a block position into well-distributed bits.
func hash3(seed uint64, x, y, z int32) uint64 {
	return hash(mix(seed^uint64(uint32(y))), x, z)
}
