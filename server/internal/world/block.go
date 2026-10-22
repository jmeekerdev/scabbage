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
	WhiteWool    // 16 colors in woolColors order
	Lamp         = WhiteWool + 16
	Torch        = Lamp + 1
	StoneSlab    = Torch + 1        // one per shapedMaterials entry
	StoneStairs  = StoneSlab + 4    // four facings per material: north, east, south, west
	Wire         = StoneStairs + 16 // each logic block is followed by its powered state
	Switch       = Wire + 2
	Button       = Switch + 2
	Plate        = Button + 2
	SignalLamp   = Plate + 2
	Door         = SignalLamp + 2 // closed, then open
	Piston       = Door + 2       // six facings, each followed by its powered state
	Timer        = Piston + 12
	Counter      = Timer + 2 // counts 0 to 9, each followed by its state with power coming in
	Teleporter   = Counter + 20
	NoteBlock    = Teleporter + 1    // eight pitches, each followed by its sounding state
	Portal       = NoteBlock + 16    // a lobby pad that joins an arena's round
	CountingDown = Portal + 1        // a counter past 9 counting 1 to 8 back down, each with its powered state
	ScriptBlock  = CountingDown + 16 // runs its Lua when used or powered, then its powered state
	Chest        = ScriptBlock + 2   // gives loot when first used, then stays open
	BossAltar    = Chest + 2         // wakes a boss when used
	GoldBlock    = BossAltar + 1
	GemBlock     = GoldBlock + 1
	Checkpoint   = GemBlock + 1 // a pad that remembers where you come back after fainting
	TallGrass    = Checkpoint + 1
	RedFlower    = TallGrass + 1 // one per flowerColors entry
	Gravel       = RedFlower + 4
	PineLeaves   = Gravel + 1
)

// MaxLight is the brightest a block glows; light fades by one per block, so about ten meters.
const MaxLight = 10 * Meter

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
	Directed    bool   // gives power out its front a tick after its input changes; rechecked on load
	Conducts    bool   // carries power to its neighbors like wire
}

var (
	woolColors = []string{
		"white", "orange", "magenta", "light_blue", "yellow", "lime", "pink", "gray",
		"light_gray", "cyan", "purple", "blue", "brown", "green", "red", "black",
	}
	flowerColors    = []string{"red", "yellow", "blue", "white"}
	shapedMaterials = []string{"stone", "cobblestone", "planks", "brick"}
	cableMaterials  = []Block{Stone, Cobblestone, Planks, Brick, Dirt}
	facings         = []string{"", "_east", "_south", "_west", "_up", "_down"}
)

var (
	blocks = perSmallBlock(palette())
	later  = perSmallBlock(numberFrom(SecondBuiltin, laterPalette()))
)

// SecondBuiltin is where built-ins continue, past the IDs kept for painted blocks.
const SecondBuiltin Block = 4096

// Built-in blocks from SecondBuiltin, in laterPalette order.
const (
	Diode  = SecondBuiltin // passes power back to front; six facings, each followed by its powered state
	Cables = Diode + 12    // stone, cobblestone, planks, brick and dirt that carry power, each then on
	Gate   = Cables + 10   // GateModes, each facing six ways, each then on
	Delay  = Gate + 72     // Delays settings, each facing six ways, each then on
	Relay  = Delay + 96    // passes power back to front while a side is on; six facings, each then on
	// Transmitter powers every loaded receiver on its channel; Channels of them, each then on.
	Transmitter = Relay + 12
	Receiver    = Transmitter + 2*Channels // gives power while its channel is on; Channels of them
	// Capacitor keeps giving power a while after its input stops; Capacitors settings, each
	// facing six ways, each then on.
	Capacitor = Receiver + 2*Channels
	Toggle    = Capacitor + 12*Capacitors // flips on each time power arrives; off then giving, each then fed
	// PlayerDetector and CreatureDetector give power while someone is near; each then on.
	PlayerDetector   = Toggle + 4
	CreatureDetector = PlayerDetector + 2
	Clock            = CreatureDetector + 2 // gives power in one of ClockModes; each then on
	// BlockDetector gives power while a block is in front of it; six facings, each then on.
	BlockDetector = Clock + 2*6
	// WeatherSensor gives power while it rains or snows; then on.
	WeatherSensor = BlockDetector + 2*6
	// StickyPiston pushes like a piston and pulls the block back when power stops; six facings, each then on.
	StickyPiston = WeatherSensor + 2
	Trapdoor     = StickyPiston + 2*6 // closed, then open; opens with the trapdoors beside it
	FenceGate    = Trapdoor + 2       // closed, then open; opens with the fence gates joined to it
	// LaunchPad throws players along its facing while powered: north, east, south, west, up; each then on.
	LaunchPad = FenceGate + 2
	// NumberDisplay shows 0 to 9: the count of a counter beside it, or its own pulses; each then fed.
	NumberDisplay = LaunchPad + 2*5
	// Timers beat on and off every TimerBeats quarter seconds; each then on. Tapping a timer
	// steps through these and Timer by speed.
	Timers   = NumberDisplay + 2*10
	Tripwire = Timers + 2*4 // gives power while anyone is crossing it; then on
)

// TimerBeats is how many quarter seconds each of Timers stays on, then off; Timer itself beats 4.
var TimerBeats = []int{1, 2, 8, 16}

const (
	// Delays is how many settings a delay has, a quarter second apart.
	Delays = 8
	// Channels is how many channels wireless blocks have, shown as colors.
	Channels = 16
	// Capacitors is how many settings a capacitor has, a second apart.
	Capacitors = 10
)

// ClockModes are the parts of the day a clock sensor gives power in, in the order tapping cycles them.
var ClockModes = []string{"day", "night", "dawn", "dusk", "noon", "midnight"}

// GateModes are what a gate gives power for, in the order tapping cycles them.
var GateModes = []string{"and", "or", "xor", "nand", "nor", "not"}

// laterPalette lists the built-ins from SecondBuiltin by ID; only ever append to it.
func laterPalette() []BlockInfo {
	out := faced(BlockInfo{Name: "diode", Solid: true, MineMs: 300, Directed: true})
	out = append(out, cables()...)
	out = append(out, gates()...)
	out = append(out, delays()...)
	out = append(out, faced(BlockInfo{Name: "relay", Solid: true, MineMs: 300, Directed: true})...)
	out = append(out, numbered(BlockInfo{Name: "transmitter", Solid: true, Usable: true, MineMs: 300}, Channels)...)
	out = append(out, numbered(BlockInfo{Name: "receiver", Solid: true, Usable: true, MineMs: 300}, Channels)...)
	out = append(out, capacitors()...)
	out = append(out, numbered(BlockInfo{Name: "toggle", Solid: true, Usable: true, MineMs: 300}, 2)...)
	out = append(out, sensors()...)
	out = append(out, faced(BlockInfo{Name: "sticky_piston", Solid: true, MineMs: 600})...)
	out = append(out, hatches()...)
	out = append(out, turnedWays(BlockInfo{Name: "launch_pad", Transparent: true, Shape: Flat, MineMs: 300}, facings[:5])...)
	display := BlockInfo{Name: "number_display", Solid: true, Usable: true, Light: MaxLight / 4, MineMs: 300}
	out = append(out, numbered(display, 10)...)
	for i, beats := range TimerBeats {
		timer := BlockInfo{Name: fmt.Sprintf("timer_%d", beats), Solid: true, Usable: true, MineMs: 300, Ticks: true}
		out = append(out, withOn(timer, "_on", 2*i)...)
	}
	return append(out, withOn(BlockInfo{Name: "tripwire", Transparent: true, Shape: Flat, MineMs: 100}, "_on", 0)...)
}

// hatches are trapdoors and fence gates: closed and solid, or open to walk through.
func hatches() []BlockInfo {
	var out []BlockInfo
	for _, name := range []string{"trapdoor", "fence_gate"} {
		h := withOn(BlockInfo{Name: name, Solid: true, Transparent: name == "fence_gate", Usable: true, MineMs: 600}, "_open", 0)
		h[1].Solid, h[1].Transparent = false, true
		out = append(out, h...)
	}
	return out
}

// sensors are blocks the server checks while loaded, giving power while they notice something.
func sensors() []BlockInfo {
	sensor := BlockInfo{Solid: true, MineMs: 300, Ticks: true}
	var out []BlockInfo
	for _, name := range []string{"player_detector", "creature_detector"} {
		sensor.Name = name
		out = append(out, withOn(sensor, "_on", 0)...)
	}
	for m, mode := range ClockModes {
		clock := sensor
		clock.Name, clock.Usable = "clock_"+mode, true
		out = append(out, withOn(clock, "_on", 2*m)...)
	}
	sensor.Name = "block_detector"
	out = append(out, faced(sensor)...)
	sensor.Name = "weather_sensor"
	return append(out, withOn(sensor, "_on", 0)...)
}

// gates are directed blocks in each of GateModes; every mode is placeable as it is.
func gates() []BlockInfo {
	var out []BlockInfo
	for m, mode := range GateModes {
		out = append(out, setting(BlockInfo{Name: "gate_" + mode}, m)...)
	}
	return out
}

// delays are directed blocks waiting 1 to Delays quarter seconds; every setting is placeable.
func delays() []BlockInfo {
	var out []BlockInfo
	for n := range Delays {
		out = append(out, setting(BlockInfo{Name: fmt.Sprintf("delay_%d", n+1)}, n)...)
	}
	return out
}

// capacitors are directed blocks holding power 1 to Capacitors seconds; every setting is placeable.
func capacitors() []BlockInfo {
	var out []BlockInfo
	for n := range Capacitors {
		out = append(out, setting(BlockInfo{Name: fmt.Sprintf("capacitor_%d", n+1)}, n)...)
	}
	return out
}

// setting is the usable directed block b, set the n-th of its item's ways, facing each way.
func setting(b BlockInfo, n int) []BlockInfo {
	b.Solid, b.Usable, b.MineMs, b.Directed = true, true, 300, true
	out := faced(b)
	for i := range out {
		out[i].Variant += uint8(len(out) * n)
		out[i].Facing = !out[i].Powered && out[i].Variant > 0
	}
	return out
}

// cables are blocks that look and dig like their material but carry power like wire.
func cables() []BlockInfo {
	first := palette()
	var out []BlockInfo
	for _, m := range cableMaterials {
		cable := BlockInfo{Name: first[m].Name + "_cable", Solid: true, MineMs: first[m].MineMs, Conducts: true}
		out = append(out, withOn(cable, "_on", 0)...)
	}
	return out
}

// numberFrom gives blocks consecutive IDs from first.
func numberFrom(first Block, out []BlockInfo) []BlockInfo {
	for i := range out {
		out[i].ID = first + Block(i)
	}
	return out
}

// perSmallBlock turns the palette's digging times, written for blocks a meter across, into times
// for blocks a third that size.
func perSmallBlock(out []BlockInfo) []BlockInfo {
	for i := range out {
		out[i].MineMs /= Meter
	}
	return out
}

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
		BlockInfo{Name: "torch", Transparent: true, Shape: TorchShape, Light: MaxLight - Meter},
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
	out = append(out, nature()...)
	for id := range out {
		out[id].ID = Block(id)
	}
	return out
}

// logicBlocks pairs each circuit block with its powered state.
func logicBlocks() []BlockInfo {
	var out []BlockInfo
	for _, off := range []BlockInfo{
		{Name: "wire", Transparent: true, Shape: Flat, Conducts: true},
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
// nature lists the plants and ground that generator v3 adds.
func nature() []BlockInfo {
	out := []BlockInfo{{Name: "tall_grass", Transparent: true, Shape: Tuft}}
	for _, c := range flowerColors {
		out = append(out, BlockInfo{Name: c + "_flower", Transparent: true, Shape: Flower})
	}
	return append(out,
		BlockInfo{Name: "gravel", Solid: true, MineMs: 600},
		BlockInfo{Name: "pine_leaves", Solid: true, Transparent: true, MineMs: 200},
	)
}

func machines() []BlockInfo {
	door := withOn(BlockInfo{Name: "door", Solid: true, Usable: true, MineMs: 600}, "_open", 0)
	door[1].Solid, door[1].Transparent = false, true
	out := append(door, faced(BlockInfo{Name: "piston", Solid: true, MineMs: 600})...)
	out = append(out, withOn(BlockInfo{Name: "timer", Solid: true, Usable: true, MineMs: 300, Ticks: true}, "_on", 0)...)
	out = append(out, numbered(BlockInfo{Name: "counter", Solid: true, Usable: true, MineMs: 300}, 10)...)
	out = append(out, BlockInfo{Name: "teleporter", Transparent: true, Shape: Flat, Light: MaxLight / 2, MineMs: 300})
	out = append(out, numbered(BlockInfo{Name: "note_block", Solid: true, Usable: true, MineMs: 300}, 8)...)
	out = append(out, BlockInfo{Name: "portal", Transparent: true, Shape: Flat, Light: MaxLight / 2, MineMs: 300})
	out = append(out, countingDown()...)
	out = append(out, withOn(BlockInfo{Name: "script_block", Solid: true, Usable: true, MineMs: 300}, "_on", 0)...)
	out = append(out, treasures()...)
	return append(out, BlockInfo{Name: "checkpoint", Transparent: true, Shape: Flat, Light: MaxLight / 3, MineMs: 300})
}

// treasures are what dungeons hold: chests, the boss altar and the riches only loot gives.
func treasures() []BlockInfo {
	return []BlockInfo{
		{Name: "chest", Solid: true, Usable: true, MineMs: 800, Drop: Planks},
		{Name: "chest_open", Solid: true, MineMs: 800, Drop: Planks, Variant: 1},
		{Name: "boss_altar", Solid: true, Usable: true, Light: MaxLight / 2, MineMs: 3000, Drop: Cobblestone},
		{Name: "gold_block", Solid: true, MineMs: 800},
		{Name: "gem_block", Solid: true, Light: MaxLight / 3, MineMs: 800},
	}
}

// countingDown is the counter's states on its way back from 9, held as a counter.
func countingDown() []BlockInfo {
	var out []BlockInfo
	for n := 1; n <= 8; n++ {
		b := BlockInfo{Name: fmt.Sprintf("counter_down_%d", n), Solid: true, Usable: true, MineMs: 300}
		out = append(out, withOn(b, "_on", int(CountingDown-Counter)+2*(n-1))...)
	}
	return out
}

// faced is b turned each of the six facings, each followed by its powered state.
func faced(b BlockInfo) []BlockInfo { return turnedWays(b, facings) }

// turnedWays is b turned each of the given facings, each followed by its powered state.
func turnedWays(b BlockInfo, ways []string) []BlockInfo {
	var out []BlockInfo
	for f, suffix := range ways {
		turned := b
		turned.Name, turned.Facing = b.Name+suffix, f > 0
		out = append(out, withOn(turned, "_on", 2*f)...)
	}
	return out
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

// FirstCustom is the lowest ID for blocks players paint; they stay below SecondBuiltin.
const FirstCustom Block = 256

// IsCustom reports whether b is in the range kept for painted blocks.
func IsCustom(b Block) bool { return b >= FirstCustom && b < SecondBuiltin }

var (
	ErrBuiltinID = errors.New("custom block IDs run from FirstCustom to SecondBuiltin")
	customMu     sync.Mutex                          // serializes Register
	custom       atomic.Pointer[map[Block]BlockInfo] // copy-on-write, so lookups never lock
)

// customMineMs is how long painted blocks take to dig when they don't say.
const customMineMs = 250

// Register adds or replaces a custom block, e.g. one approved by an admin.
func Register(b BlockInfo) error {
	if !IsCustom(b.ID) {
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

// ForgetCustomBlocks unregisters every custom block, e.g. when the world is wiped.
func ForgetCustomBlocks() {
	customMu.Lock()
	defer customMu.Unlock()
	custom.Store(nil)
}

// Blocks returns every known block in ID order: the first built-ins, custom blocks, then later built-ins.
func Blocks() []BlockInfo {
	out := slices.Clone(blocks)
	if m := custom.Load(); m != nil {
		start := len(out)
		for _, info := range *m {
			out = append(out, info)
		}
		slices.SortFunc(out[start:], func(a, b BlockInfo) int { return int(a.ID) - int(b.ID) })
	}
	return append(out, later...)
}

// IsBuiltinName reports whether a built-in block is called name.
func IsBuiltinName(name string) bool {
	_, ok := Builtin(name)
	return ok
}

// Builtin finds the built-in block called name.
func Builtin(name string) (Block, bool) {
	for _, table := range [][]BlockInfo{blocks, later} {
		if i := slices.IndexFunc(table, func(b BlockInfo) bool { return b.Name == name }); i >= 0 {
			return table[i].ID, true
		}
	}
	return Air, false
}

// Named finds the built-in or custom block called name.
func Named(name string) (Block, bool) {
	if b, ok := Builtin(name); ok {
		return b, true
	}
	if m := custom.Load(); m != nil {
		for _, info := range *m {
			if info.Name == name {
				return info.ID, true
			}
		}
	}
	return Air, false
}

// Info returns a block's properties; unknown IDs behave like air.
func Info(b Block) BlockInfo {
	if int(b) < len(blocks) {
		return blocks[b]
	}
	if b >= SecondBuiltin && int(b-SecondBuiltin) < len(later) {
		return later[b-SecondBuiltin]
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
