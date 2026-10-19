// Package world holds voxel chunks, terrain generation, the world border and chunk storage.
package world

import (
	"errors"
	"slices"
	"sync"
	"sync/atomic"
)

// Block is a block type ID. IDs are stored in chunks, so existing values must never change.
type Block uint16

// Starter blocks. Append new blocks at the end.
const (
	Air Block = iota
	Stone
	Dirt
	Grass
	Sand
	Water
	Log
	Leaves
	Planks
	Cobblestone
	Glass
	Brick
	Bedrock
	WhiteWool   // 16 colors in woolColors order
	Lamp        = WhiteWool + 16
	Torch       = Lamp + 1
	StoneSlab   = Torch + 1     // one per shapedMaterials entry
	StoneStairs = StoneSlab + 4 // four facings per material: north, east, south, west
)

// MaxLight is the brightest a block glows; light fades by one per block.
const MaxLight = 12

// BlockInfo describes how a block behaves.
type BlockInfo struct {
	ID          Block
	Name        string
	Solid       bool // collides with players
	Transparent bool // neighbors' faces stay visible
	Shape       Shape
	Light       uint8  // how brightly it glows, up to MaxLight
	MineMs      uint16 // how long survival players dig to break it
	Drop        Block  // what breaking it gives, if not the block itself
}

var (
	woolColors = []string{
		"white", "orange", "magenta", "light_blue", "yellow", "lime", "pink", "gray",
		"light_gray", "cyan", "purple", "blue", "brown", "green", "red", "black",
	}
	shapedMaterials = []string{"stone", "cobblestone", "planks", "brick"}
	facings         = []string{"", "_east", "_south", "_west"}
)

var blocks = palette()

// palette lists the built-in blocks by ID; only ever append to it.
func palette() []BlockInfo {
	out := []BlockInfo{
		{ID: Air, Name: "air", Solid: false, Transparent: true},
		{ID: Stone, Name: "stone", Solid: true, Transparent: false, MineMs: 1500, Drop: Cobblestone},
		{ID: Dirt, Name: "dirt", Solid: true, Transparent: false, MineMs: 500},
		{ID: Grass, Name: "grass", Solid: true, Transparent: false, MineMs: 600, Drop: Dirt},
		{ID: Sand, Name: "sand", Solid: true, Transparent: false, MineMs: 500},
		{ID: Water, Name: "water", Solid: false, Transparent: true},
		{ID: Log, Name: "log", Solid: true, Transparent: false, MineMs: 1000},
		{ID: Leaves, Name: "leaves", Solid: true, Transparent: true, MineMs: 200},
		{ID: Planks, Name: "planks", Solid: true, Transparent: false, MineMs: 1000},
		{ID: Cobblestone, Name: "cobblestone", Solid: true, Transparent: false, MineMs: 1500},
		{ID: Glass, Name: "glass", Solid: true, Transparent: true, MineMs: 300},
		{ID: Brick, Name: "brick", Solid: true, Transparent: false, MineMs: 1500},
		{ID: Bedrock, Name: "bedrock", Solid: true},
	}
	for _, c := range woolColors {
		out = append(out, BlockInfo{Name: c + "_wool", Solid: true, MineMs: 400})
	}
	out = append(out,
		BlockInfo{Name: "lamp", Solid: true, Light: MaxLight, MineMs: 300},
		BlockInfo{Name: "torch", Transparent: true, Shape: TorchShape, Light: MaxLight - 1},
	)
	digs := func(name string) uint16 {
		return out[slices.IndexFunc(out, func(b BlockInfo) bool { return b.Name == name })].MineMs
	}
	for _, m := range shapedMaterials {
		out = append(out, BlockInfo{Name: m + "_slab", Solid: true, Transparent: true, Shape: Slab, MineMs: digs(m)})
	}
	for _, m := range shapedMaterials {
		for f, suffix := range facings {
			out = append(out, BlockInfo{
				Name: m + "_stairs" + suffix, Solid: true, Transparent: true, Shape: StairsNorth + Shape(f),
				MineMs: digs(m),
			})
		}
	}
	for id := range out {
		out[id].ID = Block(id)
	}
	return out
}

// FirstCustom is the lowest ID for blocks players paint; built-ins stay below it.
const FirstCustom Block = 256

var (
	ErrBuiltinID = errors.New("custom block IDs start at FirstCustom")
	customMu     sync.Mutex                          // serializes Register
	custom       atomic.Pointer[map[Block]BlockInfo] // copy-on-write, so lookups never lock
)

// customMineMs is how long painted blocks take to dig when they don't say.
const customMineMs = 750

// Register adds or replaces a custom block, e.g. one approved by an admin.
func Register(b BlockInfo) error {
	if b.ID < FirstCustom {
		return ErrBuiltinID
	}
	if b.MineMs == 0 {
		b.MineMs = customMineMs
	}
	customMu.Lock()
	defer customMu.Unlock()
	next := map[Block]BlockInfo{b.ID: b}
	if old := custom.Load(); old != nil {
		for id, info := range *old {
			if id != b.ID {
				next[id] = info
			}
		}
	}
	custom.Store(&next)
	return nil
}

// Blocks returns every known block in ID order: built-ins (indexed by ID), then custom blocks.
func Blocks() []BlockInfo {
	out := slices.Clone(blocks)
	if m := custom.Load(); m != nil {
		start := len(out)
		for _, info := range *m {
			out = append(out, info)
		}
		slices.SortFunc(out[start:], func(a, b BlockInfo) int { return int(a.ID) - int(b.ID) })
	}
	return out
}

// IsBuiltinName reports whether a built-in block is called name.
func IsBuiltinName(name string) bool {
	_, ok := Builtin(name)
	return ok
}

// Builtin finds the built-in block called name.
func Builtin(name string) (Block, bool) {
	for _, b := range blocks {
		if b.Name == name {
			return b.ID, true
		}
	}
	return Air, false
}

// Info returns a block's properties; unknown IDs behave like air.
func Info(b Block) BlockInfo {
	if int(b) < len(blocks) {
		return blocks[b]
	}
	if m := custom.Load(); m != nil {
		if info, ok := (*m)[b]; ok {
			return info
		}
	}
	return blocks[Air]
}

// Item is the block as players hold it: stairs face north until placed.
func (b Block) Item() Block {
	if s := Info(b).Shape; s >= StairsNorth && s <= StairsWest {
		return b - Block(s-StairsNorth)
	}
	return b
}

// Drops is the item breaking b gives.
func Drops(b Block) Block {
	if d := Info(b).Drop; d != Air {
		return d
	}
	return b.Item()
}

// Valid reports whether b is a known block.
func (b Block) Valid() bool { return b == Air || Info(b).ID == b }

func (b Block) String() string { return Info(b).Name }
