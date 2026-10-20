// Package world holds voxel chunks, terrain generation, the world border and chunk storage.
package world

import (
	"errors"
	"fmt"
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
	StoneSlab   = Torch + 1        // one per shapedMaterials entry
	StoneStairs = StoneSlab + 4    // four facings per material: north, east, south, west
	Wire        = StoneStairs + 16 // each logic block is followed by its powered state
	Switch      = Wire + 2
	Button      = Switch + 2
	Plate       = Button + 2
	SignalLamp  = Plate + 2
	Door        = SignalLamp + 2 // closed, then open
	Piston      = Door + 2       // six facings, each followed by its powered state
	Timer       = Piston + 12
	Counter     = Timer + 2 // counts 0 to 9, each followed by its state with power coming in
	Teleporter  = Counter + 20
	NoteBlock   = Teleporter + 1 // eight pitches, each followed by its sounding state
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
	Usable      bool   // players switch it by hand
	Powered     bool   // a logic block's on state
	Variant     uint8  // how many IDs past the block players hold it is
	Facing      bool   // its item turned another way, so players place it as it is
	Ticks       bool   // the server steps it while its chunk is loaded
}

var (
	woolColors = []string{
		"white", "orange", "magenta", "light_blue", "yellow", "lime", "pink", "gray",
		"light_gray", "cyan", "purple", "blue", "brown", "green", "red", "black",
	}
	shapedMaterials = []string{"stone", "cobblestone", "planks", "brick"}
	facings         = []string{"", "_east", "_south", "_west", "_up", "_down"}
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
		for f, suffix := range facings[:4] {
			out = append(out, BlockInfo{
				Name: m + "_stairs" + suffix, Solid: true, Transparent: true, Shape: StairsNorth + Shape(f),
				MineMs: digs(m), Variant: uint8(f), Facing: f > 0,
			})
		}
	}
	out = append(out, logicBlocks()...)
	out = append(out, machines()...)
	for id := range out {
		out[id].ID = Block(id)
	}
	return out
}

// logicBlocks pairs each circuit block with its powered state.
func logicBlocks() []BlockInfo {
	var out []BlockInfo
	for _, off := range []BlockInfo{
		{Name: "wire", Transparent: true, Shape: Flat},
		{Name: "switch", Solid: true, Usable: true, MineMs: 300},
		{Name: "button", Solid: true, Usable: true, MineMs: 300},
		{Name: "pressure_plate", Transparent: true, Shape: Flat, MineMs: 300},
		{Name: "signal_lamp", Solid: true, MineMs: 300},
	} {
		out = append(out, withOn(off, "_on", 0)...)
	}
	out[len(out)-1].Light = MaxLight
	return out
}

// machines are the logic blocks that do something when powered.
func machines() []BlockInfo {
	door := withOn(BlockInfo{Name: "door", Solid: true, Usable: true, MineMs: 600}, "_open", 0)
	door[1].Solid, door[1].Transparent = false, true
	out := door
	for f, suffix := range facings {
		out = append(out, withOn(BlockInfo{Name: "piston" + suffix, Solid: true, MineMs: 600, Facing: f > 0}, "_on", 2*f)...)
	}
	out = append(out, withOn(BlockInfo{Name: "timer", Solid: true, MineMs: 300, Ticks: true}, "_on", 0)...)
	out = append(out, numbered(BlockInfo{Name: "counter", Solid: true, Usable: true, MineMs: 300}, 10)...)
	out = append(out, BlockInfo{Name: "teleporter", Transparent: true, Shape: Flat, Light: MaxLight / 2, MineMs: 300})
	return append(out, numbered(BlockInfo{Name: "note_block", Solid: true, Usable: true, MineMs: 300}, 8)...)
}

// withOn is b and its powered state, both `variant` IDs past their item.
func withOn(b BlockInfo, suffix string, variant int) []BlockInfo {
	b.Variant = uint8(variant)
	on := b
	on.Name, on.Powered, on.Variant, on.Facing = b.Name+suffix, true, uint8(variant+1), false
	return []BlockInfo{b, on}
}

// numbered is b in n numbered states, each with its powered state; the first keeps b's name.
func numbered(b BlockInfo, n int) []BlockInfo {
	var out []BlockInfo
	for i := range n {
		state := b
		if i > 0 {
			state.Name = fmt.Sprintf("%s_%d", b.Name, i)
		}
		out = append(out, withOn(state, "_on", 2*i)...)
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

// Item is the block as players hold it: stairs and pistons face north, logic blocks are off.
func (b Block) Item() Block { return b - Block(Info(b).Variant) }

// Placeable reports whether players may place b: an item, or an item turned to face some way.
func Placeable(b Block) bool {
	info := Info(b)
	return info.Variant == 0 || info.Facing
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
