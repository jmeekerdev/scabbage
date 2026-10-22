import { describe, expect, it } from "vitest";
import { Shape } from "../world/shapes";
import { Smooth } from "../world/soft-corners";
import { Wiring } from "../world/wire";
import { buildRenderTable, BLOCK_STYLES } from "./blocks";
import { TEXTURE_NAMES, paintTextures, TEXTURE_SIZE } from "./textures";

// A slice of the server's block table (server/internal/world/block.go).
const types = [
  { id: 0, name: "air", transparent: true },
  { id: 1, name: "stone", transparent: false },
  { id: 2, name: "grass", transparent: false },
  { id: 3, name: "water", transparent: true },
  { id: 4, name: "mystery", transparent: false },
  { id: 5, name: "brick_stairs_east", transparent: true, shape: Shape.STAIRS_EAST },
  { id: 6, name: "torch", transparent: true, shape: Shape.TORCH, light: 11 },
  { id: 7, name: "red_wool", transparent: false },
  { id: 8, name: "piston_east_on", transparent: false },
  { id: 9, name: "piston", transparent: false },
  { id: 10, name: "counter_3_on", transparent: false },
  { id: 11, name: "note_block_2", transparent: false },
  { id: 12, name: "counter", transparent: false },
  { id: 14, name: "diode_west_on", transparent: false },
];
const layer = (name: string) => TEXTURE_NAMES.indexOf(name);

describe("buildRenderTable", () => {
  const t = buildRenderTable(types);

  it("softens natural cubes, makes room at water and pins at anything built", () => {
    const types2 = [...types, { id: 13, name: "stone_slab", transparent: true, shape: Shape.SLAB }];
    const kinds = buildRenderTable(types2).smooth;
    const { EMPTY, NATURAL, BUILT } = Smooth;
    expect([...kinds.subarray(0, 7)]).toEqual([
      EMPTY,
      NATURAL,
      NATURAL,
      EMPTY,
      BUILT,
      BUILT,
      BUILT,
    ]);
    expect(kinds[13]).toBe(Smooth.BUILT);
  });

  it("gives grass a green top, dirt bottom and grassy sides", () => {
    expect(t.layers[2 * 6 + 2]).toBe(layer("grass_top"));
    expect(t.layers[2 * 6 + 3]).toBe(layer("dirt"));
    expect(t.layers[2 * 6 + 0]).toBe(layer("grass_side"));
  });

  it("never draws air and marks the server's see-through blocks", () => {
    expect(t.layers[0]).toBe(-1);
    expect([...t.transparent.subarray(0, 13)]).toEqual([1, 0, 0, 1, 0, 1, 1, 0, 0, 0, 0, 0, 0]);
  });

  it("blends only water", () => {
    expect([...t.translucent.subarray(0, 13)]).toEqual([0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0]);
  });

  it("dresses slabs and stairs in their material and keeps shapes and glow", () => {
    expect(t.layers[5 * 6 + 2]).toBe(layer("brick"));
    expect(t.shape[5]).toBe(Shape.STAIRS_EAST);
    expect(t.light[6]).toBe(11);
    expect(t.layers[6 * 6]).toBe(layer("torch"));
    expect(t.layers[7 * 6]).toBe(layer("red_wool"));
  });

  it("turns a piston's face the way it pushes", () => {
    const faces = (id: number) => [...t.layers.slice(id * 6, id * 6 + 6)];
    const [side, front] = [layer("piston_side"), layer("piston_front")];
    expect(faces(8)).toEqual([layer("piston_front_on"), side, side, side, side, side]);
    expect(faces(9)).toEqual([side, side, side, side, side, front]);
  });

  it("turns a diode's face the way it gives power", () => {
    const side = layer("diode_side");
    expect([...t.layers.slice(14 * 6, 15 * 6)]).toEqual([
      side,
      layer("diode_front_on"),
      side,
      side,
      side,
      side,
    ]);
  });

  it("shows a gate's mode on its sides and its light on its front", () => {
    expect(BLOCK_STYLES["gate_xor_east"]).toMatchObject({ top: "gate_xor", side: "gate_xor" });
    expect(BLOCK_STYLES["gate_not_down_on"]).toMatchObject({
      side: "gate_not_on",
      front: "diode_front_on",
    });
    expect(BLOCK_STYLES["gate_and"]?.front).toBe("diode_front");
  });

  it("shows a delay's setting on its sides and a relay's cross", () => {
    expect(BLOCK_STYLES["delay_3_west"]).toMatchObject({ side: "delay_3", front: "diode_front" });
    expect(BLOCK_STYLES["delay_8_up_on"]).toMatchObject({
      side: "delay_8",
      front: "diode_front_on",
    });
    expect(BLOCK_STYLES["relay_south_on"]).toMatchObject({
      side: "relay_side",
      front: "diode_front_on",
    });
  });

  it("shows a capacitor's setting on its sides and whether a toggle gives power", () => {
    expect(BLOCK_STYLES["capacitor_10_east_on"]).toMatchObject({
      side: "capacitor_10",
      front: "diode_front_on",
    });
    expect([BLOCK_STYLES["toggle_on"]?.side, BLOCK_STYLES["toggle_1"]?.side]).toEqual([
      "toggle",
      "toggle_1",
    ]);
  });

  it("shows what sensors watch for on their sides and their power on top or in front", () => {
    expect(BLOCK_STYLES["player_detector_on"]).toMatchObject({
      top: "radio_top_on",
      side: "player_detector",
    });
    expect(BLOCK_STYLES["clock_midnight"]).toMatchObject({
      top: "radio_top",
      side: "clock_midnight",
    });
    expect(BLOCK_STYLES["weather_sensor_on"]).toMatchObject({
      top: "radio_top_on",
      side: "weather_sensor",
    });
    expect(BLOCK_STYLES["block_detector_up_on"]).toMatchObject({
      side: "diode_side",
      front: "block_detector_eye_on",
    });
  });

  it("dresses the new powered devices", () => {
    expect(BLOCK_STYLES["sticky_piston_east_on"]).toMatchObject({
      side: "piston_side",
      front: "sticky_piston_front_on",
    });
    for (const b of ["trapdoor", "trapdoor_open", "fence_gate", "fence_gate_open"])
      expect(BLOCK_STYLES[b]?.side, b).toBe(b);
    expect(BLOCK_STYLES["launch_pad_east_on"]).toMatchObject({
      top: "launch_pad_east_on",
      side: "launch_pad_side",
    });
    expect(BLOCK_STYLES["launch_pad"]?.top).toBe("launch_pad_north");
    expect(BLOCK_STYLES["number_display_7_on"]?.side).toBe("display_7");
    expect(BLOCK_STYLES["number_display"]?.side).toBe("display_0");
  });

  it("colours each timer speed's face and strings tripwires", () => {
    for (const b of ["timer_1", "timer_2_on", "timer_8", "timer_16_on", "tripwire", "tripwire_on"])
      expect(BLOCK_STYLES[b]?.side, b).toBe(b);
  });

  it("shows a wireless block's channel on its sides and its power on top", () => {
    expect(BLOCK_STYLES["transmitter"]).toEqual({
      top: "radio_top",
      side: "transmitter_0",
      bottom: "transmitter_0",
    });
    expect(BLOCK_STYLES["receiver_15_on"]).toMatchObject({
      top: "radio_top_on",
      side: "receiver_15",
    });
  });

  it("marks how blocks meet drawn wire", () => {
    const { WIRE, JOINS, HOLDS } = Wiring;
    const wired = buildRenderTable([
      { id: 0, name: "air", transparent: true },
      { id: 1, name: "wire", transparent: true },
      { id: 2, name: "wire_on", transparent: true },
      { id: 3, name: "stone_cable_on", transparent: false, solid: true },
      { id: 4, name: "switch", transparent: false, solid: true },
      { id: 5, name: "stone", transparent: false, solid: true },
      { id: 6, name: "pressure_plate", transparent: true },
      { id: 7, name: "water", transparent: true },
      { id: 8, name: "toggle_1", transparent: false, solid: true },
      { id: 9, name: "capacitor_2_east", transparent: false, solid: true },
      { id: 10, name: "clock_dawn_on", transparent: false, solid: true },
      { id: 11, name: "weather_sensor", transparent: false, solid: true },
      { id: 12, name: "sticky_piston_up", transparent: false, solid: true },
      { id: 13, name: "trapdoor_open", transparent: true },
      { id: 14, name: "fence_gate", transparent: true, solid: true },
      { id: 15, name: "launch_pad_west_on", transparent: true },
      { id: 16, name: "number_display_3", transparent: false, solid: true },
      { id: 17, name: "timer_8_on", transparent: false, solid: true },
      { id: 18, name: "tripwire", transparent: true },
    ]).wiring;
    const [both, device] = [WIRE | JOINS, JOINS | HOLDS];
    expect([...wired!]).toEqual([
      0,
      both,
      both,
      device,
      device,
      HOLDS,
      JOINS,
      0,
      device,
      device,
      device,
      device,
      device,
      JOINS,
      device,
      JOINS,
      device,
      device,
      JOINS,
    ]);
  });

  it("dresses cables as their material with a line of wire through it", () => {
    for (const m of ["stone", "cobblestone", "planks", "brick", "dirt"]) {
      expect(BLOCK_STYLES[`${m}_cable`]?.side).toBe(`${m}_cable`);
      expect(BLOCK_STYLES[`${m}_cable_on`]?.top).toBe(`${m}_cable_on`);
    }
  });

  it("shows a counter's count and a note block's pitch on every face", () => {
    expect(t.layers[10 * 6 + 2]).toBe(layer("counter_3"));
    expect(t.layers[12 * 6 + 4]).toBe(layer("counter_0"));
    expect(t.layers[11 * 6]).toBe(layer("note_2"));
  });

  it("lights a counter's digits while it counts back down", () => {
    expect(BLOCK_STYLES.counter_down_3_on?.side).toBe("counter_down_3");
    expect(layer("counter_down_3")).toBeGreaterThan(layer("counter_9"));
  });

  it("draws script blocks lit while powered", () => {
    expect(BLOCK_STYLES.script_block?.side).toBe("script_block");
    expect(BLOCK_STYLES.script_block_on?.side).toBe("script_block_on");
  });

  it("paints dungeon treasure", () => {
    for (const name of ["chest", "chest_open", "boss_altar", "gold_block", "gem_block"])
      expect(BLOCK_STYLES[name], name).toBeDefined();
    expect(BLOCK_STYLES.chest.top).not.toBe(BLOCK_STYLES.chest_open.top);
  });

  it("paints checkpoints for adventure maps", () => {
    expect(BLOCK_STYLES.checkpoint?.top).toBe("checkpoint");
  });

  it("paints the generator's plants and ground, with flower heads seen from above", () => {
    for (const name of ["tall_grass", "gravel", "pine_leaves"])
      expect(BLOCK_STYLES[name], name).toBeDefined();
    for (const c of ["red", "yellow", "blue", "white"]) {
      const flower = BLOCK_STYLES[`${c}_flower`];
      expect(flower?.top).toBe(`${c}_flower_top`);
      expect(flower?.side).toBe(`${c}_flower`);
    }
  });

  it("looks up built-ins from the second range at 4096", () => {
    const wide = buildRenderTable([...types, { id: 4100, name: "brick", transparent: false }]);
    expect(wide.transparent[4100]).toBe(0);
    expect(wide.layers[4100 * 6 + 2]).toBe(layer("brick"));
    expect(wide.transparent[4099]).toBe(1);
  });

  it("shows unknown blocks with a placeholder texture instead of failing", () => {
    expect(t.layers[4 * 6]).toBe(layer("missing"));
  });
});

describe("textures", () => {
  it("paints every texture the block styles use", () => {
    for (const s of Object.values(BLOCK_STYLES)) {
      for (const name of [s.top, s.side, s.bottom, s.front ?? s.side]) {
        expect(TEXTURE_NAMES).toContain(name);
      }
    }
  });

  it("paints opaque pixels deterministically", () => {
    const a = paintTextures();
    expect(a.length).toBe(TEXTURE_NAMES.length * TEXTURE_SIZE * TEXTURE_SIZE * 4);
    expect(paintTextures()).toEqual(a);
    const stone = layer("stone") * TEXTURE_SIZE * TEXTURE_SIZE * 4;
    expect(a[stone + 3]).toBe(255);
  });
});
