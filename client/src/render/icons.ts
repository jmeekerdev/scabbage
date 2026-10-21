import type { RenderTable } from "../world/mesher";
import { Shape, sideView } from "../world/shapes";
import { flipRows, type Atlas } from "./atlas";
import { TEXTURE_SIZE } from "./textures";

/** Block icons from their side textures, each painted once and handed out as fresh copies. */
export function blockIconPainter(
  atlas: Atlas,
  table: RenderTable,
): (block: number) => HTMLCanvasElement {
  const icons = new Map<number, HTMLCanvasElement>();
  return (block) => {
    let icon = icons.get(block);
    if (!icon) {
      icon = paintIcon(atlas, table.layers[block * 6] ?? -1, table.shape[block] ?? Shape.CUBE);
      icons.set(block, icon);
    }
    return copyCanvas(icon);
  };
}

function copyCanvas(from: HTMLCanvasElement): HTMLCanvasElement {
  const canvas = document.createElement("canvas");
  canvas.width = from.width;
  canvas.height = from.height;
  canvas.getContext("2d")?.drawImage(from, 0, 0);
  return canvas;
}

/** One layer cut to the shape's side view; texture rows run bottom-up, canvas rows top-down. */
export function paintIcon(atlas: Atlas, layer: number, shape: number): HTMLCanvasElement {
  const bytes = TEXTURE_SIZE * TEXTURE_SIZE * 4;
  const from = Math.max(0, layer) * bytes;
  const pixels = flipRows(atlas.pixels.subarray(from, from + bytes));
  const shown = sideView(shape);
  for (let y = 0; y < TEXTURE_SIZE; y++) {
    for (let x = 0; x < TEXTURE_SIZE; x++)
      if (!shown(x, y)) pixels[(y * TEXTURE_SIZE + x) * 4 + 3] = 0;
  }
  return paintPixels(pixels, TEXTURE_SIZE, TEXTURE_SIZE);
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
