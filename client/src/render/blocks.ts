import type { RenderTable } from "../world/mesher";
import { customLayerName } from "./atlas";
import { isStairs, Shape } from "../world/shapes";
import { Smooth } from "../world/soft-corners";
import { Wiring } from "../world/wire";
import { CHANNELS, CLOCK_MODES, COUNT_DOWN, GATE_MODES } from "./machine-textures";
import { CABLE_MATERIALS, TEXTURE_NAMES, WOOL_COLORS } from "./textures";

export interface BlockStyle {
  top: string;
  side: string;
  bottom: string;
  front?: string; // the face a piston pushes from or a diode gives power out of
  translucent?: boolean;
  natural?: boolean; // ground the world grows, which soft corners round off
}

// Each piston facing's name suffix and the face it pushes from, in +x, -x, +y, -y, +z, -z order.
const FACINGS: [string, number][] = [
  ["", 5],
  ["_east", 0],
  ["_south", 4],
  ["_west", 1],
  ["_up", 2],
  ["_down", 3],
];
// Each launch pad facing's name suffix and the way its top's arrow points.
const PAD_FACINGS = [
  ["", "north"],
  ["_east", "east"],
  ["_south", "south"],
  ["_west", "west"],
  ["_up", "up"],
];
// The way the top arrow of a logic block facing along the ground points, by facing suffix.
const ARROW_WAYS: Record<string, string> = {
  "": "north",
  _east: "east",
  _south: "south",
  _west: "west",
};

/** A logic block off and on: its light in front and, facing along the ground, an arrow on top. */
function flowing(name: string, f: string, side: string, sideOn = side): [string, BlockStyle][] {
  const way = ARROW_WAYS[f];
  const style = (t: string, on: string): BlockStyle => ({
    ...same(t),
    front: `diode_front${on}`,
    ...(way && { top: `logic_arrow_${way}${on}` }),
  });
  return [
    [name, style(side, "")],
    [`${name}_on`, style(sideOn, "_on")],
  ];
}

const piston = (front: string): BlockStyle => ({ ...same("piston_side"), front });
const diode = (front: string): BlockStyle => ({ ...same("diode_side"), front });
const sensor = (side: string, top: string): BlockStyle => ({ top, side, bottom: side });
const numbered = (block: string, texture: string, n: number) =>
  Array.from({ length: n }, (_, i) => {
    const name = i ? `${block}_${i}` : block;
    return [
      [name, same(`${texture}_${i}`)],
      [`${name}_on`, same(`${texture}_${i}`)],
    ];
  }).flat();

const same = (t: string): BlockStyle => ({ top: t, side: t, bottom: t });
const natural = (style: BlockStyle): BlockStyle => ({ ...style, natural: true });

/** How each server block name looks; unknown names get the "missing" texture. */
export const BLOCK_STYLES: Record<string, BlockStyle> = {
  stone: natural(same("stone")),
  dirt: natural(same("dirt")),
  grass: natural({ top: "grass_top", side: "grass_side", bottom: "dirt" }),
  sand: natural(same("sand")),
  water: { ...same("water"), translucent: true },
  log: { top: "log_top", side: "log_side", bottom: "log_top" },
  leaves: natural(same("leaves")),
  planks: same("planks"),
  cobblestone: same("cobblestone"),
  glass: same("glass"),
  brick: same("brick"),
  bedrock: natural(same("bedrock")),
  lamp: same("lamp"),
  torch: same("torch"),
  ...Object.fromEntries(
    [
      ...["wire", "switch", "button", "pressure_plate", "signal_lamp", "timer", "script_block"],
      ...["timer_1", "timer_2", "timer_8", "timer_16", "tripwire"],
    ].flatMap((b) => [
      [b, same(b)],
      [`${b}_on`, same(`${b}_on`)],
    ]),
  ),
  door: same("door"),
  door_open: same("door_open"),
  ...Object.fromEntries(
    FACINGS.flatMap(([f]) => [
      [`piston${f}`, piston("piston_front")],
      [`piston${f}_on`, piston("piston_front_on")],
      ...flowing(`diode${f}`, f, "diode_side"),
      ...GATE_MODES.flatMap((m) => flowing(`gate_${m}${f}`, f, `gate_${m}`, `gate_${m}_on`)),
      ...Array.from({ length: 8 }, (_, i) =>
        flowing(`delay_${i + 1}${f}`, f, `delay_${i + 1}`),
      ).flat(),
      ...flowing(`relay${f}`, f, "relay_side"),
      ...Array.from({ length: 10 }, (_, i) =>
        flowing(`capacitor_${i + 1}${f}`, f, `capacitor_${i + 1}`),
      ).flat(),
      [`sticky_piston${f}`, piston("sticky_piston_front")],
      [`sticky_piston${f}_on`, piston("sticky_piston_front_on")],
      [`block_detector${f}`, diode("block_detector_eye")],
      [`block_detector${f}_on`, diode("block_detector_eye_on")],
    ]),
  ),
  ...Object.fromEntries(
    CABLE_MATERIALS.flatMap((m) => [
      [`${m}_cable`, same(`${m}_cable`)],
      [`${m}_cable_on`, same(`${m}_cable_on`)],
    ]),
  ),
  ...Object.fromEntries(numbered("counter", "counter", 10)),
  ...Object.fromEntries(
    COUNT_DOWN.flatMap((d) => [
      [`counter_down_${d}`, same(`counter_down_${d}`)],
      [`counter_down_${d}_on`, same(`counter_down_${d}`)],
    ]),
  ),
  ...Object.fromEntries(
    ["transmitter", "receiver"].flatMap((kind) =>
      Array.from({ length: CHANNELS }, (_, i) => {
        const name = i ? `${kind}_${i}` : kind;
        const side = `${kind}_${i}`;
        return [
          [name, { top: "radio_top", side, bottom: side }],
          [`${name}_on`, { top: "radio_top_on", side, bottom: side }],
        ];
      }).flat(),
    ),
  ),
  ...Object.fromEntries(
    [
      "player_detector",
      "creature_detector",
      "weather_sensor",
      ...CLOCK_MODES.map((m) => `clock_${m}`),
    ].flatMap((name) => [
      [name, sensor(name, "radio_top")],
      [`${name}_on`, sensor(name, "radio_top_on")],
    ]),
  ),
  ...Object.fromEntries(
    ["trapdoor", "trapdoor_open", "fence_gate", "fence_gate_open"].map((b) => [b, same(b)]),
  ),
  ...Object.fromEntries(
    PAD_FACINGS.flatMap(([suffix, way]) => [
      [`launch_pad${suffix}`, { ...same("launch_pad_side"), top: `launch_pad_${way}` }],
      [`launch_pad${suffix}_on`, { ...same("launch_pad_side"), top: `launch_pad_${way}_on` }],
    ]),
  ),
  ...Object.fromEntries(numbered("number_display", "display", 10)),
  toggle: same("toggle"),
  toggle_on: same("toggle"),
  toggle_1: same("toggle_1"),
  toggle_1_on: same("toggle_1"),
  teleporter: same("teleporter"),
  portal: same("portal"),
  ...Object.fromEntries(numbered("note_block", "note", 8)),
  ...Object.fromEntries(WOOL_COLORS.map((c) => [`${c}_wool`, same(`${c}_wool`)])),
  chest: { top: "chest_top", side: "chest_side", bottom: "planks" },
  chest_open: { top: "chest_open_top", side: "chest_side", bottom: "planks" },
  boss_altar: { top: "altar_top", side: "altar_side", bottom: "cobblestone" },
  gold_block: same("gold_block"),
  gem_block: same("gem_block"),
  checkpoint: same("checkpoint"),
  tall_grass: same("tall_grass"),
  gravel: natural(same("gravel")),
  pine_leaves: natural(same("pine_leaves")),
  ...Object.fromEntries(
    ["red", "yellow", "blue", "white"].map((c) => [
      `${c}_flower`,
      { top: `${c}_flower_top`, side: `${c}_flower`, bottom: `${c}_flower` },
    ]),
  ),
};

export interface BlockTypeInfo {
  id: number;
  name: string;
  transparent: boolean;
  solid?: boolean;
  shape?: number;
  light?: number;
  pixels?: Uint8Array; // custom blocks only
}

const builtinLayer = (name: string) => TEXTURE_NAMES.indexOf(name);

/** How a block looks: custom blocks use their own picture, slabs and stairs their material's. */
export function styleOf(t: BlockTypeInfo): BlockStyle {
  if (t.pixels?.length) return same(customLayerName(t.id));
  const material = t.name.replace(/_(slab|stairs)(_\w+)?$/, "");
  return BLOCK_STYLES[t.name] ?? BLOCK_STYLES[material] ?? same("missing");
}

/** Stairs and slabs are solid stuff filling part of a block: see-through, but light doesn't pass. */
const fillsPart = (shape: number) => shape === Shape.SLAB || isStairs(shape);

/** Turns the server's block list into the mesher's per-ID lookup arrays. */
export function buildRenderTable(
  types: readonly BlockTypeInfo[],
  layerOf: (texture: string) => number = builtinLayer,
): RenderTable {
  const n = Math.max(0, ...types.map((t) => t.id)) + 1;
  const table: RenderTable = {
    layers: new Int16Array(n * 6).fill(-1),
    transparent: new Uint8Array(n).fill(1),
    passes: new Uint8Array(n).fill(1),
    translucent: new Uint8Array(n),
    shape: new Uint8Array(n),
    light: new Uint8Array(n),
    smooth: new Uint8Array(n),
    wiring: new Uint8Array(n),
  };
  for (const t of types) {
    table.transparent[t.id] = t.transparent ? 1 : 0;
    table.shape[t.id] = t.shape ?? Shape.CUBE;
    table.passes[t.id] = t.transparent && !fillsPart(table.shape[t.id]) ? 1 : 0;
    table.light[t.id] = t.light ?? 0;
    table.wiring![t.id] = wiringOf(t);
    if (t.name === "air") continue;
    const style = styleOf(t);
    table.translucent[t.id] = style.translucent ? 1 : 0;
    table.smooth[t.id] = smoothKind(style, table.shape[t.id]);
    // Face order: +x, -x, +y, -y, +z, -z.
    const faces = [style.side, style.side, style.top, style.bottom, style.side, style.side];
    if (style.front) faces[frontFace(t.name)] = style.front;
    const missing = layerOf("missing");
    faces.forEach((name, f) => {
      const layer = layerOf(name);
      table.layers[t.id * 6 + f] = layer < 0 ? missing : layer;
    });
  }
  return table;
}

const WIRED =
  /^(wire|switch|button|pressure_plate|signal_lamp|timer|script_block|door|piston|sticky_piston|trapdoor|fence_gate|launch_pad|number_display|tripwire|diode|gate|delay|relay|capacitor|toggle|player_detector|creature_detector|clock|block_detector|weather_sensor|transmitter|receiver|counter|note_block|teleporter)(_|$)|_cable(_on)?$/;

/** How a block meets drawn wire: wire joins up to circuit blocks and lies on solid ones. */
function wiringOf(t: BlockTypeInfo): number {
  const wire = /^wire(_on)?$/.test(t.name) ? Wiring.WIRE : 0;
  const joins = WIRED.test(t.name) ? Wiring.JOINS : 0;
  return wire | joins | (t.solid ? Wiring.HOLDS : 0);
}

/** Natural cubes soften; water and plants leave room for them; anything else is built and pins them. */
function smoothKind(style: BlockStyle, shape: number): number {
  if (style.natural && shape === Shape.CUBE) return Smooth.NATURAL;
  if (style.translucent || shape === Shape.TUFT || shape === Shape.FLOWER) return Smooth.EMPTY;
  return Smooth.BUILT;
}

/** The face a turned block like a piston looks out of, from its name's facing suffix. */
function frontFace(name: string): number {
  const facing = name.replace(/_on$/, "").match(/_(east|south|west|up|down)$/)?.[0] ?? "";
  return FACINGS.find(([f]) => f === facing)![1];
}
