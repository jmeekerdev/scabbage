import type { RenderTable } from "../world/mesher";
import { paintTextures, TEXTURE_NAMES, TEXTURE_SIZE } from "./textures";

const LAYER_BYTES = TEXTURE_SIZE * TEXTURE_SIZE * 4;

export interface ArtBlock {
  id: number;
  pixels?: Uint8Array; // custom blocks only: 16×16 RGBA, rows top-down
}

export interface RepaintedTexture {
  name: string;
  pixels: Uint8Array; // rows top-down
}

/** Every block texture as layers of one array texture; rows run bottom-up (v = 0 first). */
export interface Atlas {
  pixels: Uint8Array;
  depth: number;
  layerOf(name: string): number; // -1 if unknown
  /** A layer's pixels with rows top-down, as the pixel editor shows them. */
  read(name: string): Uint8Array | undefined;
}

export const customLayerName = (id: number) => `custom:${id}`;

/** Built-in textures (with any admin repaints), then one layer per custom block in ID order. */
export function buildAtlas(
  blocks: readonly ArtBlock[],
  textures: readonly RepaintedTexture[],
): Atlas {
  const customs = blocks
    .filter((b) => b.pixels?.length === LAYER_BYTES)
    .sort((a, b) => a.id - b.id);
  const names = [...TEXTURE_NAMES, ...customs.map((b) => customLayerName(b.id))];
  const pixels = new Uint8Array(names.length * LAYER_BYTES);
  pixels.set(paintTextures());
  const layerOf = (name: string) => names.indexOf(name);
  const write = (layer: number, topDown: Uint8Array) =>
    pixels.set(flipRows(topDown), layer * LAYER_BYTES);
  for (const t of textures) {
    const layer = TEXTURE_NAMES.indexOf(t.name);
    if (layer >= 0 && t.pixels.length === LAYER_BYTES) write(layer, t.pixels);
  }
  customs.forEach((b, i) => write(TEXTURE_NAMES.length + i, b.pixels as Uint8Array));
  return {
    pixels,
    depth: names.length,
    layerOf,
    read(name) {
      const layer = layerOf(name);
      if (layer < 0) return undefined;
      return flipRows(pixels.subarray(layer * LAYER_BYTES, (layer + 1) * LAYER_BYTES));
    },
  };
}

/** Swaps a 16×16 layer between top-down and bottom-up row order. */
export function flipRows(layer: Uint8Array): Uint8Array {
  const row = TEXTURE_SIZE * 4;
  const out = new Uint8Array(LAYER_BYTES);
  for (let y = 0; y < TEXTURE_SIZE; y++)
    out.set(layer.subarray(y * row, (y + 1) * row), (TEXTURE_SIZE - 1 - y) * row);
  return out;
}

/** Each block's top face as one sRGB color, RGB per block ID, from its picture's solid pixels. */
export function topColors(atlas: Atlas, table: RenderTable): Uint8Array {
  const n = table.shape.length;
  const out = new Uint8Array(n * 3);
  for (let id = 0; id < n; id++) {
    const layer = table.layers[id * 6 + 2];
    if (layer >= 0) out.set(averageColor(atlas.pixels, layer), id * 3);
  }
  return out;
}

function averageColor(pixels: Uint8Array, layer: number): number[] {
  const sum = [0, 0, 0];
  let count = 0;
  for (let i = layer * LAYER_BYTES; i < (layer + 1) * LAYER_BYTES; i += 4) {
    if (pixels[i + 3] < 128) continue;
    for (let c = 0; c < 3; c++) sum[c] += pixels[i + c];
    count++;
  }
  return sum.map((v) => Math.round(v / Math.max(1, count)));
}
