import { describe, expect, it } from "vitest";
import { buildAtlas, customLayerName, topColors } from "./atlas";
import { buildRenderTable } from "./blocks";
import { paintTextures, TEXTURE_NAMES, TEXTURE_SIZE } from "./textures";

const LAYER = TEXTURE_SIZE * TEXTURE_SIZE * 4;

/** A 16×16 top-down picture whose top row is red and the rest blue. */
function topRed(): Uint8Array {
  const px = new Uint8Array(LAYER);
  for (let i = 0; i < LAYER; i += 4)
    px.set(i < TEXTURE_SIZE * 4 ? [255, 0, 0, 255] : [0, 0, 255, 255], i);
  return px;
}

const custom = { id: 256, name: "flower", transparent: false, pixels: topRed() };

describe("buildAtlas", () => {
  it("keeps built-in textures first, unchanged", () => {
    const atlas = buildAtlas([], []);
    expect(atlas.depth).toBe(TEXTURE_NAMES.length);
    expect(atlas.pixels).toEqual(paintTextures());
    expect(atlas.layerOf("stone")).toBe(TEXTURE_NAMES.indexOf("stone"));
  });

  it("adds a layer per custom block, flipped so its top row is drawn at the top", () => {
    const atlas = buildAtlas([custom], []);
    const layer = atlas.layerOf(customLayerName(256));
    expect(layer).toBe(TEXTURE_NAMES.length);
    const lastRow = layer * LAYER + (TEXTURE_SIZE - 1) * TEXTURE_SIZE * 4;
    expect([...atlas.pixels.subarray(lastRow, lastRow + 4)]).toEqual([255, 0, 0, 255]);
  });

  it("repaints a built-in texture in place", () => {
    const atlas = buildAtlas([], [{ name: "brick", pixels: topRed() }]);
    expect(atlas.depth).toBe(TEXTURE_NAMES.length);
    expect(atlas.read("brick")).toEqual(topRed());
  });

  it("skips art of the wrong size or for textures it doesn't know", () => {
    const atlas = buildAtlas(
      [{ ...custom, pixels: new Uint8Array(4) }],
      [{ name: "nope", pixels: topRed() }],
    );
    expect(atlas.depth).toBe(TEXTURE_NAMES.length);
    expect(atlas.layerOf(customLayerName(256))).toBe(-1);
  });
});

describe("buildRenderTable with custom blocks", () => {
  it("draws every face of a custom block with its own picture", () => {
    const atlas = buildAtlas([custom], []);
    const table = buildRenderTable(
      [{ id: 0, name: "air", transparent: true }, custom],
      atlas.layerOf,
    );
    const own = atlas.layerOf(customLayerName(256));
    expect([...table.layers.subarray(256 * 6, 257 * 6)]).toEqual(Array(6).fill(own));
  });
});

describe("topColors", () => {
  const air = { id: 0, name: "air", transparent: true };
  const colorsOf = (pixels: Uint8Array) => {
    const atlas = buildAtlas([{ ...custom, pixels }], []);
    const table = buildRenderTable([air, { ...custom, pixels }], atlas.layerOf);
    return [...topColors(atlas, table).subarray(256 * 3, 257 * 3)];
  };

  it("averages each block's top picture", () => {
    expect(colorsOf(topRed())).toEqual([16, 0, 239]);
  });

  it("leaves see-through pixels out", () => {
    const px = topRed();
    for (let i = TEXTURE_SIZE * 4 + 3; i < LAYER; i += 4) px[i] = 0;
    expect(colorsOf(px)).toEqual([255, 0, 0]);
  });
});
