package world

import "math"

// FarCells is how many samples a far tile has along each side.
const FarCells = 8

const farStep = Size / FarCells // blocks between samples

// FarTile sketches one chunk column's surface for drawing far beyond the loaded chunks. Sample i
// is at FarSample(tx, tz, i), x varying fastest.
type FarTile struct {
	Tops   [FarCells * FarCells]int32 // the y just above the highest block
	Blocks [FarCells * FarCells]Block // that block
}

// FarSample is the block column sample i of far tile tx,tz stands for, the middle of its cell.
func FarSample(tx, tz int32, i int) (x, z int32) {
	return tx*Size + int32(i%FarCells)*farStep + farStep/2, tz*Size + int32(i/FarCells)*farStep + farStep/2
}

// FarTile sketches chunk column tx,tz as generated: the ground, the sea, and the crowns of
// trees and bushes, leaving out plants and stones.
func (g *GenV3) FarTile(tx, tz int32) FarTile {
	var t FarTile
	for i := range t.Tops {
		t.Tops[i], t.Blocks[i] = g.groundTop(FarSample(tx, tz, i))
	}
	g.addCrowns(tx, tz, &t)
	return t
}

// groundTop is the y above column x,z's ground or sea, and the block there.
func (g *GenV3) groundTop(x, z int32) (int32, Block) {
	c := g.column(x, z)
	if c.h < seaLevel {
		return seaLevel + 1, Water
	}
	return c.h + 1, c.top
}

// addCrowns raises the tile's samples under any tree or bush to the top of its leaves.
func (g *GenV3) addCrowns(tx, tz int32, t *FarTile) {
	lo, hi := floorDiv(tx*Size, Meter)-featureReach, floorDiv(tx*Size+Size-1, Meter)+featureReach
	loZ, hiZ := floorDiv(tz*Size, Meter)-featureReach, floorDiv(tz*Size+Size-1, Meter)+featureReach
	for mx := lo; mx <= hi; mx++ {
		for mz := loZ; mz <= hiZ; mz++ {
			f, ok := g.featureAt(mx, mz)
			if !ok || f.kind == boulder || f.kind == rock {
				continue
			}
			for i := range t.Tops {
				x, z := FarSample(tx, tz, i)
				if top, leaf, ok := f.canopy(float64(x)+0.5, float64(z)+0.5); ok && top > t.Tops[i] {
					t.Tops[i], t.Blocks[i] = top, leaf
				}
			}
		}
	}
}

// canopy is roughly the y above the feature's leaves over x,z, if they cover it.
func (f feature) canopy(x, z float64) (int32, Block, bool) {
	if f.kind == bush {
		cx, cz := f.middle()
		r, ry := 3.5+float64(f.r%3)*0.75, 2.5+float64(f.r>>4%3)*0.5
		return dome(math.Hypot(x-cx, z-cz), float64(f.ground)+1.5, r, ry, Leaves)
	}
	t := f.tree()
	half := float64(t.width) / 2
	d := math.Hypot(x-float64(t.x)-half, z-float64(t.z)-half)
	top := float64(t.top)
	switch t.kind {
	case oakTree:
		r := oakCrown(t.r)
		return dome(d, top-1, r, r*0.75, Leaves)
	case pineTree:
		y := top + 4 - max(0, (d-0.9)/pineTaper) // the narrowest its tiers get
		return int32(y), PineLeaves, y >= float64(t.ground+2*Meter)
	}
	r := 4.0 + float64(t.r%2)
	return dome(d, top-2, r, r*1.6, Leaves)
}

// dome is the y above a blob of radii r and ry around height cy, d across from its middle; only
// its core counts, inside the roughened edge.
func dome(d, cy, r, ry float64, leaf Block) (int32, Block, bool) {
	n := d * d / (r * r)
	if n >= 0.75 {
		return 0, Air, false
	}
	return int32(math.Floor(cy + ry*math.Sqrt(0.875-n) + 0.5)), leaf, true
}
