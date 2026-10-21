package world

import "testing"

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
		Checkpoint: "checkpoint",
	}
	for id, name := range pinned {
		if Info(id).Name != name {
			t.Errorf("block %d = %q, want %q", id, Info(id).Name, name)
		}
	}
	if len(blocks) != int(Checkpoint)+1 {
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
