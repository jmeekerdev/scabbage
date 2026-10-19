import { describe, expect, it } from "vitest";
import { Shape } from "../world/shapes";
import { buildRenderTable, BLOCK_STYLES } from "./blocks";
import { TEXTURE_NAMES, paintTextures, TEXTURE_SIZE } from "./textures";

// A slice of the server's block table (server/internal/world/block.go).
const types = [
  { id: 0, name: "air", transparent: true },
  { id: 1, name: "stone", transparent: false },
  { id: 2, name: "grass", transparent: false },
  { id: 3, name: "water", transparent: true },
  { id: 4, name: "mystery", transparent: false },
  { id: 5, name: "brick_stairs_east", transparent: true, shape: Shape.STAIRS_EAST },
  { id: 6, name: "torch", transparent: true, shape: Shape.TORCH, light: 11 },
  { id: 7, name: "red_wool", transparent: false },
];
const layer = (name: string) => TEXTURE_NAMES.indexOf(name);

describe("buildRenderTable", () => {
  const t = buildRenderTable(types);

  it("gives grass a green top, dirt bottom and grassy sides", () => {
    expect(t.layers[2 * 6 + 2]).toBe(layer("grass_top"));
    expect(t.layers[2 * 6 + 3]).toBe(layer("dirt"));
    expect(t.layers[2 * 6 + 0]).toBe(layer("grass_side"));
  });

  it("never draws air and marks the server's see-through blocks", () => {
    expect(t.layers[0]).toBe(-1);
    expect([...t.transparent]).toEqual([1, 0, 0, 1, 0, 1, 1, 0]);
  });

  it("blends only water", () => {
    expect([...t.translucent]).toEqual([0, 0, 0, 1, 0, 0, 0, 0]);
  });

  it("dresses slabs and stairs in their material and keeps shapes and glow", () => {
    expect(t.layers[5 * 6 + 2]).toBe(layer("brick"));
    expect(t.shape[5]).toBe(Shape.STAIRS_EAST);
    expect(t.light[6]).toBe(11);
    expect(t.layers[6 * 6]).toBe(layer("torch"));
    expect(t.layers[7 * 6]).toBe(layer("red_wool"));
  });

  it("shows unknown blocks with a placeholder texture instead of failing", () => {
    expect(t.layers[4 * 6]).toBe(layer("missing"));
  });
});

describe("textures", () => {
  it("paints every texture the block styles use", () => {
    for (const s of Object.values(BLOCK_STYLES)) {
      for (const name of [s.top, s.side, s.bottom]) expect(TEXTURE_NAMES).toContain(name);
    }
  });

  it("paints opaque pixels deterministically", () => {
    const a = paintTextures();
    expect(a.length).toBe(TEXTURE_NAMES.length * TEXTURE_SIZE * TEXTURE_SIZE * 4);
    expect(paintTextures()).toEqual(a);
    const stone = layer("stone") * TEXTURE_SIZE * TEXTURE_SIZE * 4;
    expect(a[stone + 3]).toBe(255);
  });
});
