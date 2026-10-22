package world

import (
	"fmt"
	"testing"
)

func TestRegisterAddsCustomBlock(t *testing.T) {
	id := FirstCustom + 7
	if id.Valid() {
		t.Fatal("unregistered custom block is valid")
	}
	if err := Register(BlockInfo{ID: id, Name: "flower tile", Solid: true}); err != nil {
		t.Fatal(err)
	}
	if !id.Valid() || Info(id).Name != "flower tile" || !Info(id).Solid {
		t.Fatalf("Info = %+v", Info(id))
	}
	if b, ok := Named("flower tile"); !ok || b != id {
		t.Fatalf("Named found %v, %v", b, ok)
	}
	if b, ok := Named("brick"); !ok || b != Brick {
		t.Fatal("Named missed a built-in")
	}
	if _, ok := Named("nothing"); ok {
		t.Fatal("Named made up a block")
	}
	last := Blocks()[len(Blocks())-1]
	if last.ID < id {
		t.Fatalf("Blocks() is missing the custom block, ends with %+v", last)
	}
}

func TestRegisterRefusesBuiltinRange(t *testing.T) {
	if err := Register(BlockInfo{ID: Stone, Name: "fake"}); err == nil {
		t.Fatal("overwrote a built-in block")
	}
	if Info(Stone).Name != "stone" {
		t.Fatal("stone changed")
	}
}

func TestBlocksListsCustomInIDOrder(t *testing.T) {
	for _, id := range []Block{FirstCustom + 21, FirstCustom + 20} {
		if err := Register(BlockInfo{ID: id, Name: "b", Solid: true}); err != nil {
			t.Fatal(err)
		}
	}
	all := Blocks()
	for i := 1; i < len(all); i++ {
		if all[i].ID <= all[i-1].ID {
			t.Fatalf("out of order at %d: %v then %v", i, all[i-1].ID, all[i].ID)
		}
	}
}

func TestPaletteKeepsItsIDs(t *testing.T) {
	for id, b := range blocks {
		if b.ID != Block(id) {
			t.Fatalf("block %d has ID %d", id, b.ID)
		}
	}
	pinned := map[Block]string{
		WhiteWool: "white_wool", WhiteWool + 15: "black_wool", Lamp: "lamp", Torch: "torch",
		StoneSlab: "stone_slab", StoneStairs: "stone_stairs", StoneStairs + 15: "brick_stairs_west",
		Wire: "wire", SignalLamp + 1: "signal_lamp_on", Door + 1: "door_open", Piston + 9: "piston_up_on",
		Timer: "timer", Counter + 19: "counter_9_on", Teleporter: "teleporter", NoteBlock + 15: "note_block_7_on", Portal: "portal",
		CountingDown: "counter_down_1", CountingDown + 15: "counter_down_8_on",
		ScriptBlock: "script_block", ScriptBlock + 1: "script_block_on",
		Chest: "chest", Chest + 1: "chest_open", BossAltar: "boss_altar", GoldBlock: "gold_block", GemBlock: "gem_block",
		Checkpoint: "checkpoint", TallGrass: "tall_grass", RedFlower: "red_flower", RedFlower + 3: "white_flower",
		Gravel: "gravel", PineLeaves: "pine_leaves",
	}
	for id, name := range pinned {
		if Info(id).Name != name {
			t.Errorf("block %d = %q, want %q", id, Info(id).Name, name)
		}
	}
	if len(blocks) != int(PineLeaves)+1 {
		t.Fatalf("%d built-in blocks", len(blocks))
	}
}

func TestStairsFacingsAreConsecutive(t *testing.T) {
	for id := StoneStairs; id < StoneStairs+16; id += 4 {
		for f := range Block(4) {
			if got, want := Info(id+f).Shape, StairsNorth+Shape(f); got != want {
				t.Fatalf("block %v has shape %d, want %d", id+f, got, want)
			}
		}
	}
}

func TestTorchesGlowAndLetPlayersThrough(t *testing.T) {
	torch, lamp := Info(Torch), Info(Lamp)
	if torch.Light == 0 || lamp.Light != MaxLight || torch.Solid || !torch.Transparent {
		t.Fatalf("torch = %+v, lamp = %+v", torch, lamp)
	}
	if !torch.Targetable() || !Info(Door+1).Targetable() || Info(Water).Targetable() || Info(Air).Targetable() {
		t.Fatal("only solid, usable blocks and torches can be aimed at")
	}
	if torch.Boxes() != nil {
		t.Fatal("torches don't collide")
	}
}

func TestPlantsAreWalkedAndBuiltThrough(t *testing.T) {
	for _, b := range []Block{TallGrass, RedFlower, RedFlower + 1, RedFlower + 2, RedFlower + 3} {
		info := Info(b)
		if !info.Plant() || info.Solid || !info.Transparent || info.Targetable() || info.Boxes() != nil {
			t.Fatalf("%v = %+v", b, info)
		}
	}
	if Info(TallGrass).Shape != Tuft || Info(RedFlower).Shape != Flower || Info(Grass).Plant() || Info(Torch).Plant() {
		t.Fatal("only tufts and flowers are plants")
	}
}

func TestLogicBlocksHaveOnStates(t *testing.T) {
	for _, off := range []Block{Wire, Switch, Button, Plate, SignalLamp} {
		on := Info(off + 1)
		if Info(off).Powered || !on.Powered || (off+1).Item() != off || Drops(off+1) != off {
			t.Fatalf("%v and %v", off, on.Name)
		}
	}
	if !Info(Switch).Usable || !Info(Button).Usable || Info(Wire).Usable {
		t.Fatal("only switches and buttons are used by hand")
	}
	if Info(SignalLamp).Light != 0 || Info(SignalLamp+1).Light != MaxLight {
		t.Fatal("signal lamps glow only when on")
	}
	wire := Info(Wire)
	if wire.Solid || !wire.Targetable() || wire.Shape != Flat || Info(Plate).Shape != Flat {
		t.Fatalf("wire = %+v", wire)
	}
}

func TestShapesCollideWithTheirBoxes(t *testing.T) {
	slab := Info(StoneSlab).Boxes()
	if len(slab) != 1 || slab[0].Max[1] != 0.5 || !Info(StoneSlab).Transparent {
		t.Fatalf("slab = %+v", slab)
	}
	north := Info(StoneStairs).Boxes()
	if len(north) != 2 || north[1] != (Box{Min: [3]float64{0, 0.5, 0}, Max: [3]float64{1, 1, 0.5}}) {
		t.Fatalf("north stairs = %+v", north)
	}
	if Info(Stone).Boxes()[0] != FullBox {
		t.Fatal("stone is a full cube")
	}
}

func TestMiningTakesLongerForHarderBlocks(t *testing.T) {
	if Info(Dirt).MineMs >= Info(Stone).MineMs || Info(Leaves).MineMs >= Info(Log).MineMs {
		t.Fatalf("dirt %d stone %d leaves %d log %d",
			Info(Dirt).MineMs, Info(Stone).MineMs, Info(Leaves).MineMs, Info(Log).MineMs)
	}
	if Info(Torch).MineMs != 0 {
		t.Fatal("torches should pop off at once")
	}
	if Info(StoneStairs+2).MineMs != Info(Stone).MineMs || Info(StoneSlab).MineMs != Info(Stone).MineMs {
		t.Fatal("shaped stone should dig like stone")
	}
}

func TestSmallBlocksDigInAThirdOfAMeterBlocksTime(t *testing.T) {
	if Info(Stone).MineMs*Meter != 1500 || Info(Dirt).MineMs*Meter < 498 {
		t.Fatalf("stone %d dirt %d per small block", Info(Stone).MineMs, Info(Dirt).MineMs)
	}
}

func TestCustomBlocksTakeTimeToMine(t *testing.T) {
	id := FirstCustom + 30
	if err := Register(BlockInfo{ID: id, Name: "tile", Solid: true}); err != nil {
		t.Fatal(err)
	}
	if Info(id).MineMs == 0 {
		t.Fatal("custom block breaks instantly")
	}
}

func TestDrops(t *testing.T) {
	cases := map[Block]Block{
		Grass:           Dirt,
		Stone:           Cobblestone,
		Planks:          Planks,
		StoneStairs + 3: StoneStairs, // every facing drops the north-facing item
		StoneSlab + 2:   StoneSlab + 2,
	}
	for b, want := range cases {
		if got := Drops(b); got != want {
			t.Errorf("Drops(%v) = %v, want %v", b, got, want)
		}
	}
}

func TestItem(t *testing.T) {
	if StoneStairs.Item() != StoneStairs || (StoneStairs+5).Item() != StoneStairs+4 || Glass.Item() != Glass {
		t.Fatal("Item should turn stairs to face north and leave other blocks alone")
	}
}

func TestBuiltinFindsBlocksByName(t *testing.T) {
	if b, ok := Builtin("planks_slab"); !ok || b != StoneSlab+2 {
		t.Fatalf("planks_slab = %v, %v", b, ok)
	}
	if _, ok := Builtin("unobtainium"); ok {
		t.Fatal("found a block that does not exist")
	}
}

func TestVariantsAreHeldAsTheirItem(t *testing.T) {
	cases := map[Block]Block{
		Door + 1: Door, Piston + 7: Piston, Counter + 13: Counter, NoteBlock + 9: NoteBlock, Timer + 1: Timer,
		CountingDown + 5: Counter, ScriptBlock + 1: ScriptBlock,
	}
	for b, want := range cases {
		if b.Item() != want || Drops(b) != want {
			t.Errorf("%v is held as %v, want %v", b, b.Item(), want)
		}
	}
}

func TestOnlyItemsAndFacingsArePlaceable(t *testing.T) {
	for _, b := range []Block{Stone, StoneStairs + 2, Piston + 2, Piston + 10, Door, Counter} {
		if !Placeable(b) {
			t.Errorf("%v is placeable", b)
		}
	}
	for _, b := range []Block{Wire + 1, Door + 1, Piston + 3, Counter + 2, NoteBlock + 2, CountingDown} {
		if Placeable(b) {
			t.Errorf("%v is a state, not something to place", b)
		}
	}
}

func TestScriptBlocksAreUsedByHand(t *testing.T) {
	if !Info(ScriptBlock).Usable || !Info(ScriptBlock+1).Usable || !Placeable(ScriptBlock) || Placeable(ScriptBlock+1) {
		t.Fatalf("script block = %+v", Info(ScriptBlock))
	}
}

func TestOpenDoorsLetPlayersThrough(t *testing.T) {
	if !Info(Door).Solid || Info(Door+1).Solid || !Info(Door+1).Usable || !Info(Door+1).Transparent {
		t.Fatalf("door = %+v, open = %+v", Info(Door), Info(Door+1))
	}
}

func TestOnlyTimersTick(t *testing.T) {
	if !Info(Timer).Ticks || !Info(Timer+1).Ticks || Info(Wire).Ticks || Info(Counter).Ticks {
		t.Fatal("timers alone tick")
	}
}

func TestChestsOpenOnceAndBreakIntoPlanks(t *testing.T) {
	if !Info(Chest).Usable || Info(Chest+1).Usable || !Placeable(Chest) || Placeable(Chest+1) {
		t.Fatalf("chest = %+v, open = %+v", Info(Chest), Info(Chest+1))
	}
	if Drops(Chest) != Planks || Drops(Chest+1) != Planks {
		t.Errorf("chests drop %v and %v", Drops(Chest), Drops(Chest+1))
	}
}

func TestTreasureGlowsAndAltarsAreUsed(t *testing.T) {
	if !Info(BossAltar).Usable || Info(BossAltar).Light == 0 || Info(GemBlock).Light == 0 {
		t.Errorf("altar = %+v, gem = %+v", Info(BossAltar), Info(GemBlock))
	}
	if !Info(GoldBlock).Solid || Info(GoldBlock).MineMs == 0 {
		t.Errorf("gold = %+v", Info(GoldBlock))
	}
}

func TestForgetCustomBlocks(t *testing.T) {
	id := FirstCustom + 40
	if err := Register(BlockInfo{ID: id, Name: "to forget", Solid: true}); err != nil {
		t.Fatal(err)
	}
	ForgetCustomBlocks()
	if id.Valid() || Info(Stone).Name != "stone" {
		t.Fatal("custom block stayed or a built-in went")
	}
}

func TestLampsLightTenMetersAround(t *testing.T) {
	if reach := int(Info(Lamp).Light) - 1; reach < 9*Meter || reach > 10*Meter {
		t.Fatalf("a lamp lights %d blocks around, want about ten meters", reach)
	}
}

// useLater swaps in built-ins from SecondBuiltin for one test.
func useLater(t *testing.T, infos ...BlockInfo) {
	old := later
	later = numberFrom(SecondBuiltin, infos)
	t.Cleanup(func() { later = old })
}

func TestSecondRangeHoldsBuiltins(t *testing.T) {
	useLater(t, BlockInfo{Name: "diode", Solid: true}, BlockInfo{Name: "diode_on", Solid: true, Variant: 1})
	id := SecondBuiltin + 1
	if !id.Valid() || Info(id).Name != "diode_on" || Info(id).ID != id || id.Item() != SecondBuiltin {
		t.Fatalf("Info = %+v", Info(id))
	}
	if b, ok := Builtin("diode_on"); !ok || b != id {
		t.Fatalf("Builtin found %v, %v", b, ok)
	}
	if (SecondBuiltin + 2).Valid() {
		t.Fatal("an unlisted ID past the second range is valid")
	}
	if last := Blocks()[len(Blocks())-1]; last.ID != id {
		t.Fatalf("Blocks() ends with %+v", last)
	}
	if err := Register(BlockInfo{ID: SecondBuiltin, Name: "fake"}); err == nil {
		t.Fatal("a custom block took a built-in ID")
	}
}

func TestIsCustomCoversOnlyPaintedIDs(t *testing.T) {
	for id, want := range map[Block]bool{
		Stone: false, FirstCustom - 1: false, FirstCustom: true, SecondBuiltin - 1: true, SecondBuiltin: false,
	} {
		if IsCustom(id) != want {
			t.Errorf("IsCustom(%d) = %v", id, !want)
		}
	}
}

func TestDiodesFaceSixWaysAndAreDirected(t *testing.T) {
	for f := range Block(6) {
		off, on := Diode+2*f, Diode+2*f+1
		if off.Item() != Diode || on.Item() != Diode || !Info(off).Directed || !Info(on).Powered {
			t.Fatalf("diode facing %d: %+v, %+v", f, Info(off), Info(on))
		}
		if !Placeable(off) || Placeable(on) {
			t.Errorf("facing %d: placeable %v, on placeable %v", f, Placeable(off), Placeable(on))
		}
	}
	if Info(Diode).Name != "diode" || Info(Diode+3).Name != "diode_east_on" || Info(Piston).Directed {
		t.Fatalf("names %v %v", Info(Diode), Info(Diode+3))
	}
}

func TestCablesCarryPowerAndDigLikeTheirMaterial(t *testing.T) {
	for i, m := range []Block{Stone, Cobblestone, Planks, Brick, Dirt} {
		off, on := Cables+Block(2*i), Cables+Block(2*i+1)
		if Info(off).Name != Info(m).Name+"_cable" || !Info(off).Conducts || !Info(off).Solid {
			t.Fatalf("cable %d: %+v", i, Info(off))
		}
		if Info(off).MineMs != Info(m).MineMs || !Info(on).Powered || on.Item() != off || Placeable(on) {
			t.Errorf("cable %d: %+v, on %+v", i, Info(off), Info(on))
		}
	}
	if !Info(Wire).Conducts || Info(Stone).Conducts {
		t.Fatal("only wire and cables conduct")
	}
}

func TestGatesComeInSixModesEachFacingSixWays(t *testing.T) {
	modes := []string{"and", "or", "xor", "nand", "nor", "not"}
	for m, mode := range modes {
		for f, suffix := range facings {
			off := Gate + Block(12*m+2*f)
			info, on := Info(off), Info(off+1)
			if info.Name != "gate_"+mode+suffix || on.Name != info.Name+"_on" || !on.Powered {
				t.Fatalf("mode %d facing %d: %q, %q", m, f, info.Name, on.Name)
			}
			if !info.Directed || !info.Usable || off.Item() != Gate || !Placeable(off) || Placeable(off+1) {
				t.Errorf("%s: %+v", info.Name, info)
			}
		}
	}
}

func TestDelaysComeInEightSettingsEachFacingSixWays(t *testing.T) {
	for n := 1; n <= 8; n++ {
		for f, suffix := range facings {
			off := Delay + Block(12*(n-1)+2*f)
			info := Info(off)
			if info.Name != fmt.Sprintf("delay_%d%s", n, suffix) || !Info(off+1).Powered {
				t.Fatalf("setting %d facing %d: %q", n, f, info.Name)
			}
			if !info.Directed || !info.Usable || off.Item() != Delay || !Placeable(off) || Placeable(off+1) {
				t.Errorf("%s: %+v", info.Name, info)
			}
		}
	}
}

func TestRelaysFaceSixWays(t *testing.T) {
	info := Info(Relay + 2)
	if info.Name != "relay"+facings[1] || !info.Directed || info.Usable || (Relay+2).Item() != Relay {
		t.Fatalf("relay: %+v", info)
	}
}

func TestWirelessBlocksHaveSixteenChannels(t *testing.T) {
	for _, item := range []Block{Transmitter, Receiver} {
		last := item + 2*15
		if info := Info(last); !info.Usable || last.Item() != item || !Info(last+1).Powered {
			t.Fatalf("%v: %+v", item, info)
		}
	}
	if Info(Transmitter).Name != "transmitter" || Info(Receiver+2*15).Name != "receiver_15" {
		t.Fatalf("names: %q, %q", Info(Transmitter).Name, Info(Receiver+30).Name)
	}
}

func TestCapacitorsComeInTenSettingsEachFacingSixWays(t *testing.T) {
	for n := 1; n <= Capacitors; n++ {
		for f, suffix := range facings {
			off := Capacitor + Block(12*(n-1)+2*f)
			info := Info(off)
			if info.Name != fmt.Sprintf("capacitor_%d%s", n, suffix) || !Info(off+1).Powered {
				t.Fatalf("setting %d facing %d: %q", n, f, info.Name)
			}
			if !info.Directed || !info.Usable || off.Item() != Capacitor || !Placeable(off) || Placeable(off+1) {
				t.Errorf("%s: %+v", info.Name, info)
			}
		}
	}
}

func TestTogglesAreUsableAndShowWhetherTheyGivePower(t *testing.T) {
	if Info(Toggle).Name != "toggle" || Info(Toggle+2).Name != "toggle_1" || !Info(Toggle+3).Powered {
		t.Fatalf("names: %q, %q", Info(Toggle).Name, Info(Toggle+2).Name)
	}
	if !Info(Toggle+2).Usable || (Toggle+3).Item() != Toggle || Info(Toggle+4).Name == "toggle_2" {
		t.Fatalf("toggle: %+v", Info(Toggle+2))
	}
}

func TestSensorsTickWhileLoaded(t *testing.T) {
	for _, b := range []Block{PlayerDetector, CreatureDetector + 1, Clock + 5, BlockDetector + 6, WeatherSensor + 1} {
		if info := Info(b); !info.Ticks || !info.Solid || info.Directed {
			t.Errorf("%s: %+v", info.Name, info)
		}
	}
	if Info(WeatherSensor+1).Name != "weather_sensor_on" {
		t.Errorf("weather sensor: %q", Info(WeatherSensor+1).Name)
	}
	if Info(PlayerDetector+1).Name != "player_detector_on" || Info(CreatureDetector).Name != "creature_detector" {
		t.Fatalf("names: %q, %q", Info(PlayerDetector+1).Name, Info(CreatureDetector).Name)
	}
}

func TestClockSensorsCycleThroughPartsOfTheDay(t *testing.T) {
	for m, mode := range ClockModes {
		b := Clock + Block(2*m)
		if info := Info(b); info.Name != "clock_"+mode || !info.Usable || b.Item() != Clock || !Info(b+1).Powered {
			t.Errorf("%s: %+v", mode, info)
		}
	}
}

func TestBlockDetectorsFaceSixWays(t *testing.T) {
	for f, suffix := range facings {
		b := BlockDetector + Block(2*f)
		if Info(b).Name != "block_detector"+suffix || b.Item() != BlockDetector || !Placeable(b) {
			t.Errorf("facing %d: %+v", f, Info(b))
		}
	}
}

func TestStickyPistonsFaceSixWays(t *testing.T) {
	if Info(StickyPiston+3).Name != "sticky_piston_east_on" || (StickyPiston+11).Item() != StickyPiston {
		t.Fatalf("names %v %v", Info(StickyPiston+3), Info(StickyPiston+11))
	}
	if !Placeable(StickyPiston+2) || Info(StickyPiston).Directed {
		t.Fatalf("%+v", Info(StickyPiston+2))
	}
}

func TestTrapdoorsAndFenceGatesOpenToLetPlayersThrough(t *testing.T) {
	for _, b := range []Block{Trapdoor, FenceGate} {
		closed, open := Info(b), Info(b+1)
		if !closed.Solid || !closed.Usable || open.Solid || !open.Transparent || (b+1).Item() != b {
			t.Errorf("%v: %+v, open %+v", b, closed, open)
		}
	}
	if Info(Trapdoor+1).Name != "trapdoor_open" || Info(FenceGate).Name != "fence_gate" {
		t.Fatalf("names %v %v", Info(Trapdoor+1), Info(FenceGate))
	}
}

func TestLaunchPadsFaceAlongTheGroundOrUp(t *testing.T) {
	names := []string{"launch_pad", "launch_pad_east", "launch_pad_south", "launch_pad_west", "launch_pad_up"}
	for f, name := range names {
		pad := LaunchPad + Block(2*f)
		if Info(pad).Name != name || Info(pad+1).Name != name+"_on" || pad.Item() != LaunchPad {
			t.Errorf("facing %d: %+v, %+v", f, Info(pad), Info(pad+1))
		}
		if Info(pad).Solid || Info(pad).Shape != Flat || !Placeable(pad) {
			t.Errorf("facing %d: %+v", f, Info(pad))
		}
	}
	if NumberDisplay != LaunchPad+10 {
		t.Fatal("launch pads have no down facing")
	}
}

func TestNumberDisplaysShowZeroToNine(t *testing.T) {
	if Info(NumberDisplay+2*7+1).Name != "number_display_7_on" || (NumberDisplay+19).Item() != NumberDisplay {
		t.Fatalf("%+v", Info(NumberDisplay+15))
	}
	if !Info(NumberDisplay).Usable || Info(NumberDisplay).Light == 0 || !(NumberDisplay + 19).Valid() {
		t.Fatalf("%+v", Info(NumberDisplay))
	}
}

func TestTimersBeatAtFourMoreSpeeds(t *testing.T) {
	if Info(Timers+2*3+1).Name != "timer_16_on" || (Timers+7).Item() != Timers || !Info(Timers+2).Ticks {
		t.Fatalf("%+v", Info(Timers+7))
	}
	if !Info(Timer).Usable || !Info(Timers).Usable || Info(Timers+2).Facing {
		t.Fatalf("timers are tapped to change speed, and held as one item: %+v", Info(Timers+2))
	}
}

func TestTripwiresLetPlayersThrough(t *testing.T) {
	wire := Info(Tripwire)
	if wire.Name != "tripwire" || wire.Solid || !wire.Transparent || !Info(Tripwire+1).Powered || !(Tripwire + 1).Valid() {
		t.Fatalf("%+v", wire)
	}
}
