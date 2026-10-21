import type { RenderTable } from "../world/mesher";
import { customLayerName } from "./atlas";
import { Shape } from "../world/shapes";
import { Smooth } from "../world/soft-corners";
import { COUNT_DOWN } from "./machine-textures";
import { TEXTURE_NAMES, WOOL_COLORS } from "./textures";

export interface BlockStyle {
  top: string;
  side: string;
  bottom: string;
  front?: string; // the face a piston pushes from
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
const piston = (front: string): BlockStyle => ({ ...same("piston_side"), front });
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
    ["wire", "switch", "button", "pressure_plate", "signal_lamp", "timer", "script_block"].flatMap(
      (b) => [
        [b, same(b)],
        [`${b}_on`, same(`${b}_on`)],
      ],
    ),
  ),
  door: same("door"),
  door_open: same("door_open"),
  ...Object.fromEntries(
    FACINGS.flatMap(([f]) => [
      [`piston${f}`, piston("piston_front")],
      [`piston${f}_on`, piston("piston_front_on")],
    ]),
  ),
  ...Object.fromEntries(numbered("counter", "counter", 10)),
  ...Object.fromEntries(
    COUNT_DOWN.flatMap((d) => [
      [`counter_down_${d}`, same(`counter_down_${d}`)],
      [`counter_down_${d}_on`, same(`counter_down_${d}`)],
    ]),
  ),
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

/** Turns the server's block list into the mesher's per-ID lookup arrays. */
export function buildRenderTable(
  types: readonly BlockTypeInfo[],
  layerOf: (texture: string) => number = builtinLayer,
): RenderTable {
  const n = Math.max(0, ...types.map((t) => t.id)) + 1;
  const table: RenderTable = {
    layers: new Int16Array(n * 6).fill(-1),
    transparent: new Uint8Array(n).fill(1),
    translucent: new Uint8Array(n),
    shape: new Uint8Array(n),
    light: new Uint8Array(n),
    smooth: new Uint8Array(n),
  };
  for (const t of types) {
    table.transparent[t.id] = t.transparent ? 1 : 0;
    table.shape[t.id] = t.shape ?? Shape.CUBE;
    table.light[t.id] = t.light ?? 0;
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
