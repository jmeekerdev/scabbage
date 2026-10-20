import { describe, expect, it } from "vitest";
import { collisionBoxes, FULL_BLOCK, pistonFacing, Shape, stairsFacing } from "./shapes";

describe("collisionBoxes", () => {
  it("gives slabs their lower half and non-solid blocks nothing", () => {
    expect(collisionBoxes(Shape.SLAB, true)).toEqual([[0, 0, 0, 1, 0.5, 1]]);
    expect(collisionBoxes(Shape.TORCH, false)).toEqual([]);
    expect(collisionBoxes(99, true)).toBe(FULL_BLOCK);
    expect(collisionBoxes(Shape.FLAT, true)).toEqual([[0, 0, 0, 1, 0.125, 1]]);
  });

  it("puts the tall half of north stairs toward -z", () => {
    expect(collisionBoxes(Shape.STAIRS_NORTH, true)[1]).toEqual([0, 0.5, 0, 1, 1, 0.5]);
  });
});

describe("stairsFacing", () => {
  it("rises away from the player", () => {
    expect(stairsFacing(40, 0)).toBe(40); // looking north
    expect(stairsFacing(40, Math.PI / 2)).toBe(43); // looking west
    expect(stairsFacing(40, Math.PI)).toBe(42);
    expect(stairsFacing(40, -Math.PI / 2)).toBe(41); // looking east
    expect(stairsFacing(40, 2 * Math.PI + 0.3)).toBe(40);
  });
});

describe("pistonFacing", () => {
  it("pushes where the player looks, each facing followed by its powered state", () => {
    expect(pistonFacing(60, 0, 0)).toBe(60); // north
    expect(pistonFacing(60, -Math.PI / 2, 0)).toBe(62); // east
    expect(pistonFacing(60, Math.PI, 0.5)).toBe(64); // south
    expect(pistonFacing(60, Math.PI / 2, 0)).toBe(66); // west
    expect(pistonFacing(60, 0, 1.2)).toBe(68); // up
    expect(pistonFacing(60, 2, -1.2)).toBe(70); // down
  });
});
