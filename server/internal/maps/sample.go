package maps

import (
	"math"

	"scabbage/internal/world"
)

const (
	sampleWidth = 24
	sampleDepth = 16
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
  set_block(here.x + 1, here.y, here.z - 2, block)
  set_block(here.x + 1, here.y + 1, here.z - 2, block)
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
	b := newBuilder(sampleWidth, sampleDepth)
	g := ground
	b.yard(g)
	b.pit(g)
	b.house(g)
	b.wire(g)
	b.put(12, g+1, 8, world.Checkpoint)
	b.script(4, g+1, 11, pipScript)
	b.script(15, g+1, 9, gateScript)
	b.script(19, g+1, 10, bellScript)
	b.m.Start = &Start{X: 2.5, Y: float64(g + 1), Z: 7.5, Yaw: -math.Pi / 2}
	return b.m
}

type builder struct {
	m Map
	a world.Area
}

func newBuilder(w, d int32) *builder {
	m := Map{Name: "Pip", Width: w, Depth: d}
	m.Blocks = make([]world.Block, m.Area(0, 0).Volume())
	return &builder{m: m, a: m.Area(0, 0)}
}

func (b *builder) put(x, y, z int32, blk world.Block) { b.m.Blocks[b.a.Index(x, y, z)] = blk }

// fill puts blk in the box between two corners, both included.
func (b *builder) fill(x0, y0, z0, x1, y1, z1 int32, blk world.Block) {
	for x := x0; x <= x1; x++ {
		for y := y0; y <= y1; y++ {
			for z := z0; z <= z1; z++ {
				b.put(x, y, z, blk)
			}
		}
	}
}

func (b *builder) script(x, y, z int32, src string) {
	b.put(x, y, z, world.ScriptBlock)
	b.m.Scripts = append(b.m.Scripts, Script{Pos: world.Pos{X: x, Y: y, Z: z}, Source: src})
}

// yard is the ground and a brick wall with a gap to the west and torches on its corners.
func (b *builder) yard(g int32) {
	w, d := b.m.Width-1, b.m.Depth-1
	b.fill(0, 0, 0, w, 0, d, world.Bedrock)
	b.fill(0, 1, 0, w, g-4, d, world.Stone)
	b.fill(0, g-3, 0, w, g-1, d, world.Dirt)
	b.fill(0, g, 0, w, g, d, world.Grass)
	b.fill(0, g+1, 0, w, g+2, 0, world.Brick)
	b.fill(0, g+1, d, w, g+2, d, world.Brick)
	b.fill(0, g+1, 0, 0, g+2, d, world.Brick)
	b.fill(w, g+1, 0, w, g+2, d, world.Brick)
	b.fill(0, g+1, 7, 0, g+2, 8, world.Air)
	for _, c := range [][2]int32{{0, 0}, {w, 0}, {0, d}, {w, d}, {0, 6}, {0, 9}} {
		b.put(c[0], g+3, c[1], world.Torch)
	}
}

// pit cuts across the yard with water at the bottom, one stepping stone and steps to climb out.
func (b *builder) pit(g int32) {
	b.fill(8, g-2, 1, 10, g, b.m.Depth-2, world.Air)
	b.fill(8, g-3, 1, 10, g-3, b.m.Depth-2, world.Water)
	b.fill(9, g-3, 7, 9, g, 8, world.Cobblestone)
	z := b.m.Depth - 2
	b.put(10, g-3, z, world.Cobblestone)
	b.fill(9, g-3, z, 9, g-2, z, world.Cobblestone)
	b.fill(8, g-3, z, 8, g-1, z, world.Cobblestone)
}

// house is a cobblestone room shut by two blocks the gate script opens, with a chest and the bell.
func (b *builder) house(g int32) {
	b.fill(16, g+1, 4, 22, g+5, 11, world.Cobblestone)
	b.fill(17, g+1, 5, 21, g+4, 10, world.Air)
	b.put(19, g+5, 7, world.Lamp)
	b.put(19, g+5, 9, world.Lamp)
	b.put(21, g+1, 6, world.Chest)
}

// wire runs from the switch hidden behind the house to the gate script.
func (b *builder) wire(g int32) {
	b.put(20, g+1, 13, world.Switch)
	b.fill(15, g+1, 13, 19, g+1, 13, world.Wire)
	b.fill(15, g+1, 10, 15, g+1, 12, world.Wire)
}
