package world

import "math"

// tree is a trunk standing on the ground with a crown of leaves.
type tree struct {
	kind        featureKind
	x, z        int32 // the trunk's low corner
	width       int32 // the trunk's width in blocks
	ground, top int32 // top is the y of the highest log
	r           uint64
}

// tree sizes the feature's tree: oaks are broad, pines tall and pointed, slim trees thin.
func (f feature) tree() tree {
	t := tree{kind: f.kind, x: f.x, z: f.z, ground: f.ground, r: f.r >> 8}
	switch f.kind {
	case oakTree:
		t.width, t.top = Metre, f.ground+(4+int32(f.r%3))*Metre
	case pineTree:
		t.width, t.top = 2, f.ground+(7+int32(f.r%3))*Metre
	default:
		t.width, t.top = 2, f.ground+(5+int32(f.r%3))*Metre
	}
	return t
}

// trunk is logs from a metre underground, so slopes don't leave it floating, up to the top.
func (t tree) trunk(put putFunc) {
	fillBox(t.x, t.ground-Metre+1, t.z, t.x+t.width-1, t.top, t.z+t.width-1, func(x, y, z int32) { put(x, y, z, Log, false) })
}

// crown draws the leaves into air and plants.
func (t tree) crown(cv canvas) {
	cx, cz := float64(t.x)+float64(t.width)/2, float64(t.z)+float64(t.width)/2
	top := float64(t.top) + 1
	leaf := func(b Block) func(x, y, z int32) { return func(x, y, z int32) { cv.put(x, y, z, b, true) } }
	switch t.kind {
	case oakTree:
		r := 6.5 + float64(t.r%3)*0.75
		blob(cv, cx, top-2, cz, r, r*0.75, r, t.r, leaf(Leaves))
	case pineTree:
		t.cone(cv, cx, cz, leaf(PineLeaves))
	default:
		r := 4.0 + float64(t.r%2)
		blob(cv, cx, top-3, cz, r, r*1.6, r, t.r, leaf(Leaves))
	}
}

// cone draws a pine's tiers of branches, widest at the bottom, from two metres up to a point
// above the top log.
func (t tree) cone(cv canvas, cx, cz float64, put func(x, y, z int32)) {
	base, apex := cv.clamp(1, t.ground+2*Metre, t.top+3)
	for y := base; y <= apex; y++ {
		r := float64(t.top+3-y)*0.32 + 1.6 - float64((t.top+3-y)%5)*0.25
		reach := int32(math.Ceil(r))
		x0, x1 := cv.clamp(0, int32(math.Floor(cx))-reach, int32(math.Ceil(cx))+reach)
		z0, z1 := cv.clamp(2, int32(math.Floor(cz))-reach, int32(math.Ceil(cz))+reach)
		for x := x0; x <= x1; x++ {
			for z := z0; z <= z1; z++ {
				dx, dz := float64(x)+0.5-cx, float64(z)+0.5-cz
				rough := float64(hash3(t.r, x, y, z)%64) / 128
				if float64(dx*dx)+float64(dz*dz) <= float64(r*r)*(1-rough*0.5) {
					put(x, y, z)
				}
			}
		}
	}
}
