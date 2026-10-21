import { DataTexture, NearestFilter, RGBAFormat, SRGBColorSpace } from "three";
import type { Rgb } from "../looks/look";
import type { RenderTable } from "../world/mesher";
import { isCustomLayer, type Atlas } from "./atlas";
import { TEXTURE_SIZE } from "./textures";

/** How an untextured look draws a layer, kept in the alpha of its colour. */
export const SOLID = 0;
export const CUT_OUT = 128; // flat colour, but the picture's see-through pixels still cut it
export const STICKER = 255; // the picture itself, as custom blocks are drawn

const LAYER_BYTES = TEXTURE_SIZE * TEXTURE_SIZE * 4;

/**
 * RGBA per atlas layer: its flat sRGB colour, from the palette or the picture's average, and how to
 * draw it. Pictures named in solid lose their see-through holes.
 */
export function layerColors(
  atlas: Atlas,
  palette: Record<string, Rgb> = {},
  solid: readonly string[] = [],
): Uint8Array {
  const out = new Uint8Array(atlas.depth * 4);
  atlas.names.forEach((name, layer) => {
    const pixels = atlas.pixels.subarray(layer * LAYER_BYTES, (layer + 1) * LAYER_BYTES);
    const chosen = palette[name];
    const rgb = chosen
      ? [chosen.r, chosen.g, chosen.b].map((v) => Math.round(v * 255))
      : averageColor(pixels);
    out.set(rgb, layer * 4);
    out[layer * 4 + 3] = isCustomLayer(name)
      ? STICKER
      : hasHoles(pixels) && !solid.includes(name)
        ? CUT_OUT
        : SOLID;
  });
  return out;
}

/** layerColors as a one-row texture the block shader looks layers up in. */
export function layerColorTexture(colors: Uint8Array): DataTexture {
  const tex = new DataTexture(colors, colors.length / 4, 1, RGBAFormat);
  tex.colorSpace = SRGBColorSpace;
  tex.magFilter = tex.minFilter = NearestFilter;
  tex.needsUpdate = true;
  return tex;
}

/** Each block's top face as one sRGB colour, RGB per block ID, from layerColors. */
export function topColors(colors: Uint8Array, table: RenderTable): Uint8Array {
  const n = table.shape.length;
  const out = new Uint8Array(n * 3);
  for (let id = 0; id < n; id++) {
    const layer = table.layers[id * 6 + 2];
    if (layer >= 0) out.set(colors.subarray(layer * 4, layer * 4 + 3), id * 3);
  }
  return out;
}

function averageColor(pixels: Uint8Array): number[] {
  const sum = [0, 0, 0];
  let count = 0;
  for (let i = 0; i < pixels.length; i += 4) {
    if (pixels[i + 3] < 128) continue;
    for (let c = 0; c < 3; c++) sum[c] += pixels[i + c];
    count++;
  }
  return sum.map((v) => Math.round(v / Math.max(1, count)));
}

function hasHoles(pixels: Uint8Array): boolean {
  for (let i = 3; i < pixels.length; i += 4) if (pixels[i] < 128) return true;
  return false;
}
