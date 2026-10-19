import { describe, expect, it } from "vitest";
import { collisionBoxes, FULL_BLOCK, heldBlock, Shape, stairsFacing } from "./shapes";

describe("collisionBoxes", () => {
  it("gives slabs their lower half and non-solid blocks nothing", () => {
    expect(collisionBoxes(Shape.SLAB, true)).toEqual([[0, 0, 0, 1, 0.5, 1]]);
    expect(collisionBoxes(Shape.TORCH, false)).toEqual([]);
    expect(collisionBoxes(99, true)).toBe(FULL_BLOCK);
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

describe("heldBlock", () => {
  it("turns any stairs facing back into the north-facing block", () => {
    expect(heldBlock(43, Shape.STAIRS_WEST)).toBe(40);
    expect(heldBlock(7, Shape.SLAB)).toBe(7);
  });
});
