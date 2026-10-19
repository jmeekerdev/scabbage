import type { RenderTable } from "../world/mesher";
import { customLayerName } from "./atlas";
import { TEXTURE_NAMES } from "./textures";

export interface BlockStyle {
  top: string;
  side: string;
  bottom: string;
  translucent?: boolean;
}

const same = (t: string): BlockStyle => ({ top: t, side: t, bottom: t });

/** How each server block name looks; unknown names get the "missing" texture. */
export const BLOCK_STYLES: Record<string, BlockStyle> = {
  stone: same("stone"),
  dirt: same("dirt"),
  grass: { top: "grass_top", side: "grass_side", bottom: "dirt" },
  sand: same("sand"),
  water: { ...same("water"), translucent: true },
  log: { top: "log_top", side: "log_side", bottom: "log_top" },
  leaves: same("leaves"),
  planks: same("planks"),
  cobblestone: same("cobblestone"),
  glass: same("glass"),
  brick: same("brick"),
  bedrock: same("bedrock"),
};

export interface BlockTypeInfo {
  id: number;
  name: string;
  transparent: boolean;
  pixels?: Uint8Array; // custom blocks only
}

const builtinLayer = (name: string) => TEXTURE_NAMES.indexOf(name);

/** How a block looks: custom blocks use their own picture on every face. */
export function styleOf(t: BlockTypeInfo): BlockStyle {
  if (t.pixels?.length) return same(customLayerName(t.id));
  return BLOCK_STYLES[t.name] ?? same("missing");
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
  };
  for (const t of types) {
    table.transparent[t.id] = t.transparent ? 1 : 0;
    if (t.name === "air") continue;
    const style = styleOf(t);
    table.translucent[t.id] = style.translucent ? 1 : 0;
    // Face order: +x, -x, +y, -y, +z, -z.
    const faces = [style.side, style.side, style.top, style.bottom, style.side, style.side];
    const missing = layerOf("missing");
    faces.forEach((name, f) => {
      const layer = layerOf(name);
      table.layers[t.id * 6 + f] = layer < 0 ? missing : layer;
    });
  }
  return table;
}
