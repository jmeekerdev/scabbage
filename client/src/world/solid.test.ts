import { describe, expect, it } from "vitest";
import { solidQuery } from "./solid";

const STONE = 1;
const WATER = 2;
const solidTable = [false, true, false];
const blocks: Record<string, number> = { "0,5,0": STONE, "1,5,0": WATER, "2,5,0": 0 };
const blockAt = (x: number, y: number, z: number) => blocks[`${x},${y},${z}`];
const solid = solidQuery({ radius: 64, height: 32 }, solidTable, blockAt);

describe("solidQuery", () => {
  it("follows the block table", () => {
    expect(solid(0, 5, 0)).toBe(true);
    expect(solid(1, 5, 0)).toBe(false);
    expect(solid(2, 5, 0)).toBe(false);
  });

  it("walls off unloaded chunks, the border and the floor but not the sky", () => {
    expect(solid(3, 5, 0)).toBe(true);
    expect(solid(64, 5, 0)).toBe(true);
    expect(solid(-65, 5, 0)).toBe(true);
    expect(solid(0, -1, 0)).toBe(true);
    expect(solid(0, 32, 0)).toBe(false);
  });

  it("treats unknown block ids as passable", () => {
    const q = solidQuery({ radius: -1, height: 32 }, solidTable, () => 99);
    expect(q(1000, 5, 0)).toBe(false);
  });
});
