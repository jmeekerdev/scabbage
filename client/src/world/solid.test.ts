import { describe, expect, it } from "vitest";
import { solidQuery } from "./solid";
import { collisionBoxes, FULL_BLOCK, Shape } from "./shapes";

const STONE = 1;
const WATER = 2;
const SLAB = 3;
const boxes = [[], FULL_BLOCK, [], collisionBoxes(Shape.SLAB, true)];
const blocks: Record<string, number> = {
  "0,5,0": STONE,
  "1,5,0": WATER,
  "2,5,0": 0,
  "4,5,0": SLAB,
};
const blockAt = (x: number, y: number, z: number) => blocks[`${x},${y},${z}`];
const solid = solidQuery({ radius: 64, height: 32 }, boxes, blockAt);

describe("solidQuery", () => {
  it("follows the block table", () => {
    expect(solid(0, 5, 0)).toBe(FULL_BLOCK);
    expect(solid(1, 5, 0)).toEqual([]);
    expect(solid(2, 5, 0)).toEqual([]);
    expect(solid(4, 5, 0)).toEqual([[0, 0, 0, 1, 0.5, 1]]);
  });

  it("walls off unloaded chunks, the border and the floor but not the sky", () => {
    expect(solid(3, 5, 0)).toBe(FULL_BLOCK);
    expect(solid(64, 5, 0)).toBe(FULL_BLOCK);
    expect(solid(-65, 5, 0)).toBe(FULL_BLOCK);
    expect(solid(0, -1, 0)).toBe(FULL_BLOCK);
    expect(solid(0, 32, 0)).toEqual([]);
  });

  it("treats unknown block ids as passable", () => {
    const q = solidQuery({ radius: -1, height: 32 }, boxes, () => 99);
    expect(q(1000, 5, 0)).toEqual([]);
  });
});
