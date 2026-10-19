import type { RenderTable } from "../world/mesher";
import { flipRows, type Atlas } from "./atlas";
import { TEXTURE_SIZE } from "./textures";

/** Block icons from their side textures, each drawn once; an icon can be shown in one place. */
export function blockIconPainter(
  atlas: Atlas,
  table: RenderTable,
): (block: number) => HTMLCanvasElement {
  const icons = new Map<number, HTMLCanvasElement>();
  return (block) => {
    let icon = icons.get(block);
    if (!icon) icons.set(block, (icon = paintIcon(atlas, table.layers[block * 6] ?? -1)));
    return icon;
  };
}

/** One layer drawn onto a canvas; texture rows run bottom-up, canvas rows top-down. */
export function paintIcon(atlas: Atlas, layer: number): HTMLCanvasElement {
  const bytes = TEXTURE_SIZE * TEXTURE_SIZE * 4;
  const from = Math.max(0, layer) * bytes;
  return paintPixels(
    flipRows(atlas.pixels.subarray(from, from + bytes)),
    TEXTURE_SIZE,
    TEXTURE_SIZE,
  );
}

/** A canvas showing RGBA pixels (rows top-down) at their natural size. */
export function paintPixels(pixels: Uint8Array, w: number, h: number): HTMLCanvasElement {
  const canvas = document.createElement("canvas");
  canvas.width = w;
  canvas.height = h;
  const img = new ImageData(w, h);
  img.data.set(pixels.subarray(0, w * h * 4));
  canvas.getContext("2d")?.putImageData(img, 0, 0);
  return canvas;
}
