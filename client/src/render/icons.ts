import { BLOCK_STYLES } from "./blocks";
import { paintTextures, TEXTURE_NAMES, TEXTURE_SIZE } from "./textures";

/** Block icons from their side textures, each drawn once; an icon can be shown in one place. */
export function blockIconPainter(): (name: string) => HTMLCanvasElement {
  const pixels = paintTextures();
  const icons = new Map<string, HTMLCanvasElement>();
  return (name) => {
    let icon = icons.get(name);
    if (!icon) icons.set(name, (icon = paintIcon(pixels, name)));
    return icon;
  };
}

function paintIcon(pixels: Uint8Array, name: string): HTMLCanvasElement {
  const layer = Math.max(0, TEXTURE_NAMES.indexOf(BLOCK_STYLES[name]?.side ?? "missing"));
  const row = TEXTURE_SIZE * 4;
  const img = new ImageData(TEXTURE_SIZE, TEXTURE_SIZE);
  // Texture rows run bottom-up; canvas rows run top-down.
  for (let y = 0; y < TEXTURE_SIZE; y++) {
    const from = (layer * TEXTURE_SIZE + TEXTURE_SIZE - 1 - y) * row;
    img.data.set(pixels.subarray(from, from + row), y * row);
  }
  const canvas = document.createElement("canvas");
  canvas.width = canvas.height = TEXTURE_SIZE;
  canvas.getContext("2d")?.putImageData(img, 0, 0);
  return canvas;
}
