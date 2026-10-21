package maps

import (
	"math"

	"scabbage/internal/world"
)

// The sample is laid out in metre cells; y counts metres up from the first one above the grass.
const (
	sampleWidth = 24
	sampleDepth = 16
	m           = world.Metre
)

// Captain Pip gives the quest; the gate opens while the hidden switch is on; the bell finishes it.
const (
	pipScript = `-- Captain Pip asks for help and gives a quest.
npc("Captain Pip")

function on_talk(player)
  if recall(player, "pip_treasure") then
    talk(player, "You found my treasure! Ask a grown-up to /map reset this map to play again.")
    return
  end
  ask(player, "Ahoy! My treasure house is shut tight. Will you help me open it?", {"Yes!", "Not now"}, function(answer)
    if answer == "Yes!" then
      quest(player, "Captain Pip's treasure", "Jump the pit, then follow the wire to a hidden switch")
      talk(player, "Hop over the pit on the stepping stone. The wire leads to a switch that opens my house!")
    else
      talk(player, "Come back when you're ready, matey.")
    end
  end)
end`
	gateScript = `-- The wire from the hidden switch powers this block, which opens the treasure house.
function on_power(on)
  local block = "cobblestone"
  if on then
    block = "air"
    say("Rumble rumble... the treasure house opens!")
  end
  for dy = 0, 5 do
    for dz = -7, -5 do
      for dx = 2, 4 do
        set_block(here.x + dx, here.y + dy, here.z + dz, block)
      end
    end
  end
end`
	bellScript = `-- Ring the bell to finish Captain Pip's quest.
function on_touch(player)
  if recall(player, "pip_treasure") then
    say("Ding! " .. player .. " rang the bell again.")
    return
  end
  remember(player, "pip_treasure", true)
  finish_quest(player, "Captain Pip's treasure")
  say("Ding ding! " .. player .. " found Captain Pip's treasure!")
end`
)

// Sample is a small adventure to play and to learn from, with its grass at y ground: talk to
// Captain Pip, jump a pit past a checkpoint, follow the wire to a switch and open the treasure house.
func Sample(ground int32) Map {
	b := newBuilder(sampleWidth, sampleDepth, ground)
	b.yard()
	b.pit()
	b.house()
	b.wire()
	b.pad(12, 0, 8, world.Checkpoint)
	b.script(4, 0, 11, pipScript)
	b.script(15, 0, 9, gateScript)
	b.script(19, 0, 10, bellScript)
	b.m.Start = &Start{X: 2.5 * m, Y: float64(ground + 1), Z: 7.5 * m, Yaw: -math.Pi / 2}
	return b.m
}

type builder struct {
	m      Map
	a      world.Area
	ground int32
}

func newBuilder(w, d, ground int32) *builder {
	mp := Map{Name: "Pip", Width: w * m, Depth: d * m}
	mp.Blocks = make([]world.Block, mp.Area(0, 0).Volume())
	return &builder{m: mp, a: mp.Area(0, 0), ground: ground}
}

func (b *builder) set(x, y, z int32, blk world.Block) { b.m.Blocks[b.a.Index(x, y, z)] = blk }

// bottom is the y of the lowest block in metre layer y.
func (b *builder) bottom(y int32) int32 { return b.ground + 1 + y*m }

// fill puts blk in every metre cell between two corners, both included.
func (b *builder) fill(x0, y0, z0, x1, y1, z1 int32, blk world.Block) {
	fillBlocks(x0*m, b.bottom(y0), z0*m, x1*m+m-1, b.bottom(y1)+m-1, z1*m+m-1, func(x, y, z int32) { b.set(x, y, z, blk) })
}

// dot puts one small block in the middle of a metre cell's floor.
func (b *builder) dot(x, y, z int32, blk world.Block) { b.set(x*m+m/2, b.bottom(y), z*m+m/2, blk) }

// pad covers a metre cell's floor with blk.
func (b *builder) pad(x, y, z int32, blk world.Block) {
	fillBlocks(x*m, b.bottom(y), z*m, x*m+m-1, b.bottom(y), z*m+m-1, func(x, y, z int32) { b.set(x, y, z, blk) })
}

func (b *builder) script(x, y, z int32, src string) {
	b.dot(x, y, z, world.ScriptBlock)
	p := world.Pos{X: x*m + m/2, Y: b.bottom(y), Z: z*m + m/2}
	b.m.Scripts = append(b.m.Scripts, Script{Pos: p, Source: src})
}

// yard is the ground and a brick wall with a gap to the west and torches on its corners.
func (b *builder) yard() {
	w, d, g := b.m.Width-1, b.m.Depth-1, b.ground
	set := func(blk world.Block) func(x, y, z int32) { return func(x, y, z int32) { b.set(x, y, z, blk) } }
	fillBlocks(0, 0, 0, w, 0, d, set(world.Bedrock))
	fillBlocks(0, 1, 0, w, g-3*m, d, set(world.Stone))
	fillBlocks(0, g-3*m+1, 0, w, g-1, d, set(world.Dirt))
	fillBlocks(0, g, 0, w, g, d, set(world.Grass))
	w, d = sampleWidth-1, sampleDepth-1
	b.fill(0, 0, 0, w, 1, 0, world.Brick)
	b.fill(0, 0, d, w, 1, d, world.Brick)
	b.fill(0, 0, 0, 0, 1, d, world.Brick)
	b.fill(w, 0, 0, w, 1, d, world.Brick)
	b.fill(0, 0, 7, 0, 1, 8, world.Air)
	for _, c := range [][2]int32{{0, 0}, {w, 0}, {0, d}, {w, d}, {0, 6}, {0, 9}} {
		b.dot(c[0], 2, c[1], world.Torch)
	}
}

// pit cuts across the yard with water at the bottom, one stepping stone and steps to climb out.
func (b *builder) pit() {
	z := int32(sampleDepth - 2)
	b.fill(8, -3, 1, 10, -1, z, world.Air)
	b.fill(8, -4, 1, 10, -4, z, world.Water)
	b.fill(9, -4, 7, 9, -1, 8, world.Cobblestone)
	b.fill(10, -4, z, 10, -4, z, world.Cobblestone)
	b.fill(9, -4, z, 9, -3, z, world.Cobblestone)
	b.fill(8, -4, z, 8, -2, z, world.Cobblestone)
}

// house is a cobblestone room shut by a doorway the gate script opens, with a chest and the bell.
func (b *builder) house() {
	b.fill(16, 0, 4, 22, 4, 11, world.Cobblestone)
	b.fill(17, 0, 5, 21, 3, 10, world.Air)
	b.dot(19, 4, 7, world.Lamp)
	b.dot(19, 4, 9, world.Lamp)
	b.dot(21, 0, 6, world.Chest)
}

// wire runs from the switch hidden behind the house to the gate script, along cell middles.
func (b *builder) wire() {
	b.dot(20, 0, 13, world.Switch)
	y, mid := b.bottom(0), int32(m/2)
	wire := func(x, _, z int32) { b.set(x, y, z, world.Wire) }
	fillBlocks(15*m+mid, y, 9*m+mid+1, 15*m+mid, y, 13*m+mid, wire)
	fillBlocks(15*m+mid, y, 13*m+mid, 20*m+mid-1, y, 13*m+mid, wire)
}

// fillBlocks calls put for every block between two corners, both included.
func fillBlocks(x0, y0, z0, x1, y1, z1 int32, put func(x, y, z int32)) {
	for x := x0; x <= x1; x++ {
		for y := y0; y <= y1; y++ {
			for z := z0; z <= z1; z++ {
				put(x, y, z)
			}
		}
	}
}
