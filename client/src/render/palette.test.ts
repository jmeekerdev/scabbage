import { describe, expect, it } from "vitest";
import { buildAtlas, customLayerName } from "./atlas";
import { buildRenderTable } from "./blocks";
import { CUT_OUT, layerColors, SOLID, STICKER, topColors } from "./palette";
import { TEXTURE_SIZE } from "./textures";

const LAYER = TEXTURE_SIZE * TEXTURE_SIZE * 4;

/** A 16×16 top-down picture whose top row is red and the rest blue. */
function topRed(): Uint8Array {
  const px = new Uint8Array(LAYER);
  for (let i = 0; i < LAYER; i += 4)
    px.set(i < TEXTURE_SIZE * 4 ? [255, 0, 0, 255] : [0, 0, 255, 255], i);
  return px;
}

const custom = { id: 256, name: "flower", transparent: false, pixels: topRed() };
const air = { id: 0, name: "air", transparent: true };

/** The RGBA layerColors gives the layer of a built-in texture, painted with `pixels`. */
function colorOf(pixels: Uint8Array, palette?: Parameters<typeof layerColors>[1]): number[] {
  const atlas = buildAtlas([], [{ name: "brick", pixels }]);
  const layer = atlas.layerOf("brick");
  return [...layerColors(atlas, palette).subarray(layer * 4, layer * 4 + 4)];
}

describe("layerColors", () => {
  it("averages each picture's solid pixels", () => {
    expect(colorOf(topRed())).toEqual([16, 0, 239, SOLID]);
  });

  it("leaves see-through pixels out, and marks the picture as cut out", () => {
    const px = topRed();
    for (let i = TEXTURE_SIZE * 4 + 3; i < LAYER; i += 4) px[i] = 0;
    expect(colorOf(px)).toEqual([255, 0, 0, CUT_OUT]);
  });

  it("takes a look's palette over the average, by texture name", () => {
    expect(colorOf(topRed(), { brick: { r: 1, g: 0.5, b: 0 } })).toEqual([255, 128, 0, SOLID]);
  });

  it("keeps custom blocks' pictures as stickers", () => {
    const atlas = buildAtlas([custom], []);
    const layer = atlas.layerOf(customLayerName(256));
    expect(layerColors(atlas)[layer * 4 + 3]).toBe(STICKER);
  });
});

describe("topColors", () => {
  it("gives each block its top layer's colour", () => {
    const atlas = buildAtlas([custom], []);
    const table = buildRenderTable([air, custom], atlas.layerOf);
    const colors = topColors(layerColors(atlas), table);
    expect([...colors.subarray(256 * 3, 257 * 3)]).toEqual([16, 0, 239]);
  });
});
