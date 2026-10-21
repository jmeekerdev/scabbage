package world

// Dungeons are cobblestone rooms deep underground, each in its own region and reached by stairs
// from an arch on the surface. A region holds all of its dungeon, so chunks never need neighbors.
const (
	regionSize  = 64 * Metre
	dungeonOdds = 3          // one region in this many holds a dungeon
	roomOuter   = 11 * Metre // walls included, along x and z
	roomHeight  = 5 * Metre  // inside, floor to ceiling
	roomDepth   = 8 * Metre  // ground at least this thick over the ceiling
	wall        = Metre      // thickness of walls, floor and ceiling
	maxStairs   = 40 * Metre // steps, each one block up
	headroom    = 3 * Metre  // clear air over every step and through the door
	archHeight  = 5 * Metre  // from the top step to the arch's torch
)

// dungeon is a room whose outer walls start at x0,z0 and whose lowest air is at y floor.
type dungeon struct {
	x0, z0, floor int32
	steps         int32 // stairs from the door, the last one at the surface
}

func regionOf(v int32) int32 { return floorDiv(v, regionSize) }

// dungeonIn finds the dungeon in region rx,rz, if it has one.
func (g *GenV3) dungeonIn(rx, rz int32) (dungeon, bool) {
	r := hash(g.dungeons, rx, rz)
	if r%dungeonOdds != 0 || (rx >= -1 && rx <= 0 && rz >= -1 && rz <= 0) {
		return dungeon{}, false
	}
	d := dungeon{x0: rx*regionSize + (44+int32(r>>8%8))*Metre, z0: rz*regionSize + (8+int32(r>>16%40))*Metre}
	low := int32(MaxY)
	for i := int32(0); i <= 2; i++ {
		for j := int32(0); j <= 2; j++ {
			low = min(low, g.SurfaceY(d.x0+i*(roomOuter-1)/2, d.z0+j*(roomOuter-1)/2))
		}
	}
	d.floor = low - roomDepth - roomHeight - wall + 1
	if d.floor-wall < 2*Metre {
		return dungeon{}, false
	}
	z := d.door()
	for ; d.steps < maxStairs; d.steps++ {
		x, y := d.x0-1-d.steps, d.floor+d.steps
		if h := g.SurfaceY(x, z); y >= h {
			d.steps++
			return d, h > seaLevel+beachRise
		}
	}
	return dungeon{}, false
}

// door is the z in the middle of the doorway in the room's west wall, where the stairs start.
func (d dungeon) door() int32 { return d.z0 + roomOuter/2 }

// top is the y of the ceiling's highest block.
func (d dungeon) top() int32 { return d.floor + roomHeight + wall - 1 }

// build calls put for every block of the room, its stairs and the arch over the way in.
func (d dungeon) build(put func(x, y, z int32, b Block)) {
	d.room(put)
	z := d.door()
	air := func(x, y, z int32) { put(x, y, z, Air) }
	fillBox(d.x0, d.floor, z-Metre/2, d.x0+wall-1, d.floor+headroom-1, z+Metre/2, air)
	for i := range d.steps {
		x, y := d.x0-1-i, d.floor+i
		for side := z - Metre/2; side <= z+Metre/2; side++ {
			put(x, y, side, StoneStairs+3) // rising to the west
		}
		fillBox(x, y+1, z-Metre/2, x, y+headroom, z+Metre/2, air)
	}
	d.arch(d.x0-d.steps, d.floor+d.steps-1, z, put)
}

// room is a hollow cobblestone box with lamps in its ceiling, two chests and the altar.
func (d dungeon) room(put func(x, y, z int32, b Block)) {
	x1, z1 := d.x0+roomOuter-1, d.z0+roomOuter-1
	fillBox(d.x0, d.floor-wall, d.z0, x1, d.top(), z1, func(x, y, z int32) { put(x, y, z, Cobblestone) })
	fillBox(d.x0+wall, d.floor, d.z0+wall, x1-wall, d.floor+roomHeight-1, z1-wall, func(x, y, z int32) { put(x, y, z, Air) })
	for _, at := range [][2]int32{{3, 3}, {3, 7}, {7, 3}, {7, 7}} {
		put(d.x0+at[0]*Metre+Metre/2, d.floor+roomHeight, d.z0+at[1]*Metre+Metre/2, Lamp)
	}
	back := x1 - wall
	put(back, d.floor, d.z0+wall, Chest)
	put(back, d.floor, z1-wall, Chest)
	put(back, d.floor, d.door(), BossAltar)
}

// arch stands two pillars and a lintel with a torch around the top step at x,y,z.
func (d dungeon) arch(x, y, z int32, put func(x, y, z int32, b Block)) {
	stone := func(x, y, z int32) { put(x, y, z, Cobblestone) }
	x0, x1 := x-Metre/2, x+Metre/2
	inner, outer := int32(Metre/2+1), int32(Metre/2+Metre)
	fillBox(x0, y+1, z-outer, x1, y+headroom, z-inner, stone)
	fillBox(x0, y+1, z+inner, x1, y+headroom, z+outer, stone)
	fillBox(x0, y+headroom+1, z-outer, x1, y+headroom+Metre, z+outer, stone)
	put(x, y+headroom+Metre+1, z, Torch)
}
