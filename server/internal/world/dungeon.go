package world

// Dungeons are cobblestone rooms deep underground, each in its own region and reached by stairs
// from an arch on the surface. A region holds all of its dungeon, so chunks never need neighbors.
const (
	regionSize  = 64
	dungeonOdds = 3  // one region in this many holds a dungeon
	roomOuter   = 11 // walls included, along x and z
	roomHeight  = 5  // inside, floor to ceiling
	roomDepth   = 8  // ground at least this thick over the ceiling
	maxStairs   = 40
)

// dungeon is a room whose outer walls start at x0,z0 and whose floor is at y floor.
type dungeon struct {
	x0, z0, floor int32
	steps         int32 // stairs from the door, the last one at the surface
}

func regionOf(v int32) int32 {
	if v < 0 {
		return (v+1)/regionSize - 1
	}
	return v / regionSize
}

// dungeonIn finds the dungeon in region rx,rz, if it has one.
func (g *GenV2) dungeonIn(rx, rz int32) (dungeon, bool) {
	r := hash(g.dungeons, rx, rz)
	if r%dungeonOdds != 0 || (rx >= -1 && rx <= 0 && rz >= -1 && rz <= 0) {
		return dungeon{}, false
	}
	d := dungeon{x0: rx*regionSize + 44 + int32(r>>8%8), z0: rz*regionSize + 8 + int32(r>>16%40)}
	low := int32(MaxY)
	for i := int32(0); i <= 2; i++ {
		for j := int32(0); j <= 2; j++ {
			low = min(low, g.SurfaceY(d.x0+i*(roomOuter-1)/2, d.z0+j*(roomOuter-1)/2))
		}
	}
	d.floor = low - roomDepth - roomHeight - 1
	if d.floor < 2 {
		return dungeon{}, false
	}
	z := d.door()
	for ; d.steps < maxStairs; d.steps++ {
		x := d.x0 - 1 - d.steps
		if h := g.SurfaceY(x, z); d.floor+1+d.steps >= h {
			d.steps++
			return d, h > seaLevel+1
		}
	}
	return dungeon{}, false
}

// door is the z of the doorway in the room's west wall, where the stairs start.
func (d dungeon) door() int32 { return d.z0 + roomOuter/2 }

// build calls put for every block of the room, its stairs and the arch over the way in.
func (d dungeon) build(put func(x, y, z int32, b Block)) {
	d.room(put)
	z := d.door()
	for y := d.floor + 1; y <= d.floor+3; y++ {
		put(d.x0, y, z, Air)
	}
	for i := range d.steps {
		x, y := d.x0-1-i, d.floor+1+i
		put(x, y, z, StoneStairs+3) // rising to the west
		for h := int32(1); h <= 3; h++ {
			put(x, y+h, z, Air)
		}
	}
	d.arch(d.x0-d.steps, d.floor+d.steps, z, put)
}

// room is a hollow cobblestone box with lamps in its ceiling, two chests and the altar.
func (d dungeon) room(put func(x, y, z int32, b Block)) {
	top := d.floor + roomHeight + 1
	for x := d.x0; x < d.x0+roomOuter; x++ {
		for z := d.z0; z < d.z0+roomOuter; z++ {
			for y := d.floor; y <= top; y++ {
				wall := x == d.x0 || z == d.z0 || x == d.x0+roomOuter-1 || z == d.z0+roomOuter-1
				if wall || y == d.floor || y == top {
					put(x, y, z, Cobblestone)
				} else {
					put(x, y, z, Air)
				}
			}
		}
	}
	for _, at := range [][2]int32{{3, 3}, {3, 7}, {7, 3}, {7, 7}} {
		put(d.x0+at[0], top, d.z0+at[1], Lamp)
	}
	back := d.x0 + roomOuter - 2
	put(back, d.floor+1, d.z0+1, Chest)
	put(back, d.floor+1, d.z0+roomOuter-2, Chest)
	put(back, d.floor+1, d.door(), BossAltar)
}

// arch stands two pillars and a lintel with a torch around the top step at x,y,z.
func (d dungeon) arch(x, y, z int32, put func(x, y, z int32, b Block)) {
	for _, side := range []int32{z - 1, z + 1} {
		for h := int32(1); h <= 3; h++ {
			put(x, y+h, side, Cobblestone)
		}
	}
	for side := z - 1; side <= z+1; side++ {
		put(x, y+4, side, Cobblestone)
	}
	put(x, y+5, z, Torch)
}

// chunks lists every chunk the dungeon's blocks reach into.
func (d dungeon) chunks() []ChunkPos {
	lo, _, _, _ := ChunkOf(d.x0-d.steps, d.floor, d.z0)
	hi, _, _, _ := ChunkOf(d.x0+roomOuter-1, d.floor+d.steps+5, d.z0+roomOuter-1)
	var out []ChunkPos
	for x := lo.X; x <= hi.X; x++ {
		for y := lo.Y; y <= hi.Y; y++ {
			for z := lo.Z; z <= hi.Z; z++ {
				out = append(out, ChunkPos{x, y, z})
			}
		}
	}
	return out
}
