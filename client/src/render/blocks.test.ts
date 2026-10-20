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
  { id: 8, name: "piston_east_on", transparent: false },
  { id: 9, name: "piston", transparent: false },
  { id: 10, name: "counter_3_on", transparent: false },
  { id: 11, name: "note_block_2", transparent: false },
  { id: 12, name: "counter", transparent: false },
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
    expect([...t.transparent]).toEqual([1, 0, 0, 1, 0, 1, 1, 0, 0, 0, 0, 0, 0]);
  });

  it("blends only water", () => {
    expect([...t.translucent]).toEqual([0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0]);
  });

  it("dresses slabs and stairs in their material and keeps shapes and glow", () => {
    expect(t.layers[5 * 6 + 2]).toBe(layer("brick"));
    expect(t.shape[5]).toBe(Shape.STAIRS_EAST);
    expect(t.light[6]).toBe(11);
    expect(t.layers[6 * 6]).toBe(layer("torch"));
    expect(t.layers[7 * 6]).toBe(layer("red_wool"));
  });

  it("turns a piston's face the way it pushes", () => {
    const faces = (id: number) => [...t.layers.slice(id * 6, id * 6 + 6)];
    const [side, front] = [layer("piston_side"), layer("piston_front")];
    expect(faces(8)).toEqual([layer("piston_front_on"), side, side, side, side, side]);
    expect(faces(9)).toEqual([side, side, side, side, side, front]);
  });

  it("shows a counter's count and a note block's pitch on every face", () => {
    expect(t.layers[10 * 6 + 2]).toBe(layer("counter_3"));
    expect(t.layers[12 * 6 + 4]).toBe(layer("counter_0"));
    expect(t.layers[11 * 6]).toBe(layer("note_2"));
  });

  it("lights a counter's digits while it counts back down", () => {
    expect(BLOCK_STYLES.counter_down_3_on?.side).toBe("counter_down_3");
    expect(layer("counter_down_3")).toBeGreaterThan(layer("counter_9"));
  });

  it("shows unknown blocks with a placeholder texture instead of failing", () => {
    expect(t.layers[4 * 6]).toBe(layer("missing"));
  });
});

describe("textures", () => {
  it("paints every texture the block styles use", () => {
    for (const s of Object.values(BLOCK_STYLES)) {
      for (const name of [s.top, s.side, s.bottom, s.front ?? s.side]) {
        expect(TEXTURE_NAMES).toContain(name);
      }
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
