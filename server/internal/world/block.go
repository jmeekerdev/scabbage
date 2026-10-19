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
	Light       uint8 // how brightly it glows, up to MaxLight
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
		{ID: Stone, Name: "stone", Solid: true, Transparent: false},
		{ID: Dirt, Name: "dirt", Solid: true, Transparent: false},
		{ID: Grass, Name: "grass", Solid: true, Transparent: false},
		{ID: Sand, Name: "sand", Solid: true, Transparent: false},
		{ID: Water, Name: "water", Solid: false, Transparent: true},
		{ID: Log, Name: "log", Solid: true, Transparent: false},
		{ID: Leaves, Name: "leaves", Solid: true, Transparent: true},
		{ID: Planks, Name: "planks", Solid: true, Transparent: false},
		{ID: Cobblestone, Name: "cobblestone", Solid: true, Transparent: false},
		{ID: Glass, Name: "glass", Solid: true, Transparent: true},
		{ID: Brick, Name: "brick", Solid: true, Transparent: false},
		{ID: Bedrock, Name: "bedrock", Solid: true},
	}
	for _, c := range woolColors {
		out = append(out, BlockInfo{Name: c + "_wool", Solid: true})
	}
	out = append(out,
		BlockInfo{Name: "lamp", Solid: true, Light: MaxLight},
		BlockInfo{Name: "torch", Transparent: true, Shape: TorchShape, Light: MaxLight - 1},
	)
	for _, m := range shapedMaterials {
		out = append(out, BlockInfo{Name: m + "_slab", Solid: true, Transparent: true, Shape: Slab})
	}
	for _, m := range shapedMaterials {
		for f, suffix := range facings {
			out = append(out, BlockInfo{
				Name: m + "_stairs" + suffix, Solid: true, Transparent: true, Shape: StairsNorth + Shape(f),
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

// Register adds or replaces a custom block, e.g. one approved by an admin.
func Register(b BlockInfo) error {
	if b.ID < FirstCustom {
		return ErrBuiltinID
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
	for _, b := range blocks {
		if b.Name == name {
			return true
		}
	}
	return false
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

// Valid reports whether b is a known block.
func (b Block) Valid() bool { return b == Air || Info(b).ID == b }

func (b Block) String() string { return Info(b).Name }
